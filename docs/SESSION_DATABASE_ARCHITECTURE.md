# Session Database Architecture - P2P Learning System

## Overview

P2P Garage OS uses a **multi-tier memory system** to gather insights from repairs and become smarter over time. As users make repairs, the system learns machine-specific variations, mid-year changes, and validated "hacks" that improve future repair sessions.

## Design Goals (from planning.md)

1. **Token Efficiency**: Don't overload context window with entire manuals
2. **Machine-Specific Memory**: Track exact manufacturer changes per VIN
3. **Self-Learning**: Gather and retain session memories for future use
4. **P2P Knowledge Sharing**: Validated repair insights shared across network
5. **Precision**: "Hard-Link" requirement - never guess critical specs

---

## Three-Tier Memory Architecture

### **Tier 1: Vector Database (Global Source of Truth)**

**Purpose:** Semantic search of repair manuals and TSBs

**Technology Options:**
- **ChromaDB** (recommended for offline/field use)
  - Local-first, no cloud dependency
  - Lightweight, runs on $10 hardware
  - Perfect for garage/field environments
  
- **Pinecone** (cloud-managed alternative)
  - Requires reliable connectivity
  - Managed service, less local maintenance
  - Not ideal for offline garages

**Storage:**
- Indexed shop manuals (PDFs)
- Technical Service Bulletins (TSBs)
- OEM documentation
- Wiring diagrams and schematics

**Access Pattern:**
- Librarian tool performs semantic search
- Returns exact snippet + source citation
- 800 tokens/page extraction via PinchTab
- **Never** loads entire manual into context

**Example Query:**
```
Query: "oil drain plug torque 2015 Honda Civic"
Result: 
  Snippet: "25-30 ft-lbs (34-41 Nm)"
  Source: 2015 Honda Civic Service Manual
  Page: 8-12
  Section: Engine Oil - Drain and Refill
```

---

### **Tier 2: Machine Profile (VIN-Keyed Facts)**

**Purpose:** Store confirmed, machine-specific characteristics

**File Structure:**
```
~/.picoclaw/workspace/machines/{VIN}/MACHINE_PROFILE.md
```

**Example: 1999 Buick LeSabre**
```markdown
# Machine Profile

## Identification
- VIN: 1G4HP54K9XH123456
- Make: Buick
- Model: LeSabre
- Year: 1999
- VIN 7th Digit: K (3800 Series II V6)
- Build Date: 10/15/1999

## Mid-Year Changes
- **Manufacturing Date: 10/01/1999 - 2005**
  - Updated bracket design (no dash removal required)
  - New alternator mount (part #12345678)
  - Revised wiring harness connector (green vs blue)

## Confirmed Variations
- Has optional towing package
- Factory remote start installed
- Non-standard battery location (trunk mount)

## Repair History
- 2024-03-01: Oil change, confirmed 5W-30 synthetic
- 2024-03-05: Alternator replacement, used updated bracket
- 2024-03-10: Starter motor, verified trunk battery disconnect

## Validated Hacks
- Trim 2mm from bracket edge to avoid dash removal (verified by 3 users)
- Use 10mm deep socket for hidden bolt access
```

**Update Pattern:**
- AI writes to this file during repair sessions
- Facts confirmed through Librarian citations
- User-discovered variations added with validation
- P2P network can sync validated hacks

---

### **Tier 3: Session Logs (Conversation History)**

**Purpose:** Detailed repair session history per machine

**File Structure:**
```
~/.picoclaw/workspace/machines/{VIN}/sessions/{session_id}.json
```

**Schema:**
```json
{
  "session_id": "sess_2024-03-10_14-30-00",
  "vin": "1G4HP54K9XH123456",
  "machine_profile": "machines/1G4HP54K9XH123456/MACHINE_PROFILE.md",
  "started_at": "2024-03-10T14:30:00Z",
  "completed_at": "2024-03-10T16:45:00Z",
  "repair_type": "alternator_replacement",
  "tools_used": ["librarian", "pinchtab", "web_search"],
  "manuals_referenced": [
    {
      "source": "1999 Buick LeSabre Service Manual",
      "pages": ["5-12", "5-13", "8-45"],
      "citations": 3
    }
  ],
  "conversation_summary": "User replaced alternator. Confirmed mid-year bracket change. Discovered trunk battery disconnect required. Added validated hack for bracket trimming.",
  "insights_learned": [
    "Trunk battery disconnect required (not in manual)",
    "Updated bracket fits without dash removal",
    "10mm deep socket required for hidden bolt"
  ],
  "whiteboard_snapshots": [
    "sessions/sess_2024-03-10_14-30-00/whiteboard_001.png",
    "sessions/sess_2024-03-10_14-30-00/whiteboard_002.png"
  ],
  "safety_checks_performed": [
    "Battery disconnect confirmed",
    "Jack stands verified",
    "Torque spec cited (65 ft-lbs from manual p.5-13)"
  ],
  "p2p_contributions": [
    {
      "type": "validated_hack",
      "description": "Bracket trim eliminates dash removal",
      "validation_count": 1,
      "shared_to_network": false
    }
  ]
}
```

**Session Management:**
- Auto-summarize after 20 messages or 75% token limit
- Preserve key insights in summary
- Link to machine profile for context injection
- Store whiteboard snapshots for future reference

---

## Context Injection Strategy

**On Session Start:**
1. Load `IDENTITY.md` (P2P Garage OS persona)
2. Load `SOUL.md` (Safety-First behavior)
3. Load `USER.md` (user preferences)
4. **If VIN provided:**
   - Load `machines/{VIN}/MACHINE_PROFILE.md`
   - Load last 3 session summaries for this VIN
   - Inject machine-specific facts into context

**During Repair:**
1. Librarian searches Vector DB (Tier 1)
2. Returns exact snippet + citation
3. AI confirms against machine profile (Tier 2)
4. Updates session log (Tier 3)
5. Writes new facts to machine profile if discovered

**Token Budget:**
- IDENTITY + SOUL + USER: ~2,000 tokens
- Machine Profile: ~1,000 tokens
- Session summaries (3): ~1,500 tokens
- Librarian snippets: ~800 tokens/query
- **Total context overhead: ~5,300 tokens**
- Leaves ~2,700 tokens for conversation (8K model)

---

## P2P Learning & Knowledge Sharing

### **Validated Hack Workflow**

1. **User discovers variation** (e.g., "I trimmed the bracket 2mm")
2. **AI records in session log** with validation_count: 1
3. **AI writes to machine profile** as "Unvalidated Hack"
4. **Other users encounter same machine/repair:**
   - AI suggests the hack
   - User confirms it worked
   - validation_count increments
5. **After 3 validations:**
   - Marked as "Validated Hack"
   - Eligible for P2P network sharing
6. **P2P sync** (future):
   - Validated hacks shared across network
   - Other users benefit from collective knowledge

### **Safety Enforcement**

**Hard-Link Rule:**
- Critical specs (torque, wiring, fuses) **require** Librarian citation
- If no citation found, AI **refuses** to provide spec
- Session log tracks all citations for audit

**Verification Checkpoints:**
- Battery disconnect (before electrical work)
- Jack stands (before going under vehicle)
- Torque specs (must cite manual page)
- Mid-year changes (check VIN + build date)

---

## Database Technology Decisions

### **Vector Database: ChromaDB (Recommended)**

**Pros:**
- ✅ Local-first (offline garage use)
- ✅ Lightweight (~50MB memory footprint)
- ✅ Python/Go bindings available
- ✅ Runs on $10 hardware
- ✅ No cloud dependency

**Cons:**
- ⚠️ Manual backup/sync required
- ⚠️ No built-in P2P sync

**Use Case:** Single-user garage, field repairs, offline environments

### **Vector Database: Pinecone (Alternative)**

**Pros:**
- ✅ Managed service (less maintenance)
- ✅ Built-in scaling
- ✅ Cloud sync across devices

**Cons:**
- ❌ Requires internet connectivity
- ❌ Monthly cost
- ❌ Not suitable for offline use

**Use Case:** Multi-location shops, cloud-first deployments

### **Session Storage: SQLite (Recommended)**

**Pros:**
- ✅ Single file database
- ✅ ACID transactions
- ✅ Built into Go stdlib
- ✅ Perfect for VIN-keyed sessions
- ✅ Efficient queries

**Schema:**
```sql
CREATE TABLE machines (
  vin TEXT PRIMARY KEY,
  make TEXT,
  model TEXT,
  year INTEGER,
  build_date TEXT,
  profile_path TEXT,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);

CREATE TABLE sessions (
  session_id TEXT PRIMARY KEY,
  vin TEXT,
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  repair_type TEXT,
  conversation_summary TEXT,
  insights_json TEXT,
  FOREIGN KEY (vin) REFERENCES machines(vin)
);

CREATE TABLE validated_hacks (
  hack_id TEXT PRIMARY KEY,
  vin TEXT,
  description TEXT,
  validation_count INTEGER,
  shared_to_network BOOLEAN,
  created_at TIMESTAMP,
  FOREIGN KEY (vin) REFERENCES machines(vin)
);

CREATE TABLE citations (
  citation_id TEXT PRIMARY KEY,
  session_id TEXT,
  source TEXT,
  page TEXT,
  snippet TEXT,
  created_at TIMESTAMP,
  FOREIGN KEY (session_id) REFERENCES sessions(session_id)
);
```

---

## Implementation Roadmap

### **Phase 1: Foundation (Current)**
- ✅ Librarian tool stub
- ✅ Hard-Link rule in IDENTITY.md
- ⏭️ SQLite session database
- ⏭️ VIN-keyed machine profiles

### **Phase 2: Vector Search**
- ⏭️ ChromaDB integration
- ⏭️ Manual ingestion pipeline
- ⏭️ Semantic search in Librarian tool
- ⏭️ Citation tracking

### **Phase 3: P2P Learning**
- ⏭️ Validated hack system
- ⏭️ Session insight extraction
- ⏭️ Machine profile auto-updates
- ⏭️ P2P network sync (optional)

### **Phase 4: Advanced**
- ⏭️ Vision model for whiteboard analysis
- ⏭️ Photo verification for mid-year changes
- ⏭️ Multimodal validation
- ⏭️ Cross-machine pattern detection

---

## File Structure

```
~/.picoclaw/workspace/
├── IDENTITY.md              # P2P Garage OS persona
├── SOUL.md                  # Safety-First behavior
├── USER.md                  # User preferences
├── MEMORY.md                # General long-term facts
├── sessions.db              # SQLite database (all sessions)
├── vector_db/               # ChromaDB storage
│   ├── manuals/
│   ├── tsbs/
│   └── index/
└── machines/
    ├── 1G4HP54K9XH123456/   # VIN-keyed directory
    │   ├── MACHINE_PROFILE.md
    │   └── sessions/
    │       ├── sess_2024-03-10_14-30-00.json
    │       └── whiteboard_snapshots/
    └── 5FNRL5H40GB123456/   # Another VIN
        ├── MACHINE_PROFILE.md
        └── sessions/
```

---

## Next Steps

1. **Implement SQLite session database** in `pkg/session/`
2. **Create MACHINE_PROFILE.md template**
3. **Add VIN extraction/validation logic**
4. **Integrate ChromaDB for vector search**
5. **Build manual ingestion pipeline**
6. **Test with real repair session**

---

**Key Insight:** This architecture ensures P2P Garage OS learns from every repair, becoming smarter and more precise over time while maintaining token efficiency and safety-first behavior.
