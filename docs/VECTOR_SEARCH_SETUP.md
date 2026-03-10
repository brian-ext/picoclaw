# Vector Search Setup - ChromaDB Integration

## Overview

P2P Garage OS uses **ChromaDB with direct Go embedding** for semantic search of repair manuals. This provides token-efficient retrieval (~800 tokens/page) and enables the Hard-Link requirement for verified specifications.

## Architecture

```
┌─────────────┐
│ Repair PDF  │
└──────┬──────┘
       │ Ingest
       ▼
┌─────────────────┐
│ PDF Extraction  │ (ledongthuc/pdf)
│ - Text per page │
│ - Chunking      │
└──────┬──────────┘
       │
       ▼
┌──────────────────┐
│ ChromaDB         │ (amikos-tech/chroma-go)
│ - Embeddings     │ (default: ONNX Runtime)
│ - Vector storage │
│ - Metadata index │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Librarian Tool   │
│ - Semantic search│
│ - Citation format│
│ - Hard-Link check│
└──────────────────┘
```

## Dependencies

Add to `go.mod`:
```bash
go get github.com/amikos-tech/chroma-go
go get github.com/ledongthuc/pdf
```

### Required Libraries

**ChromaDB Go Client:**
- Package: `github.com/amikos-tech/chroma-go`
- Features: Direct embedding, persistent storage, ONNX Runtime
- Docs: https://go-client.chromadb.dev/

**PDF Extraction:**
- Package: `github.com/ledongthuc/pdf`
- Pure Go PDF parser

## Installation

### 1. Install Dependencies

```bash
# Install Go packages
go get github.com/amikos-tech/chroma-go
go get github.com/ledongthuc/pdf

# Build PicoClaw
go build -o picoclaw.exe ./cmd/picoclaw
```

### 2. Initialize Workspace

The vector database is stored in `~/.picoclaw/workspace/vector_db/`:

```bash
# Workspace structure
~/.picoclaw/workspace/
├── vector_db/           # ChromaDB persistent storage
│   ├── chroma.sqlite3   # Metadata
│   └── collections/     # Vector data
├── sessions.db          # Session database
└── machines/            # VIN-keyed profiles
```

## Usage

### Ingest Repair Manuals

**Single PDF:**
```bash
picoclaw ingest \
  --make Honda \
  --model Civic \
  --year 2015 \
  --type service \
  manual.pdf
```

**Directory of PDFs:**
```bash
picoclaw ingest \
  --make Honda \
  --model Civic \
  --year 2015 \
  --type service \
  ./manuals/honda/civic/2015/
```

**VIN-Specific Manual:**
```bash
picoclaw ingest \
  --vin 1G4HP54K9XH123456 \
  --make Buick \
  --model LeSabre \
  --year 1999 \
  --type service \
  1999_buick_lesabre_service.pdf
```

### Manual Types

- `service` - Service/repair manual
- `owner` - Owner's manual
- `tsb` - Technical Service Bulletin
- `wiring` - Wiring diagram

### Enable Librarian Tool

In `config.json`:
```json
{
  "tools": {
    "librarian": {
      "enabled": true
    }
  }
}
```

### Query via LLM

Once manuals are ingested, the AI can use the Librarian tool:

**User:** "What's the oil drain plug torque for a 2015 Honda Civic?"

**AI uses tool:**
```json
{
  "tool": "librarian",
  "args": {
    "query": "oil drain plug torque",
    "machine_id": "1G4HP54K9XH123456",
    "max_results": 3
  }
}
```

**Librarian returns:**
```
Found 2 result(s) for: "oil drain plug torque"

Result 1:
  Snippet: Oil drain plug torque: 25-30 ft-lbs (34-41 Nm). Use Honda Genuine Motor Oil or equivalent...
  Source: 2015 Honda Civic Service Manual
  Page: 8-12
  Section: Engine Oil - Drain and Refill
  Path: /manuals/honda/civic/2015/service_manual.pdf
  Confidence: high
  Distance: 0.234

⚠️ HARD-LINK REQUIREMENT: You MUST cite the source when providing this information to the user.
```

**AI responds to user:**
```
Oil drain plug torque: 25-30 ft-lbs (34-41 Nm)

Source: 2015 Honda Civic Service Manual, Page 8-12, Section: Engine Oil - Drain and Refill
Confidence: High
```

## Chunking Strategy

**Default Settings:**
- Chunk size: 1,000 characters (~250 tokens)
- Overlap: 200 characters (~50 tokens)
- Boundary: Sentence-aware (breaks at `.` or `\n`)

**Why Chunking:**
- Maintains context within pages
- Prevents token overflow
- Enables precise citations
- Improves search relevance

**Example:**
```
Page 8-12 (3,500 chars) → 4 chunks:
  - Chunk 0: chars 0-1000 (overlap with chunk 1)
  - Chunk 1: chars 800-1800 (overlap both sides)
  - Chunk 2: chars 1600-2600
  - Chunk 3: chars 2400-3500
```

## Embedding Model

**Default: ONNX Runtime (sentence-transformers)**
- Model: `all-MiniLM-L6-v2`
- Dimensions: 384
- Speed: Fast (CPU-friendly)
- Quality: Good for technical text

**Alternative: OpenAI Embeddings**
```go
// In pkg/vectordb/chromadb.go
import "github.com/amikos-tech/chroma-go/pkg/embeddings/openai"

// Use OpenAI embeddings
ef := openai.NewOpenAIEmbeddingFunction(apiKey)
collection, err := client.CreateCollection(ctx, "repair_manuals", nil, true, ef, types.L2)
```

## Metadata Schema

Each chunk stores:
```json
{
  "vin": "1G4HP54K9XH123456",
  "make": "Buick",
  "model": "LeSabre",
  "year": 1999,
  "manual_type": "service",
  "manual_title": "1999 Buick LeSabre Service Manual",
  "source_path": "/manuals/buick/1999_service.pdf",
  "page_number": "8-12",
  "section": "Engine Oil",
  "chunk_index": 0,
  "total_chunks": 4
}
```

## Search Filters

**By VIN:**
```go
results, err := db.SearchByVIN(ctx, "alternator replacement", "1G4HP54K9XH123456", 5)
```

**By Make/Model/Year:**
```go
results, err := db.SearchByMakeModel(ctx, "fuse box diagram", "Honda", "Civic", 2015, 5)
```

**General Search:**
```go
results, err := db.Search(ctx, "torque specifications", nil, 5)
```

## Performance

**Ingestion:**
- ~100 pages/minute (single-threaded)
- ~1MB PDF = ~200 chunks
- Embedding generation: ~10ms/chunk (ONNX)

**Search:**
- Query time: ~50-100ms (1000 chunks)
- Scales to 100,000+ chunks
- Distance metric: L2 (Euclidean)

**Storage:**
- ~1KB per chunk (text + metadata)
- ~500KB per 100-page manual
- SQLite metadata + vector files

## Troubleshooting

### ChromaDB initialization fails

**Error:** `failed to initialize ChromaDB`

**Solution:**
```bash
# Ensure workspace directory exists
mkdir -p ~/.picoclaw/workspace/vector_db

# Check permissions
chmod 755 ~/.picoclaw/workspace
```

### ONNX Runtime not found

**Error:** `ONNX Runtime library not found`

**Solution:**
ChromaDB Go client downloads ONNX Runtime automatically on first use. If it fails:
```bash
# Set custom library path
export ONNX_LIBRARY_PATH=/path/to/onnxruntime.so
```

### PDF extraction fails

**Error:** `failed to extract PDF text`

**Solution:**
- Verify PDF is not encrypted
- Check PDF is not image-only (requires OCR)
- Try extracting a single page to test

### No search results

**Possible causes:**
1. No manuals ingested yet
2. Query too specific/generic
3. Wrong VIN filter applied

**Debug:**
```bash
# Check collection count
# In Go code:
count, _ := db.GetCollectionCount(ctx)
fmt.Printf("Total chunks: %d\n", count)
```

## Next Steps

1. **Ingest manuals** for your target vehicles
2. **Enable Librarian** in config.json
3. **Test queries** via LLM
4. **Verify citations** in responses
5. **Add more manuals** as needed

## Future Enhancements

- [ ] OCR support for image-based PDFs
- [ ] Diagram/schematic extraction
- [ ] Multi-language support
- [ ] Custom embedding models
- [ ] P2P manual sharing network
- [ ] Automatic TSB updates

---

**The vector search system is ready for D6 testing!**
