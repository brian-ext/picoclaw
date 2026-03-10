# Web Sources Integration - charm.li & Beyond

## Overview

Instead of downloading 700GB of manuals from charm.li, P2P Garage OS uses **on-demand fetching** via PinchTab. This provides:
- **Token efficiency**: ~800 tokens/page vs 10,000+ for screenshots
- **No storage overhead**: Fetch only what's needed
- **Always up-to-date**: Access latest online content
- **Hybrid approach**: Local ChromaDB + web sources

## Architecture

```
User Query: "oil drain plug torque 1999 Buick LeSabre"
     │
     ▼
┌─────────────────┐
│ Librarian Tool  │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌─────────┐ ┌──────────────┐
│ Local   │ │ Web Sources  │
│ ChromaDB│ │ (charm.li)   │
└─────────┘ └──────┬───────┘
                   │
                   ▼
            ┌──────────────┐
            │  PinchTab    │
            │  - Navigate  │
            │  - Extract   │
            │  - Cache     │
            └──────────────┘
```

## charm.li Integration

### URL Structure

```
Base: https://charm.li/
Make: https://charm.li/Buick/
Model: https://charm.li/Buick/LeSabre/
Year: https://charm.li/Buick/LeSabre/1999/
```

### Supported Years

**1982 - 2013** (31 years of manuals)

### Coverage

50+ manufacturers:
- Acura, Audi, BMW, Buick, Cadillac, Chevrolet, Chrysler
- Dodge, Ford, GMC, Honda, Hyundai, Infiniti, Jeep
- Kia, Lexus, Mazda, Mercedes-Benz, Nissan, Toyota
- Volkswagen, Volvo, and more...

## Workflow

### 1. User Query

```
User: "What's the oil drain plug torque for a 1999 Buick LeSabre?"
```

### 2. Librarian Decision Tree

```
1. Check local ChromaDB
   ├─ Found? → Return local result (high confidence)
   └─ Not found? → Continue to web sources

2. Check year range
   ├─ 1982-2013? → Try charm.li
   ├─ 2014+? → Try other sources (forums, OEM sites)
   └─ Pre-1982? → Return "manual not available"

3. Fetch from charm.li
   ├─ Navigate: https://charm.li/Buick/LeSabre/1999/
   ├─ Extract: PinchTab text extraction (~800 tokens)
   ├─ Search: Find relevant snippet
   └─ Cache: Store in ChromaDB for future queries

4. Return result with citation
```

### 3. PinchTab Execution

```go
// Navigate to manual
pinchtab.Execute(ctx, map[string]any{
    "action": "navigate",
    "url": "https://charm.li/Buick/LeSabre/1999/",
})

// Extract text (token-efficient!)
result := pinchtab.Execute(ctx, map[string]any{
    "action": "text",
})
// Returns ~800 tokens of text content
```

### 4. Caching Strategy

**First query:** Fetch from web (slow)
```
Query: "oil drain plug torque 1999 Buick"
→ Fetch from charm.li (2-3 seconds)
→ Cache in ChromaDB
→ Return result
```

**Subsequent queries:** Use cached data (fast)
```
Query: "alternator replacement 1999 Buick"
→ Check ChromaDB (50ms)
→ Found cached charm.li page
→ Return result
```

## Token Efficiency

### Comparison

| Method | Tokens | Cost (GPT-4) | Speed |
|--------|--------|--------------|-------|
| **PinchTab text** | ~800 | $0.024 | Fast |
| Vision (screenshot) | ~10,000 | $0.30 | Slow |
| Full PDF download | ~100,000+ | $3.00+ | Very slow |

### Example: 1999 Buick LeSabre Manual

**Full download approach:**
- Size: 50MB PDF
- Pages: 500
- Tokens if fully loaded: ~500,000
- Cost: $15.00 per query

**PinchTab approach:**
- Navigate to relevant page: 1 second
- Extract text: ~800 tokens
- Cost: $0.024 per query
- **625x cheaper!**

## Implementation

### Enable Web Sources

In `config.json`:
```json
{
  "tools": {
    "librarian": {
      "enabled": true,
      "web_sources": {
        "enabled": true,
        "charm_li": true,
        "cache_fetched": true
      }
    },
    "pinchtab": {
      "enabled": true
    }
  }
}
```

### Usage Example

```go
// Librarian automatically tries web sources if local search fails
result := librarian.Execute(ctx, map[string]any{
    "query": "oil drain plug torque",
    "machine_id": "1G4HP54K9XH123456", // 1999 Buick LeSabre
    "max_results": 3,
})

// Result includes web source citation:
// "Source: 1999 Buick LeSabre Service Manual (charm.li)"
// "URL: https://charm.li/Buick/LeSabre/1999/"
// "Confidence: medium (web source)"
```

## Future Web Sources

### Forums & Communities

**mechanicadvice (Reddit)**
- URL: reddit.com/r/MechanicAdvice
- Content: User-reported issues, solutions
- Confidence: Low (requires validation)

**justanswer.com**
- URL: justanswer.com/car
- Content: Expert Q&A
- Confidence: Medium

### TSB Databases

**NHTSA (National Highway Traffic Safety Administration)**
- URL: nhtsa.gov
- Content: Safety recalls, TSBs
- Confidence: High (official)

### OEM Websites

**Honda Service Express**
- URL: techinfo.honda.com
- Content: Official service info
- Confidence: High (requires subscription)

**Toyota TIS**
- URL: techinfo.toyota.com
- Content: Technical information
- Confidence: High (requires subscription)

## Hard-Link Compliance

**Web sources MUST include:**
1. Source URL
2. Fetch timestamp
3. Confidence level (medium by default)
4. Warning: "Web source - verify when possible"

**Example citation:**
```
Oil drain plug torque: 25-30 ft-lbs (34-41 Nm)

Source: 1999 Buick LeSabre Service Manual (charm.li)
URL: https://charm.li/Buick/LeSabre/1999/
Fetched: 2026-03-10 14:30:00 UTC
Confidence: Medium (web source)

⚠️ Web source - verify against local manual when available
```

## Caching Policy

### What to Cache

✅ **Cache:**
- Frequently accessed pages
- Complete manual sections
- TSB content
- Verified forum solutions

❌ **Don't Cache:**
- User-generated content (forums)
- Time-sensitive data (recalls)
- Subscription-only content

### Cache Expiration

- **charm.li pages:** 30 days (manuals don't change)
- **Forum posts:** 7 days (may be updated)
- **TSBs:** 90 days (occasionally updated)
- **OEM sites:** 14 days (subscription content)

### Cache Storage

Cached web content stored in ChromaDB with metadata:
```json
{
  "source_type": "web",
  "web_source": "charm.li",
  "fetched_at": "2026-03-10T14:30:00Z",
  "cache_expires": "2026-04-09T14:30:00Z",
  "url": "https://charm.li/Buick/LeSabre/1999/"
}
```

## Rate Limiting

### charm.li

**Respectful scraping:**
- Max 1 request per second
- User-Agent: "PicoClaw-P2P-Garage-OS/0.2.0"
- Cache aggressively to minimize requests

### Implementation

```go
// Rate limiter in PinchTab tool
type RateLimiter struct {
    lastRequest time.Time
    minInterval time.Duration
}

func (r *RateLimiter) Wait() {
    elapsed := time.Since(r.lastRequest)
    if elapsed < r.minInterval {
        time.Sleep(r.minInterval - elapsed)
    }
    r.lastRequest = time.Now()
}
```

## Error Handling

### charm.li Unavailable

```
1. Try cached version (if available)
2. Try alternative sources
3. Return: "Manual temporarily unavailable - try local sources"
```

### Page Not Found (404)

```
1. Check year range (1982-2013)
2. Try alternate model names
3. Return: "Manual not found on charm.li - try local ingestion"
```

### Network Timeout

```
1. Retry once (after 2 seconds)
2. Fall back to cached version
3. Return: "Network error - using cached data"
```

## Privacy & Ethics

### User Privacy

- ✅ No user data sent to charm.li
- ✅ Queries are anonymous
- ✅ No tracking cookies

### Respectful Use

- ✅ Rate limiting (1 req/sec)
- ✅ Proper User-Agent
- ✅ Cache to minimize load
- ✅ No bulk downloading

### Attribution

- ✅ Always cite charm.li as source
- ✅ Include URL in citations
- ✅ Acknowledge "Operation CHARM"

## Testing

### Test charm.li Integration

```bash
# Build with web sources
go build -o picoclaw.exe ./cmd/picoclaw

# Start PinchTab
pinchtab server

# Start Gateway
picoclaw gateway

# Test query (1999 Buick - in charm.li range)
# Ask AI: "What's the oil drain plug torque for a 1999 Buick LeSabre?"

# Expected: Librarian fetches from charm.li, caches result
```

### Verify Caching

```bash
# First query (slow - fetches from web)
time: 2-3 seconds

# Second query (fast - uses cache)
time: 50-100ms
```

## Summary

**Hybrid Librarian Strategy:**
1. **Local first:** Check ChromaDB (fast, high confidence)
2. **Web fallback:** Fetch from charm.li (token-efficient)
3. **Cache aggressively:** Store for future queries
4. **Always cite:** Hard-Link requirement

**Benefits:**
- No 700GB download required
- Token-efficient (~800 tokens/page)
- Always up-to-date content
- Scales to unlimited manuals

**Ready for testing with real queries!**
