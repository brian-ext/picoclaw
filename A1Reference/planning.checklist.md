# PicoClaw → P2P Garage OS (Checklist)

This checklist consolidates the actionable decisions and next steps discussed in `A1Reference/planning.md`.

## 0) North Star (Product + Constraints)

- [ ] **Purpose**: “P2P Garage OS” repair assistant (not a general chatbot).
- [ ] **Primary flow**: Identify machine (VIN/MPN or guided identification chat) → open whiteboard → AI guides repair step-by-step.
- [ ] **Single human user** (no multi-user realtime required for v1).
- [ ] **Token efficiency is a requirement** (minimize context usage, avoid screenshot/vision ingestion when plain text is available).
- [ ] **Safety-first**: enforce “No source, no answer” for specs (torque, wiring, etc.).

## 1) Repo Consolidation (3 → 1 workspace)

**Decision**: `picoclaw` is the base repo.

- [ ] Create top-level directories (initial target layout):
  - [ ] `third_party/pinchtab/` (PinchTab source imported as-is initially)
  - [ ] `web/whiteboard/` (whiteboard static assets)
- [ ] Keep **two binaries** initially (acceptable):
  - [ ] `picoclaw` (Gateway + agent)
  - [ ] `pinchtab` (sidecar browser automation)
- [ ] Decide how PinchTab runs during dev:
  - [ ] **Manual start** (simplest initially)
  - [ ] Later: **auto-spawn** by Gateway (optional)
- [ ] Resolve Go toolchain strategy:
  - [ ] Keep PinchTab as its own Go module at first (avoid immediate `go.mod` merge friction)
  - [ ] Optionally add a `go.work` later for developer convenience

## 2) Extend PicoClaw Gateway (Port 18790) to Serve the OS UI

**Decision**: extend Gateway (not the separate Web Console backend).

- [ ] Add/confirm Gateway routes:
  - [ ] `GET /` landing page (VIN/MPN input + “Don’t have info?” link)
  - [ ] `GET /whiteboard` whiteboard entry
  - [ ] `GET /whiteboard/*` serve whiteboard static assets
- [ ] Use an embedded static file server pattern (similar to `web/backend/embed.go`).
- [ ] Keep existing health endpoints (`/health`, `/ready`) unchanged.

## 3) Whiteboard (Single-User First)

**Decision**: single-user only; multi-user realtime can be deferred.

- [ ] Import whiteboard static assets from `A1Reference/go-drawingboard/`:
  - [ ] `index.html`
  - [ ] `css/`
  - [ ] `js/`
  - [ ] optional demo pages
- [ ] Do **not** attempt GoInstant integration for v1 (treat as legacy).
- [ ] Define how the AI will annotate:
  - [ ] Add a minimal JS API on the page (example: `window.whiteboard.highlightRect(...)`)
  - [ ] Ensure PinchTab can call it via `evaluate` or scripted actions

## 4) PinchTab Integration (Economical Retrieval + Whiteboard Control)

**Goal**: PinchTab is the “browser eyes + hands” layer.

- [ ] Stand up PinchTab sidecar locally:
  - [ ] Confirm `GET /health` works
  - [ ] Confirm navigation works (`POST /navigate`)
  - [ ] Confirm extraction works (`GET /text`)
  - [ ] Confirm screenshot/snapshot works (`GET /screenshot`/`GET /snapshot`)
- [ ] Build a PicoClaw Tool wrapper (HTTP client) for PinchTab:
  - [ ] `pinchtab.navigate(url)`
  - [ ] `pinchtab.text(url|tab)`
  - [ ] `pinchtab.screenshot(url|tab)`
  - [ ] `pinchtab.evaluate(js, tab)` or an `actions` wrapper (for whiteboard interaction)
- [ ] Whiteboard automation MVP:
  - [ ] Open `/whiteboard` via PinchTab
  - [ ] Apply one deterministic highlight annotation
  - [ ] Capture a screenshot for verification

## 5) Librarian + Manuals Storage (RAG)

**Key rules**:
- Manuals should be stored locally.
- Retrieval should be semantic (RAG) to avoid overloading the context window.

### ChromaDB vs Pinecone
- [ ] Prefer **local ChromaDB** when offline/field use matters.
- [ ] Pinecone only if you accept a managed cloud dependency and reliable connectivity.

### First implementation steps
- [ ] Define the Librarian tool result contract (structured output):
  - [ ] `snippet`
  - [ ] `source_manual`
  - [ ] `page_number`
  - [ ] `source_path_or_url`
  - [ ] `machine_id` (VIN)
  - [ ] optional `confidence/notes`
- [ ] Decide v1 approach:
  - [ ] Stub Librarian (interface-only) while whiteboard + PinchTab are being integrated
  - [ ] Then implement vector search + manual ingestion pipeline

## 6) “Hard-Link” Rule (No Source, No Answer)

**Requirement**: If no verified manual page/source exists, the agent must refuse to give the spec.

- [ ] Encode the rule into agent identity (Soul/Identity files) so the LLM is prompted correctly.
- [ ] Define what counts as a “spec requiring citation”:
  - [ ] torque values
  - [ ] wiring colors/pinouts
  - [ ] fuse ratings/locations
  - [ ] fluid capacities
  - [ ] critical safety steps (battery disconnect, jack stands, etc.)
- [ ] Add runtime enforcement later (not required on day 1):
  - [ ] detect “spec-like outputs”
  - [ ] require a Librarian citation present in the same turn/session
  - [ ] otherwise force a self-correction loop or refusal

## 7) VIN-Keyed Sessions + Memory Tiers (Machine-Specific Learning)

**Goal**: store repair history and confirmed variations per machine.

- [ ] Define `machine_id` keying scheme:
  - [ ] VIN if available
  - [ ] otherwise a generated machine ID
- [ ] Store sessions per machine (VIN-keyed):
  - [ ] `workspace/sessions/{VIN}.json` (or `vin:{VIN}` session key)
- [ ] Define memory tiers:
  - [ ] Tier 1: manuals/TSBs in vector DB (source of truth)
  - [ ] Tier 2: machine profile facts (`MACHINE_PROFILE.md` per VIN)
  - [ ] Tier 3: session logs (chat + tool outputs)

## 8) Machine-to-Docs Verification (Deferred but Designed-In)

**Goal**: verify physical machine state against documentation.

- [ ] Define when verification is required:
  - [ ] mid-year changes suspected
  - [ ] safety-critical steps
  - [ ] wiring/fusebox ambiguities
- [ ] Later implement “photo request” hook/tool:
  - [ ] ask for targeted photo (wire colors, connector, fusebox label)
  - [ ] compare to Librarian-retrieved diagram/spec

## 9) Minimal Deliverables (Sequence)

### D1 — Gateway hosts whiteboard
- [ ] `/whiteboard` loads from Gateway (`18790`) with embedded assets.

### D2 — PinchTab can drive the whiteboard
- [ ] PinchTab navigates to `/whiteboard` and captures a screenshot.

### D3 — First annotation automation
- [ ] PinchTab triggers a deterministic highlight and verifies via screenshot.

### D4 — Librarian contract + Hard-Link behavior (prompt-level)
- [ ] Librarian tool schema exists (even stubbed)
- [ ] Agent identity refuses specs without citation

---

## Open Decisions (Fill In)

- [ ] PinchTab startup:
  - [ ] manual start
  - [ ] auto-spawn by Gateway
- [ ] Storage choice:
  - [ ] local ChromaDB
  - [ ] Pinecone
- [ ] Machine key for non-vehicles:
  - [ ] schema TBD
