package tool

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"

	"open-dan/internal/config"
)

// BrowserTool provides browser automation via rod.
type BrowserTool struct {
	cfg     config.BrowserConfig
	mu      sync.Mutex
	browser *rod.Browser
	pages   map[string]*rod.Page
	nextID  int
}

// NewBrowserTool creates a new browser tool.
func NewBrowserTool(cfg config.BrowserConfig) *BrowserTool {
	if cfg.TimeoutSecs <= 0 {
		cfg.TimeoutSecs = 30
	}
	if cfg.MaxTabs <= 0 {
		cfg.MaxTabs = 3
	}
	if cfg.MaxPageSizeKB <= 0 {
		cfg.MaxPageSizeKB = 2048
	}
	return &BrowserTool{
		cfg:   cfg,
		pages: make(map[string]*rod.Page),
	}
}

func (t *BrowserTool) Name() string { return "browser" }
func (t *BrowserTool) Description() string {
	return "Control a web browser. Actions: navigate (open URL), get_content (page text), click (CSS selector), fill (type text into input), screenshot (capture page), eval_js (run JavaScript), get_links (list all links), close (close tab)."
}

func (t *BrowserTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {
				"type": "string",
				"enum": ["navigate", "get_content", "click", "fill", "screenshot", "eval_js", "get_links", "close"],
				"description": "The browser action to perform"
			},
			"url": {
				"type": "string",
				"description": "URL to navigate to (for navigate action)"
			},
			"page_id": {
				"type": "string",
				"description": "Page ID returned by navigate (for all actions except navigate)"
			},
			"selector": {
				"type": "string",
				"description": "CSS selector (for click and fill actions)"
			},
			"text": {
				"type": "string",
				"description": "Text to type (for fill action)"
			},
			"script": {
				"type": "string",
				"description": "JavaScript code to execute (for eval_js action)"
			}
		},
		"required": ["action"]
	}`)
}

type browserParams struct {
	Action   string `json:"action"`
	URL      string `json:"url"`
	PageID   string `json:"page_id"`
	Selector string `json:"selector"`
	Text     string `json:"text"`
	Script   string `json:"script"`
}

func (t *BrowserTool) Execute(ctx context.Context, args json.RawMessage) (*Result, error) {
	var params browserParams
	if err := json.Unmarshal(args, &params); err != nil {
		return &Result{Error: "invalid arguments: " + err.Error(), IsError: true}, nil
	}

	timeout := time.Duration(t.cfg.TimeoutSecs) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch params.Action {
	case "navigate":
		return t.navigate(ctx, params)
	case "get_content":
		return t.getContent(ctx, params)
	case "click":
		return t.click(ctx, params)
	case "fill":
		return t.fill(ctx, params)
	case "screenshot":
		return t.screenshot(ctx, params)
	case "eval_js":
		return t.evalJS(ctx, params)
	case "get_links":
		return t.getLinks(ctx, params)
	case "close":
		return t.closePage(params)
	default:
		return &Result{Error: fmt.Sprintf("unknown action: %s", params.Action), IsError: true}, nil
	}
}

func (t *BrowserTool) ensureBrowser() error {
	if t.browser != nil {
		return nil
	}

	l := launcher.New().Headless(t.cfg.Headless)
	controlURL, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}

	t.browser = browser
	return nil
}

// validateURL checks the URL scheme, private IPs, and domain allow/deny lists.
func (t *BrowserTool) validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https
	switch u.Scheme {
	case "http", "https":
	default:
		return fmt.Errorf("only http/https schemes are allowed, got: %s", u.Scheme)
	}

	host := u.Hostname()

	// Block private/loopback/link-local addresses (SSRF protection)
	if isPrivateHost(host) {
		return fmt.Errorf("access to private/loopback addresses is denied: %s", host)
	}

	// Domain allow/deny checks
	domain := strings.ToLower(host)

	for _, d := range t.cfg.DeniedDomains {
		dl := strings.ToLower(d)
		if dl == domain || strings.HasSuffix(domain, "."+dl) {
			return fmt.Errorf("domain %s is denied", domain)
		}
	}

	if len(t.cfg.AllowedDomains) > 0 {
		allowed := false
		for _, d := range t.cfg.AllowedDomains {
			dl := strings.ToLower(d)
			if dl == domain || strings.HasSuffix(domain, "."+dl) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("domain %s is not in allowed list", domain)
		}
	}

	return nil
}

// isPrivateHost returns true for loopback, private, and link-local addresses.
func isPrivateHost(host string) bool {
	// Check common localhost names
	lower := strings.ToLower(host)
	if lower == "localhost" || lower == "ip6-localhost" || lower == "ip6-loopback" {
		return true
	}

	ip := net.ParseIP(host)
	if ip == nil {
		// Could be a hostname that resolves to a private IP.
		// We can't do DNS resolution here without risk, so rely on domain checks.
		return false
	}

	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

func (t *BrowserTool) navigate(ctx context.Context, params browserParams) (*Result, error) {
	if params.URL == "" {
		return &Result{Error: "url is required for navigate action", IsError: true}, nil
	}

	if err := t.validateURL(params.URL); err != nil {
		return &Result{Error: err.Error(), IsError: true}, nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.pages) >= t.cfg.MaxTabs {
		return &Result{Error: fmt.Sprintf("max tabs limit reached (%d)", t.cfg.MaxTabs), IsError: true}, nil
	}

	if err := t.ensureBrowser(); err != nil {
		return &Result{Error: err.Error(), IsError: true}, nil
	}

	page, err := t.browser.Page(proto.TargetCreateTarget{URL: params.URL})
	if err != nil {
		return &Result{Error: "failed to open page: " + err.Error(), IsError: true}, nil
	}

	if err := page.WaitLoad(); err != nil {
		return &Result{Error: "page load timeout: " + err.Error(), IsError: true}, nil
	}

	t.nextID++
	pageID := fmt.Sprintf("page_%d", t.nextID)
	t.pages[pageID] = page

	title, _ := page.Eval(`() => document.title`)
	titleStr := ""
	if title != nil {
		titleStr = title.Value.Str()
	}

	return &Result{Output: fmt.Sprintf("Opened page %s: %s (title: %s)", pageID, params.URL, titleStr)}, nil
}

func (t *BrowserTool) getPage(pageID string) (*rod.Page, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	page, ok := t.pages[pageID]
	if !ok {
		return nil, fmt.Errorf("page not found: %s", pageID)
	}
	return page, nil
}

func (t *BrowserTool) getContent(_ context.Context, params browserParams) (*Result, error) {
	if params.PageID == "" {
		return &Result{Error: "page_id is required", IsError: true}, nil
	}

	page, err := t.getPage(params.PageID)
	if err != nil {
		return &Result{Error: err.Error(), IsError: true}, nil
	}

	text, err := page.Eval(`() => document.body.innerText`)
	if err != nil {
		return &Result{Error: "failed to get content: " + err.Error(), IsError: true}, nil
	}

	content := text.Value.Str()
	maxChars := t.cfg.MaxPageSizeKB * 1024
	if len(content) > maxChars {
		content = content[:maxChars] + "\n... (content truncated)"
	}

	return &Result{Output: content}, nil
}

func (t *BrowserTool) click(_ context.Context, params browserParams) (*Result, error) {
	if params.PageID == "" || params.Selector == "" {
		return &Result{Error: "page_id and selector are required", IsError: true}, nil
	}

	page, err := t.getPage(params.PageID)
	if err != nil {
		return &Result{Error: err.Error(), IsError: true}, nil
	}

	el, err := page.Element(params.Selector)
	if err != nil {
		return &Result{Error: "element not found: " + err.Error(), IsError: true}, nil
	}

	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return &Result{Error: "click failed: " + err.Error(), IsError: true}, nil
	}

	return &Result{Output: fmt.Sprintf("Clicked element: %s", params.Selector)}, nil
}

func (t *BrowserTool) fill(_ context.Context, params browserParams) (*Result, error) {
	if params.PageID == "" || params.Selector == "" {
		return &Result{Error: "page_id, selector, and text are required", IsError: true}, nil
	}

	page, err := t.getPage(params.PageID)
	if err != nil {
		return &Result{Error: err.Error(), IsError: true}, nil
	}

	el, err := page.Element(params.Selector)
	if err != nil {
		return &Result{Error: "element not found: " + err.Error(), IsError: true}, nil
	}

	if err := el.SelectAllText(); err != nil {
		return &Result{Error: "failed to select text: " + err.Error(), IsError: true}, nil
	}

	if err := el.Input(params.Text); err != nil {
		return &Result{Error: "failed to fill: " + err.Error(), IsError: true}, nil
	}

	return &Result{Output: fmt.Sprintf("Filled '%s' with text (%d chars)", params.Selector, len(params.Text))}, nil
}

func (t *BrowserTool) screenshot(_ context.Context, params browserParams) (*Result, error) {
	if params.PageID == "" {
		return &Result{Error: "page_id is required", IsError: true}, nil
	}

	page, err := t.getPage(params.PageID)
	if err != nil {
		return &Result{Error: err.Error(), IsError: true}, nil
	}

	quality := 80
	data, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: &quality,
	})
	if err != nil {
		return &Result{Error: "screenshot failed: " + err.Error(), IsError: true}, nil
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return &Result{Output: fmt.Sprintf("data:image/jpeg;base64,%s", encoded)}, nil
}

func (t *BrowserTool) evalJS(_ context.Context, params browserParams) (*Result, error) {
	if params.PageID == "" || params.Script == "" {
		return &Result{Error: "page_id and script are required", IsError: true}, nil
	}

	page, err := t.getPage(params.PageID)
	if err != nil {
		return &Result{Error: err.Error(), IsError: true}, nil
	}

	result, err := page.Eval(params.Script)
	if err != nil {
		return &Result{Error: "eval failed: " + err.Error(), IsError: true}, nil
	}

	output := result.Value.String()
	if len(output) > 10000 {
		output = output[:10000] + "\n... (output truncated)"
	}

	return &Result{Output: output}, nil
}

func (t *BrowserTool) getLinks(_ context.Context, params browserParams) (*Result, error) {
	if params.PageID == "" {
		return &Result{Error: "page_id is required", IsError: true}, nil
	}

	page, err := t.getPage(params.PageID)
	if err != nil {
		return &Result{Error: err.Error(), IsError: true}, nil
	}

	result, err := page.Eval(`() => {
		const links = Array.from(document.querySelectorAll('a[href]'));
		return links.map(a => ({ text: a.innerText.trim().substring(0, 100), href: a.href })).filter(l => l.href && l.href !== 'javascript:void(0)');
	}`)
	if err != nil {
		return &Result{Error: "failed to get links: " + err.Error(), IsError: true}, nil
	}

	output, _ := json.MarshalIndent(result.Value, "", "  ")
	s := string(output)
	if len(s) > 10000 {
		s = s[:10000] + "\n... (truncated)"
	}

	return &Result{Output: s}, nil
}

func (t *BrowserTool) closePage(params browserParams) (*Result, error) {
	if params.PageID == "" {
		return &Result{Error: "page_id is required", IsError: true}, nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	page, ok := t.pages[params.PageID]
	if !ok {
		return &Result{Error: "page not found: " + params.PageID, IsError: true}, nil
	}

	if page != nil {
		page.Close()
	}
	delete(t.pages, params.PageID)

	return &Result{Output: fmt.Sprintf("Closed page %s", params.PageID)}, nil
}

// Close shuts down the browser and all pages.
func (t *BrowserTool) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for id, page := range t.pages {
		if page != nil {
			page.Close()
		}
		delete(t.pages, id)
	}

	if t.browser != nil {
		t.browser.Close()
		t.browser = nil
	}
}
