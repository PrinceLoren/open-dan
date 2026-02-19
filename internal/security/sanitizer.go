package security

import (
	"fmt"
	"regexp"
	"sync"

	"open-dan/internal/config"
)

const maxPIIMappings = 1000

// Sanitizer replaces PII in text with placeholders.
type Sanitizer struct {
	mu       sync.RWMutex
	filters  []piiFilter
	mappings map[string]string // placeholder → original value
	counter  map[string]int
	enabled  bool
}

type piiFilter struct {
	name    string
	pattern *regexp.Regexp
	prefix  string
}

var defaultFilters = []struct {
	name    string
	pattern string
	prefix  string
}{
	{"email", `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`, "EMAIL"},
	{"phone", `(?:\+?\d{1,3}[-.\s]?)?\(?\d{2,4}\)?[-.\s]?\d{3,4}[-.\s]?\d{3,4}`, "PHONE"},
	{"card", `\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`, "CARD"},
	{"ip", `\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`, "IP"},
	{"ssn", `\b\d{3}-\d{2}-\d{4}\b`, "SSN"},
}

// NewSanitizer creates a PII sanitizer from config.
func NewSanitizer(cfg config.PIIFilterConfig) *Sanitizer {
	s := &Sanitizer{
		mappings: make(map[string]string),
		counter:  make(map[string]int),
		enabled:  cfg.Enabled,
	}

	enableMap := map[string]bool{
		"email": cfg.FilterEmails,
		"phone": cfg.FilterPhones,
		"card":  cfg.FilterCards,
		"ip":    cfg.FilterIPs,
		"ssn":   cfg.FilterSSN,
	}

	for _, f := range defaultFilters {
		if enableMap[f.name] {
			s.filters = append(s.filters, piiFilter{
				name:    f.name,
				pattern: regexp.MustCompile(f.pattern),
				prefix:  f.prefix,
			})
		}
	}

	return s
}

// Sanitize replaces PII in text with placeholders.
func (s *Sanitizer) Sanitize(text string) string {
	if !s.enabled || len(s.filters) == 0 {
		return text
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Evict old mappings if limit reached to prevent unbounded growth
	if len(s.mappings) >= maxPIIMappings {
		s.mappings = make(map[string]string)
		s.counter = make(map[string]int)
	}

	result := text
	for _, f := range s.filters {
		result = f.pattern.ReplaceAllStringFunc(result, func(match string) string {
			// Check if already mapped
			for placeholder, original := range s.mappings {
				if original == match {
					return placeholder
				}
			}
			s.counter[f.prefix]++
			placeholder := fmt.Sprintf("[%s_%d]", f.prefix, s.counter[f.prefix])
			s.mappings[placeholder] = match
			return placeholder
		})
	}
	return result
}

// Restore replaces placeholders back with original values.
func (s *Sanitizer) Restore(text string) string {
	if !s.enabled {
		return text
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := text
	for placeholder, original := range s.mappings {
		result = replaceAll(result, placeholder, original)
	}
	return result
}

// Reset clears all stored mappings (e.g., between conversations).
func (s *Sanitizer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mappings = make(map[string]string)
	s.counter = make(map[string]int)
}

func replaceAll(s, old, new string) string {
	if old == "" {
		return s
	}
	result := ""
	for {
		i := indexOf(s, old)
		if i < 0 {
			result += s
			break
		}
		result += s[:i] + new
		s = s[i+len(old):]
	}
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
