package security

import (
	"testing"

	"open-dan/internal/config"
)

func TestSanitizeEmail(t *testing.T) {
	s := NewSanitizer(config.PIIFilterConfig{
		Enabled:      true,
		FilterEmails: true,
	})

	input := "My email is john@example.com and also jane@test.org"
	result := s.Sanitize(input)

	if result == input {
		t.Fatal("expected sanitization to change the input")
	}
	if indexOf(result, "john@example.com") >= 0 {
		t.Fatal("email was not sanitized")
	}
	if indexOf(result, "[EMAIL_") < 0 {
		t.Fatalf("expected EMAIL placeholder, got: %s", result)
	}
}

func TestSanitizePhone(t *testing.T) {
	s := NewSanitizer(config.PIIFilterConfig{
		Enabled:      true,
		FilterPhones: true,
	})

	input := "Call me at +1-555-123-4567"
	result := s.Sanitize(input)

	if indexOf(result, "555-123-4567") >= 0 {
		t.Fatal("phone was not sanitized")
	}
}

func TestSanitizeDisabled(t *testing.T) {
	s := NewSanitizer(config.PIIFilterConfig{
		Enabled: false,
	})

	input := "john@example.com 555-123-4567"
	result := s.Sanitize(input)

	if result != input {
		t.Fatal("disabled sanitizer should not modify input")
	}
}

func TestRestorePlaceholders(t *testing.T) {
	s := NewSanitizer(config.PIIFilterConfig{
		Enabled:      true,
		FilterEmails: true,
	})

	input := "Contact john@example.com for info"
	sanitized := s.Sanitize(input)
	restored := s.Restore(sanitized)

	if restored != input {
		t.Fatalf("restore failed: expected %q, got %q", input, restored)
	}
}

func TestSanitizeCards(t *testing.T) {
	s := NewSanitizer(config.PIIFilterConfig{
		Enabled:     true,
		FilterCards: true,
	})

	input := "My card is 4111-1111-1111-1111"
	result := s.Sanitize(input)

	if indexOf(result, "4111") >= 0 {
		t.Fatal("card number was not sanitized")
	}
}
