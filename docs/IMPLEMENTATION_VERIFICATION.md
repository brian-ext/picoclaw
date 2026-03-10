# Implementation Verification - PinchTab & PicoClaw

## Overview

This document verifies our implementation against official PinchTab and PicoClaw documentation to ensure D2/D3 will work correctly.

---

## ✅ PinchTab API Compliance

### Official API Endpoints (from pinchtab docs)

| Endpoint | Method | Our Implementation | Status |
|----------|--------|-------------------|--------|
| `/navigate` | POST | ✅ `/nav` | ⚠️ **MISMATCH** |
| `/evaluate` | POST | ✅ `/evaluate` | ✅ Correct |
| `/text` | GET | ✅ `/text` | ✅ Correct |
| `/screenshot` | GET | ✅ `/screenshot` | ✅ Correct |
| `/action` | POST | ✅ `/action` | ✅ Correct |

### ⚠️ Critical Issue Found

**Navigate endpoint mismatch:**
- **Official PinchTab API:** `POST /navigate`
- **Our implementation:** `POST /nav`

**From PinchTab docs:**
```bash
curl -X POST http://localhost:9867/navigate \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://pinchtab.com"}'
```

**Our code (pkg/tools/pinchtab.go:115):**
```go
resp, err := t.doRequest(ctx, "POST", "/nav", payload)
```

**Fix required:** Change `/nav` to `/navigate`

---

## ✅ PicoClaw Tool Interface Compliance

### Required Methods (from pkg/tools/base.go)

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]any
    Execute(ctx context.Context, args map[string]any) *ToolResult
}
```

### PinchTab Tool Implementation

| Method | Implemented | Signature Correct | Status |
|--------|-------------|-------------------|--------|
| `Name()` | ✅ | ✅ Returns "pinchtab" | ✅ |
| `Description()` | ✅ | ✅ Returns string | ✅ |
| `Parameters()` | ✅ | ✅ Returns map[string]any | ✅ |
| `Execute()` | ✅ | ✅ Correct signature | ✅ |

**Verdict:** Tool interface compliance is **correct**.

---

## ✅ Librarian Tool Implementation

### Required Methods

| Method | Implemented | Signature Correct | Status |
|--------|-------------|-------------------|--------|
| `Name()` | ✅ | ✅ Returns "librarian" | ✅ |
| `Description()` | ✅ | ✅ Returns string | ✅ |
| `Parameters()` | ✅ | ✅ Returns map[string]any | ✅ |
| `Execute()` | ✅ | ✅ Correct signature | ✅ |

**Verdict:** Tool interface compliance is **correct**.

---

## ⚠️ PinchTab Default Port

### Official Documentation

**From PinchTab README:**
- Default server port: **9867**
- Dashboard: `http://localhost:9867`

**Our implementation:**
- Default in code: **9870** (pkg/tools/pinchtab.go:22)
- Environment variable: `PICOCLAW_PINCHTAB_URL`

**From our code:**
```go
func NewPinchTabTool(baseURL string) *PinchTabTool {
    if baseURL == "" {
        baseURL = "http://127.0.0.1:9870"  // ⚠️ Should be 9867
    }
    // ...
}
```

**Fix required:** Change default port from 9870 to 9867

---

## ✅ Agent Loop Registration

### Registration Code (pkg/agent/loop.go:172-179)

```go
// PinchTab browser automation tool
if cfg.Tools.IsToolEnabled("pinchtab") {
    pinchtabURL := "http://127.0.0.1:9870"
    if url := os.Getenv("PICOCLAW_PINCHTAB_URL"); url != "" {
        pinchtabURL = url
    }
    agent.Tools.Register(tools.NewPinchTabTool(pinchtabURL))
}
```

**Issues:**
1. ⚠️ Default port should be 9867 (not 9870)
2. ✅ Environment variable override works correctly
3. ✅ Registration pattern is correct

---

## ✅ Whiteboard JS API

### API Methods (web/whiteboard/js/picoclaw-api.js)

| Method | Implemented | Parameters | Status |
|--------|-------------|------------|--------|
| `highlightRect()` | ✅ | x, y, w, h, color | ✅ |
| `highlightCircle()` | ✅ | x, y, r, color | ✅ |
| `addText()` | ✅ | x, y, text, color, size | ✅ |
| `clear()` | ✅ | none | ✅ |
| `getState()` | ✅ | none | ✅ |

**Verdict:** Whiteboard API is **correct**.

---

## 🔧 Required Fixes

### 1. Fix PinchTab Navigate Endpoint

**File:** `pkg/tools/pinchtab.go`  
**Line:** 115  
**Current:** `resp, err := t.doRequest(ctx, "POST", "/nav", payload)`  
**Fix to:** `resp, err := t.doRequest(ctx, "POST", "/navigate", payload)`

### 2. Fix PinchTab Default Port

**File:** `pkg/tools/pinchtab.go`  
**Line:** 22  
**Current:** `baseURL = "http://127.0.0.1:9870"`  
**Fix to:** `baseURL = "http://127.0.0.1:9867"`

**File:** `pkg/agent/loop.go`  
**Line:** 173  
**Current:** `pinchtabURL := "http://127.0.0.1:9870"`  
**Fix to:** `pinchtabURL := "http://127.0.0.1:9867"`

---

## ✅ Testing Checklist

After applying fixes, verify:

### PinchTab Server
- [ ] Start PinchTab: `pinchtab` (not `pinchtab server`)
- [ ] Verify port: `http://localhost:9867`
- [ ] Check health: `curl http://localhost:9867/health`

### Gateway
- [ ] Start Gateway: `picoclaw gateway`
- [ ] Verify whiteboard: `http://localhost:18790/whiteboard/`
- [ ] Check whiteboard API: Open browser console, run `window.picoclaw.getState()`

### PinchTab Tool (via curl)
```bash
# Test navigate (after fix)
curl -X POST http://localhost:9867/navigate \
  -H "Content-Type: application/json" \
  -d '{"url":"http://localhost:18790/whiteboard/"}'

# Test evaluate
curl -X POST http://localhost:9867/evaluate \
  -H "Content-Type: application/json" \
  -d '{"expression":"window.picoclaw.highlightRect(0.3, 0.3, 0.2, 0.2, \"#ff0000\")"}'

# Test screenshot
curl "http://localhost:9867/screenshot?raw=true" -o test.jpg
```

### D2 Test (via LLM)
```
User: "Navigate to the whiteboard and highlight a red box in the center"

Expected:
1. AI uses pinchtab tool with action: navigate
2. AI uses pinchtab tool with action: evaluate (JavaScript)
3. Red box appears on whiteboard
```

### D3 Test (via LLM)
```
User: "Add a blue circle and take a screenshot to verify"

Expected:
1. AI uses pinchtab tool with action: evaluate (highlightCircle)
2. AI uses pinchtab tool with action: screenshot
3. Screenshot shows blue circle
```

---

## 📋 Summary

**Issues Found:** 2  
**Critical:** 1 (navigate endpoint)  
**Minor:** 1 (default port)

**Fixes Required:**
1. Change `/nav` → `/navigate` in pinchtab.go
2. Change port `9870` → `9867` in pinchtab.go and loop.go

**After fixes:** Implementation should be fully compliant with official PinchTab API and ready for D2/D3 testing.

---

## 🎯 Confidence Level

**Before fixes:** 60% (endpoint mismatch would cause failures)  
**After fixes:** 95% (should work correctly)

**Remaining 5% risk:**
- Browser automation environment setup
- Chrome/Chromium availability
- Network/firewall issues

**Recommendation:** Apply fixes, then test on machine with browser automation capability.
