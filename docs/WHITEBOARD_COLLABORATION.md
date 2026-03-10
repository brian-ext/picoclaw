# Whiteboard Collaboration - Human ↔ AI Interaction

## Overview

PicoClaw's whiteboard collaboration system enables real-time interaction between humans and AI on a shared canvas. The human can draw, highlight, and annotate schematics while PicoClaw uses PinchTab to "see" and interact with the same whiteboard.

## Architecture

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│   Human     │◄───────►│  Whiteboard  │◄───────►│  PicoClaw   │
│  (Browser)  │         │  (Gateway)   │         │   + AI      │
└─────────────┘         └──────────────┘         └─────────────┘
                               ▲                         │
                               │                         ▼
                               │                  ┌─────────────┐
                               └──────────────────│  PinchTab   │
                                                  │  (Browser)  │
                                                  └─────────────┘
```

### Components

1. **Whiteboard** (`/whiteboard/`) - Interactive canvas served by Gateway (port 18790)
2. **PicoClaw API** (`window.picoclaw`) - JavaScript API for programmatic control
3. **PinchTab Tool** - Browser automation bridge for AI interaction
4. **Gateway** - Serves whiteboard and handles HTTP routing

## Usage

### 1. Start the System

```bash
# Terminal 1: Start PicoClaw Gateway
./picoclaw gateway

# Terminal 2: Start PinchTab (if not using autospawn)
pinchtab server
# OR enable autospawn:
export PICOCLAW_PINCHTAB_AUTOSPAWN=1
./picoclaw gateway
```

### 2. Access the Whiteboard

**Human:** Open browser to `http://localhost:18790/whiteboard/`

**AI:** Uses PinchTab tool to navigate:
```json
{
  "action": "navigate",
  "url": "http://localhost:18790/whiteboard/"
}
```

### 3. Collaboration Workflow

#### Human Marks an Issue
1. Human draws/highlights a problem area on schematic
2. Human asks: "What's wrong with this connector?"

#### AI Analyzes and Responds
1. **PicoClaw extracts text** from whiteboard (token-efficient):
   ```json
   {"action": "text"}
   ```

2. **PicoClaw highlights the answer** on the schematic:
   ```json
   {
     "action": "evaluate",
     "javascript": "window.picoclaw.highlightRect(0.3, 0.4, 0.2, 0.15, '#00ff00')"
   }
   ```

3. **PicoClaw adds annotation**:
   ```json
   {
     "action": "evaluate",
     "javascript": "window.picoclaw.addText('Pin 3: Ground', 0.35, 0.38, '#00ff00', 14)"
   }
   ```

4. **PicoClaw captures verification screenshot**:
   ```json
   {"action": "screenshot"}
   ```

## PicoClaw API Reference

The whiteboard exposes `window.picoclaw` with the following methods:

### `highlightRect(x, y, width, height, color, lineWidth)`
Draw a rectangle highlight.
- **x, y, width, height**: 0-1 (relative to canvas dimensions)
- **color**: Hex or named color (default: `#ff0000`)
- **lineWidth**: Pixels (default: 3)

**Example:**
```javascript
// Highlight top-left quadrant in red
window.picoclaw.highlightRect(0.1, 0.1, 0.3, 0.3, '#ff0000', 3)
```

### `highlightCircle(x, y, radius, color, lineWidth)`
Draw a circle highlight.
- **x, y**: Center position (0-1, relative)
- **radius**: 0-1 (relative to canvas width)

**Example:**
```javascript
// Circle a specific component
window.picoclaw.highlightCircle(0.5, 0.5, 0.1, '#00ff00', 2)
```

### `addText(text, x, y, color, fontSize)`
Add text annotation.
- **text**: String to display
- **x, y**: Position (0-1, relative)
- **fontSize**: Pixels (default: 16)

**Example:**
```javascript
window.picoclaw.addText('Check this wire', 0.4, 0.3, '#ffff00', 18)
```

### `clear()`
Clear the entire canvas.

### `getDataURL(format)`
Get canvas as data URL for analysis.
- **format**: Image format (default: `'image/png'`)

### `getState()`
Get current canvas state (width, height, color, size).

## PinchTab Tool Actions

### Navigate
```json
{
  "action": "navigate",
  "url": "http://localhost:18790/whiteboard/"
}
```

### Execute JavaScript
```json
{
  "action": "evaluate",
  "javascript": "window.picoclaw.highlightRect(0.2, 0.3, 0.4, 0.2, '#ff0000')"
}
```

### Extract Text (Token-Efficient)
```json
{
  "action": "text"
}
```
Returns ~800 tokens/page (5-13x cheaper than screenshots)

### Capture Screenshot
```json
{
  "action": "screenshot"
}
```

### Click Element
```json
{
  "action": "click",
  "selector": "e5"
}
```

### Fill Input
```json
{
  "action": "fill",
  "selector": "e3",
  "value": "VIN: 1G1ZT53826F109149"
}
```

## Configuration

### Enable PinchTab Tool

Add to `config.json`:
```json
{
  "tools": {
    "enabled": ["pinchtab", "web", "filesystem"],
    "pinchtab": {
      "enabled": true
    }
  }
}
```

### Environment Variables

```bash
# PinchTab autospawn (optional)
export PICOCLAW_PINCHTAB_AUTOSPAWN=1
export PICOCLAW_PINCHTAB_PORT=9870
export PICOCLAW_PINCHTAB_BIND=127.0.0.1

# Custom PinchTab URL
export PICOCLAW_PINCHTAB_URL=http://127.0.0.1:9870
```

## Example Scenarios

### Scenario 1: Identify Connector Pinout

**Human:** Uploads schematic, highlights connector, asks "What are these pins?"

**AI Workflow:**
1. Navigate to whiteboard
2. Extract text to identify connector type
3. Query Librarian for pinout data
4. Highlight each pin with color-coded annotations
5. Add text labels for each pin function

### Scenario 2: Diagnose Wiring Issue

**Human:** Draws on schematic showing where voltage is missing

**AI Workflow:**
1. Extract whiteboard text
2. Identify the circuit path
3. Highlight suspected break point in red
4. Circle the fuse location in yellow
5. Add text: "Check F12 - 15A fuse"

### Scenario 3: Step-by-Step Repair Guide

**Human:** "Walk me through replacing this part"

**AI Workflow:**
1. Highlight step 1 area in green
2. Add text: "1. Disconnect battery negative"
3. Wait for human confirmation
4. Highlight step 2 area in green
5. Add text: "2. Remove these 3 bolts"
6. Continue sequence...

## Token Efficiency

**Text Extraction:** ~800 tokens/page
- Use for reading whiteboard content
- 5-13x cheaper than vision models

**Screenshots:** Use sparingly
- Only for verification
- When visual analysis is required
- After making annotations

## Safety Considerations

For P2P Garage OS (repair assistant):

1. **Verify before critical steps**
   - Battery disconnect
   - Jack stand placement
   - Torque specifications

2. **Require Librarian citations**
   - No specs without source
   - Link to manual page
   - Show confidence level

3. **Visual confirmation**
   - Request photos of actual machine
   - Compare to schematic
   - Verify mid-year changes

## Troubleshooting

### Whiteboard not loading
- Check Gateway is running on port 18790
- Visit `http://localhost:18790/health`
- Check browser console for errors

### PinchTab not connecting
- Verify PinchTab is running: `curl http://localhost:9870/health`
- Check `PICOCLAW_PINCHTAB_URL` environment variable
- Enable autospawn: `PICOCLAW_PINCHTAB_AUTOSPAWN=1`

### JavaScript not executing
- Verify `window.picoclaw` is defined (check browser console)
- Ensure `picoclaw-api.js` is loaded
- Check drawingboard instance is initialized

### Annotations not appearing
- Verify coordinates are in 0-1 range
- Check color format (hex or named)
- Ensure `window.picoclaw.init()` was called

## Next Steps

See `A1Reference/planning.checklist.md` for:
- D3: First annotation automation
- D4: Librarian contract + Hard-Link behavior
- VIN-keyed sessions for machine-specific memory
