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
- Token efficiency: 800 tokens/page via PinchTab extraction vs 10,000+ for screenshots

### ChromaDB vs Pinecone
- [x] **Decision: ChromaDB** for offline/field use (garage environments)
- [ ] Pinecone only if cloud deployment with reliable connectivity

### Implementation steps
- [x] Define the Librarian tool result contract (structured output):
  - [x] `snippet` - exact text excerpt
  - [x] `source_manual` - manual name/title
  - [x] `page_number` - page or section
  - [x] `source_path_or_url` - file path
  - [x] `machine_id` (VIN) - vehicle identifier
  - [x] `confidence` - high/medium/low
  - [x] `notes` - optional context
- [x] V1 approach: Stub Librarian (interface-only) ✓
- [ ] V2: Implement ChromaDB vector search
  - [ ] Install ChromaDB Go client or use HTTP API
  - [ ] Create manual ingestion pipeline (PDF → chunks → embeddings)
  - [ ] Implement semantic search in Librarian tool
  - [ ] Add embedding model (sentence-transformers or OpenAI)
- [ ] Manual ingestion workflow:
  - [ ] PDF extraction (text + images)
  - [ ] Chunk by section/page (maintain context)
  - [ ] Generate embeddings
  - [ ] Store with metadata (VIN, year, make, model, page)
  - [ ] Index for fast retrieval

## 6) "Hard-Link" Rule (No Source, No Answer)

**Requirement**: If no verified manual page/source exists, the agent must refuse to give the spec.

- [x] Encode the rule into agent identity (Soul/Identity files) 
- [x] Define what counts as a "spec requiring citation": 
  - [x] torque values
  - [x] wiring colors/pinouts
  - [x] fuse ratings/locations
  - [x] fluid capacities
  - [x] critical safety steps (battery disconnect, jack stands, etc.)
- [ ] Add runtime enforcement (future enhancement):
  - [ ] detect "spec-like outputs" in LLM response
  - [ ] require a Librarian citation present in the same turn/session
  - [ ] otherwise force a self-correction loop or refusal

## 7) VIN-Keyed Sessions + Memory Tiers (Machine-Specific Learning)

**Goal**: store repair history and confirmed variations per machine.

- [x] Define `machine_id` keying scheme: 
  - [x] VIN if available (ISO 3779 validation)
  - [x] Generated machine ID for non-vehicles (MAKE-MODEL-YEAR-SERIAL)
- [x] Store sessions per machine (VIN-keyed): 
  - [x] SQLite database (`workspace/sessions.db`)
  - [x] `machines/` directory per VIN
  - [x] Session JSON with insights tracking
- [x] Define memory tiers: 
  - [x] Tier 1: manuals/TSBs in vector DB (ChromaDB - to be implemented)
  - [x] Tier 2: machine profile facts (`MACHINE_PROFILE.md` per VIN)
  - [x] Tier 3: session logs (SQLite + JSON)
- [x] Implement session database (pkg/session/database.go) 
- [x] Implement VIN validation utilities (pkg/session/vin.go) 
- [x] Create MACHINE_PROFILE.md template 
- [ ] Integrate session DB with agent loop
- [ ] Add P2P validated hack system
  - [x] Database schema 
  - [ ] Validation workflow (3+ confirmations)
  - [ ] Network sync protocol (future)

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

### D1 — Gateway hosts whiteboard ✅
- [x] `/whiteboard` loads from Gateway (`18790`) with embedded assets
- [x] Whiteboard JS API for AI control (`window.picoclaw`)
- [x] Routes registered in Gateway

### D2 — PinchTab can drive the whiteboard (READY FOR TESTING)
- [x] PinchTab tool wrapper created and registered
- [x] PinchTab API endpoints verified and fixed
- [x] Default port corrected (9867)
- [x] Navigate endpoint fixed (/navigate)
- [x] Comprehensive test guide created (docs/D2_D3_TESTING_GUIDE.md)
- [ ] **Execute on test machine:** PinchTab navigates to `/whiteboard` and captures screenshot
- [ ] **Requires:** Test machine with Chrome/Chromium

### D3 — First annotation automation (READY FOR TESTING)
- [x] Whiteboard JS API implemented (highlightRect, highlightCircle, addText, clear)
- [x] PinchTab evaluate action implemented
- [x] Test scenarios documented
- [ ] **Execute on test machine:** PinchTab triggers deterministic highlight and verifies via screenshot
- [ ] **Depends on:** D2 completion

### D4 — Librarian contract + Hard-Link behavior ✅
- [x] Librarian tool schema exists (stubbed)
- [x] Agent identity refuses specs without citation
- [x] Hard-Link rule encoded in IDENTITY.md

### D5 — VIN-Keyed Sessions + Database (NEW) ✅
- [x] SQLite session database implemented
- [x] VIN validation and decoding
- [x] MACHINE_PROFILE.md per VIN
- [x] Multi-tier memory architecture documented
- [x] P2P validated hack schema

### D6 — Vector Search Implementation (READY FOR TESTING)
- [x] ChromaDB integration (direct Go embedding)
- [x] Manual ingestion pipeline (PDF → chunks → embeddings)
- [x] Semantic search in Librarian tool
- [x] Ingestion CLI command created
- [x] Documentation complete (VECTOR_SEARCH_SETUP.md)
- [ ] **Execute:** Test with real repair manual PDF
- [ ] **Execute:** Verify semantic search returns relevant results

### D7 — Web Sources Integration (READY FOR TESTING)
- [x] charm.li adapter (1982-2013 manuals)
- [x] PinchTab on-demand fetching (~800 tokens/page)
- [x] Hybrid search (local ChromaDB + web sources)
- [x] Caching strategy (store fetched pages in ChromaDB)
- [x] Rate limiting and respectful scraping
- [x] Documentation complete (WEB_SOURCES_INTEGRATION.md)
- [ ] **Execute:** Test with charm.li queries (1982-2013 vehicles)
- [ ] **Execute:** Verify caching to ChromaDB works

---

## Open Decisions

- [x] **PinchTab startup:** Manual start (auto-spawn optional future feature)
- [x] **Storage choice:** ChromaDB (local-first for offline garage use)
- [x] **Machine key for non-vehicles:** MAKE-MODEL-YEAR-SERIAL format
- [x] **Embedding model:** ONNX Runtime (sentence-transformers all-MiniLM-L6-v2)
- [x] **Manual sources:** charm.li (1982-2013), local PDFs, TSBs (future), forums (future)
- [ ] **P2P sync protocol:** Future consideration (validated hacks sharing)
- [ ] **Additional web sources:** Forums, TSB databases, OEM sites
