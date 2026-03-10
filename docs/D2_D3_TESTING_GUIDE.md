# D2/D3 Testing Guide - PinchTab Whiteboard Automation

## Overview

This guide provides complete instructions for testing D2 (PinchTab drives whiteboard) and D3 (First annotation automation) on a test machine with browser automation capability.

**Status:** All code implemented and verified ✅  
**Blockers:** Requires test machine with Chrome/Chromium

---

## Prerequisites

### Hardware Requirements
- Test machine with display capability (or headless with Xvfb)
- Minimum 2GB RAM
- Network connectivity to localhost

### Software Requirements
- **Chrome or Chromium** browser installed
- **Go 1.21+** (for building PicoClaw)
- **PinchTab** installed
- **Windows/Linux/macOS** (any OS supported by PinchTab)

---

## Installation Steps

### 1. Install PinchTab

**macOS/Linux:**
```bash
curl -fsSL https://pinchtab.com/install.sh | bash
```

**npm:**
```bash
npm install -g pinchtab
```

**Docker:**
```bash
docker run -d -p 9867:9867 pinchtab/pinchtab
```

**Verify installation:**
```bash
pinchtab --version
```

### 2. Build PicoClaw

```bash
cd c:\Users\sschu\p2p-garage\notes\home\picoclaw

# Install dependencies
go get github.com/amikos-tech/chroma-go
go get github.com/ledongthuc/pdf
go get github.com/mattn/go-sqlite3

# Build
go build -o picoclaw.exe ./cmd/picoclaw
```

### 3. Configure PicoClaw

Create `config/config.test.json`:
```json
{
  "workspace_path": "~/.picoclaw/workspace",
  "providers": {
    "openai": {
      "enabled": true,
      "api_key": "YOUR_API_KEY",
      "model": "gpt-4"
    }
  },
  "tools": {
    "pinchtab": {
      "enabled": true
    },
    "librarian": {
      "enabled": true
    },
    "web_fetch": {
      "enabled": true
    },
    "message": {
      "enabled": true
    }
  },
  "channels": {
    "cli": {
      "enabled": true
    }
  },
  "web": {
    "enabled": true,
    "port": 18790
  }
}
```

---

## Test Execution

### Terminal Setup

**Terminal 1 - PinchTab Server:**
```bash
pinchtab
```

Expected output:
```
PinchTab v1.x.x
Server listening on http://127.0.0.1:9867
Dashboard: http://127.0.0.1:9867/dashboard
```

**Terminal 2 - PicoClaw Gateway:**
```bash
./picoclaw.exe gateway --config config/config.test.json
```

Expected output:
```
Gateway server listening on :18790
Whiteboard available at http://localhost:18790/whiteboard/
```

**Terminal 3 - Testing:**
Keep this terminal free for test commands

---

## D2 Tests - PinchTab Drives Whiteboard

### Test 2.1: Navigate to Whiteboard

**Objective:** Verify PinchTab can navigate to the whiteboard

**Command:**
```bash
curl -X POST http://localhost:9867/navigate \
  -H "Content-Type: application/json" \
  -d '{"url":"http://localhost:18790/whiteboard/"}'
```

**Expected Response:**
```json
{
  "success": true,
  "url": "http://localhost:18790/whiteboard/",
  "title": "Go Drawing Board"
}
```

**Verification:**
- ✅ HTTP 200 response
- ✅ No error messages
- ✅ PinchTab browser window shows whiteboard (if headed mode)

---

### Test 2.2: Capture Screenshot

**Objective:** Verify PinchTab can capture whiteboard screenshot

**Command:**
```bash
curl "http://localhost:9867/screenshot?raw=true" -o whiteboard_test.jpg
```

**Expected Result:**
- ✅ File `whiteboard_test.jpg` created
- ✅ File size > 0 bytes
- ✅ Image shows blank whiteboard canvas

**Verification:**
```bash
# Check file exists and size
ls -lh whiteboard_test.jpg

# Open image (Windows)
start whiteboard_test.jpg

# Open image (Linux)
xdg-open whiteboard_test.jpg

# Open image (macOS)
open whiteboard_test.jpg
```

---

### Test 2.3: Verify Whiteboard JS API

**Objective:** Confirm `window.picoclaw` API is available

**Command:**
```bash
curl -X POST http://localhost:9867/evaluate \
  -H "Content-Type: application/json" \
  -d '{"expression":"typeof window.picoclaw"}'
```

**Expected Response:**
```json
{
  "result": "object"
}
```

**Verification:**
- ✅ Returns "object" (not "undefined")
- ✅ No JavaScript errors

---

## D3 Tests - First Annotation Automation

### Test 3.1: Highlight Red Rectangle

**Objective:** PinchTab executes JavaScript to draw on whiteboard

**Command:**
```bash
curl -X POST http://localhost:9867/evaluate \
  -H "Content-Type: application/json" \
  -d '{"expression":"window.picoclaw.highlightRect(0.3, 0.3, 0.2, 0.2, \"#ff0000\")"}'
```

**Expected Response:**
```json
{
  "result": true
}
```

**Verification:**
```bash
# Capture screenshot to verify
curl "http://localhost:9867/screenshot?raw=true" -o test_red_rect.jpg
```

**Visual Check:**
- ✅ Red rectangle visible on whiteboard
- ✅ Rectangle at position (30%, 30%) from top-left
- ✅ Rectangle size 20% x 20% of canvas

---

### Test 3.2: Highlight Blue Circle

**Objective:** Test circle drawing function

**Command:**
```bash
curl -X POST http://localhost:9867/evaluate \
  -H "Content-Type: application/json" \
  -d '{"expression":"window.picoclaw.highlightCircle(0.5, 0.5, 0.1, \"#0000ff\")"}'
```

**Verification:**
```bash
curl "http://localhost:9867/screenshot?raw=true" -o test_blue_circle.jpg
```

**Visual Check:**
- ✅ Blue circle visible
- ✅ Circle at center (50%, 50%)
- ✅ Circle radius 10% of canvas

---

### Test 3.3: Add Text Annotation

**Objective:** Test text drawing function

**Command:**
```bash
curl -X POST http://localhost:9867/evaluate \
  -H "Content-Type: application/json" \
  -d '{"expression":"window.picoclaw.addText(0.1, 0.1, \"Test Annotation\", \"#00ff00\", 24)"}'
```

**Verification:**
```bash
curl "http://localhost:9867/screenshot?raw=true" -o test_text.jpg
```

**Visual Check:**
- ✅ Green text "Test Annotation" visible
- ✅ Text at position (10%, 10%)
- ✅ Text size 24px

---

### Test 3.4: Clear Canvas

**Objective:** Test clear function

**Command:**
```bash
curl -X POST http://localhost:9867/evaluate \
  -H "Content-Type: application/json" \
  -d '{"expression":"window.picoclaw.clear()"}'
```

**Verification:**
```bash
curl "http://localhost:9867/screenshot?raw=true" -o test_cleared.jpg
```

**Visual Check:**
- ✅ Canvas is blank
- ✅ No annotations visible

---

## End-to-End LLM Test

### Test E2E.1: AI-Driven Whiteboard Annotation

**Objective:** Verify AI can use PinchTab to annotate whiteboard

**Setup:**
Start PicoClaw in CLI mode:
```bash
./picoclaw.exe cli --config config/config.test.json
```

**Test Conversation:**
```
User: Navigate to the whiteboard and draw a red box in the center

Expected AI Actions:
1. Use pinchtab tool: action=navigate, url=http://localhost:18790/whiteboard/
2. Use pinchtab tool: action=evaluate, javascript=window.picoclaw.highlightRect(0.4, 0.4, 0.2, 0.2, "#ff0000")
3. Use pinchtab tool: action=screenshot

Expected AI Response:
"I've navigated to the whiteboard and drawn a red box in the center. Screenshot captured for verification."
```

**Verification:**
- ✅ AI uses pinchtab tool (check logs)
- ✅ Red box appears on whiteboard
- ✅ AI confirms completion

---

### Test E2E.2: Multi-Step Annotation

**Test Conversation:**
```
User: Draw a blue circle, add text "Engine Diagram", then take a screenshot

Expected AI Actions:
1. pinchtab: evaluate → highlightCircle(0.5, 0.5, 0.15, "#0000ff")
2. pinchtab: evaluate → addText(0.5, 0.2, "Engine Diagram", "#000000", 32)
3. pinchtab: screenshot

Expected AI Response:
"I've drawn a blue circle and added the text 'Engine Diagram' to the whiteboard. Screenshot captured."
```

**Verification:**
- ✅ Blue circle visible
- ✅ Text "Engine Diagram" visible
- ✅ Screenshot captured

---

## Troubleshooting

### Issue: PinchTab fails to start

**Error:** `Failed to find Chrome/Chromium`

**Solution:**
```bash
# Install Chrome (Ubuntu/Debian)
sudo apt-get install chromium-browser

# Install Chrome (macOS)
brew install --cask google-chrome

# Install Chrome (Windows)
# Download from https://www.google.com/chrome/
```

---

### Issue: Navigate returns 404

**Error:** `HTTP 404: Not Found`

**Solution:**
```bash
# Verify Gateway is running
curl http://localhost:18790/health

# Verify whiteboard route
curl http://localhost:18790/whiteboard/

# Check Gateway logs for errors
```

---

### Issue: JavaScript evaluation fails

**Error:** `window.picoclaw is undefined`

**Solution:**
```bash
# Wait 2-3 seconds after navigation for page to load
curl -X POST http://localhost:9867/navigate \
  -H "Content-Type: application/json" \
  -d '{"url":"http://localhost:18790/whiteboard/"}'

sleep 3

# Then evaluate
curl -X POST http://localhost:9867/evaluate \
  -H "Content-Type: application/json" \
  -d '{"expression":"window.picoclaw.getState()"}'
```

---

### Issue: Screenshot is blank

**Possible causes:**
1. Page not fully loaded
2. Canvas not initialized
3. Headless rendering issue

**Solution:**
```bash
# Try headed mode (visible browser)
# Edit PinchTab config or use:
pinchtab --headed

# Or add delay before screenshot
sleep 5
curl "http://localhost:9867/screenshot?raw=true" -o test.jpg
```

---

## Success Criteria

### D2 Complete When:
- ✅ PinchTab navigates to whiteboard successfully
- ✅ Screenshot captures whiteboard canvas
- ✅ `window.picoclaw` API is accessible

### D3 Complete When:
- ✅ PinchTab executes `highlightRect()` successfully
- ✅ Red rectangle appears on whiteboard
- ✅ Screenshot shows the annotation
- ✅ All whiteboard API methods work (circle, text, clear)

### Full Integration Complete When:
- ✅ AI can navigate to whiteboard via PinchTab tool
- ✅ AI can execute JavaScript annotations
- ✅ AI can capture screenshots for verification
- ✅ Multi-step annotation workflows succeed

---

## Deployment Checklist

Before testing on test machine:

- [ ] PinchTab installed and verified
- [ ] Chrome/Chromium installed
- [ ] PicoClaw built successfully
- [ ] config.test.json created with API keys
- [ ] Network ports 9867 and 18790 available
- [ ] All dependencies installed (ChromaDB, SQLite, PDF parser)

---

## Next Steps After D2/D3

Once D2/D3 are verified:

1. **Test with real repair scenario:**
   - Load schematic PDF
   - AI highlights relevant components
   - User confirms understanding

2. **Test charm.li integration:**
   - Query 1999 Buick LeSabre manual
   - Verify on-demand fetching works
   - Confirm caching to ChromaDB

3. **Test VIN-keyed sessions:**
   - Create machine profile for test vehicle
   - Run repair session
   - Verify session data stored correctly

4. **Production deployment:**
   - Deploy to garage tablet/SBC
   - Configure for offline use
   - Test in real repair environment

---

## Contact & Support

**Issues found during testing:**
- Document in GitHub issues
- Include PinchTab version
- Include PicoClaw logs
- Include screenshot if visual issue

**Test machine requirements:**
- Minimum: 2GB RAM, Chrome installed
- Recommended: 4GB RAM, SSD, Linux/macOS
- Optimal: 8GB RAM, dedicated GPU (for headed mode)

---

**D2/D3 are ready for testing!** All code verified against official documentation. 🚀
