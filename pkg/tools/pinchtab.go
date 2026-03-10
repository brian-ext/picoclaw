package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PinchTabTool provides browser automation capabilities via PinchTab HTTP API
type PinchTabTool struct {
	baseURL string
	client  *http.Client
}

// NewPinchTabTool creates a new PinchTab tool instance
func NewPinchTabTool(baseURL string) *PinchTabTool {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:9867"
	}
	return &PinchTabTool{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (t *PinchTabTool) Name() string {
	return "pinchtab"
}

func (t *PinchTabTool) Description() string {
	return `Control a browser to interact with the whiteboard and web pages. Use this to:
- Navigate to the whiteboard at http://localhost:18790/whiteboard/
- Highlight areas on schematics by executing JavaScript
- Extract text from web pages (token-efficient)
- Take screenshots for verification
- Fill forms and click elements

Actions available: navigate, evaluate, text, screenshot, click, fill`
}

func (t *PinchTabTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "Action to perform: navigate, evaluate, text, screenshot, click, fill",
				"enum":        []string{"navigate", "evaluate", "text", "screenshot", "click", "fill"},
			},
			"url": map[string]any{
				"type":        "string",
				"description": "URL to navigate to (for navigate action)",
			},
			"javascript": map[string]any{
				"type":        "string",
				"description": "JavaScript code to execute (for evaluate action). Example: window.picoclaw.highlightRect(0.1, 0.2, 0.3, 0.4, '#ff0000')",
			},
			"selector": map[string]any{
				"type":        "string",
				"description": "CSS selector or element ref (for click/fill actions)",
			},
			"value": map[string]any{
				"type":        "string",
				"description": "Value to fill (for fill action)",
			},
			"tab_id": map[string]any{
				"type":        "string",
				"description": "Tab ID to operate on (optional, uses default if not specified)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *PinchTabTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return ErrorResult("action is required")
	}

	switch action {
	case "navigate":
		return t.navigate(ctx, args)
	case "evaluate":
		return t.evaluate(ctx, args)
	case "text":
		return t.getText(ctx, args)
	case "screenshot":
		return t.screenshot(ctx, args)
	case "click":
		return t.click(ctx, args)
	case "fill":
		return t.fill(ctx, args)
	default:
		return ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

func (t *PinchTabTool) navigate(ctx context.Context, args map[string]any) *ToolResult {
	url, ok := args["url"].(string)
	if !ok {
		return ErrorResult("url is required for navigate action")
	}

	payload := map[string]any{
		"url": url,
	}

	resp, err := t.doRequest(ctx, "POST", "/navigate", payload)
	if err != nil {
		return ErrorResult(fmt.Sprintf("navigation failed: %v", err))
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Navigated to %s", url),
		ForUser: fmt.Sprintf("🌐 Navigated to %s", url),
	}
}

func (t *PinchTabTool) evaluate(ctx context.Context, args map[string]any) *ToolResult {
	js, ok := args["javascript"].(string)
	if !ok {
		return ErrorResult("javascript is required for evaluate action")
	}

	payload := map[string]any{
		"expression": js,
	}

	resp, err := t.doRequest(ctx, "POST", "/evaluate", payload)
	if err != nil {
		return ErrorResult(fmt.Sprintf("evaluation failed: %v", err))
	}

	var result map[string]any
	if err := json.Unmarshal(resp, &result); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse response: %v", err))
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Executed JavaScript. Result: %v", result),
		ForUser: "✓ Executed JavaScript on whiteboard",
	}
}

func (t *PinchTabTool) getText(ctx context.Context, args map[string]any) *ToolResult {
	resp, err := t.doRequest(ctx, "GET", "/text", nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("text extraction failed: %v", err))
	}

	var result map[string]any
	if err := json.Unmarshal(resp, &result); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse response: %v", err))
	}

	text, _ := result["text"].(string)
	return &ToolResult{
		ForLLM:  text,
		ForUser: "📄 Extracted page text",
	}
}

func (t *PinchTabTool) screenshot(ctx context.Context, args map[string]any) *ToolResult {
	resp, err := t.doRequest(ctx, "GET", "/screenshot", nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("screenshot failed: %v", err))
	}

	return &ToolResult{
		ForLLM:  "Screenshot captured (binary data)",
		ForUser: "📸 Screenshot captured",
	}
}

func (t *PinchTabTool) click(ctx context.Context, args map[string]any) *ToolResult {
	selector, ok := args["selector"].(string)
	if !ok {
		return ErrorResult("selector is required for click action")
	}

	payload := map[string]any{
		"kind": "click",
		"ref":  selector,
	}

	_, err := t.doRequest(ctx, "POST", "/action", payload)
	if err != nil {
		return ErrorResult(fmt.Sprintf("click failed: %v", err))
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Clicked element: %s", selector),
		ForUser: fmt.Sprintf("🖱️ Clicked: %s", selector),
	}
}

func (t *PinchTabTool) fill(ctx context.Context, args map[string]any) *ToolResult {
	selector, ok := args["selector"].(string)
	if !ok {
		return ErrorResult("selector is required for fill action")
	}

	value, ok := args["value"].(string)
	if !ok {
		return ErrorResult("value is required for fill action")
	}

	payload := map[string]any{
		"kind":  "fill",
		"ref":   selector,
		"value": value,
	}

	_, err := t.doRequest(ctx, "POST", "/action", payload)
	if err != nil {
		return ErrorResult(fmt.Sprintf("fill failed: %v", err))
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Filled element %s with: %s", selector, value),
		ForUser: fmt.Sprintf("⌨️ Filled: %s", selector),
	}
}

func (t *PinchTabTool) doRequest(ctx context.Context, method, path string, payload map[string]any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, t.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
