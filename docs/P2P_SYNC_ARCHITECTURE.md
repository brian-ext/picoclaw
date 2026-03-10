# P2P Sync Architecture - Decentralized Knowledge Sharing

## Overview

P2P Garage OS needs to share **validated repair knowledge** across devices without relying on centralized git repositories. This enables mechanics to benefit from collective wisdom while keeping private session data local.

---

## The Problem

**Current State:**
```
Device A (Garage 1)
└── workspace/
    ├── sessions.db          ← Private (stays local)
    ├── machines/{VIN}/      ← Private (stays local)
    └── vector_db/           ← Partially shareable

Device B (Garage 2)
└── workspace/
    ├── sessions.db          ← Different data
    ├── machines/{VIN}/      ← Different machines
    └── vector_db/           ← Missing Device A's knowledge
```

**Problem:** Device B can't benefit from Device A's validated hacks and discoveries.

---

## What Should Sync vs Stay Local

### ✅ **Sync Across P2P Network**

**1. Validated Hacks (High Confidence)**
```sql
-- From sessions.db
SELECT * FROM validated_hacks 
WHERE validation_count >= 3 
  AND shared_to_network = TRUE
```

**Example:**
```json
{
  "hack_id": "hack_buick_1999_bracket_trim",
  "vin": "1G4HP54K9XH123456",
  "make": "Buick",
  "model": "LeSabre", 
  "year": 1999,
  "description": "Trim 2mm from bracket edge to avoid dash removal during alternator replacement",
  "validation_count": 5,
  "created_at": "2024-03-10T14:30:00Z",
  "validated_by": ["garage_a", "garage_b", "garage_c"]
}
```

**2. Generic Repair Insights (De-identified)**
```json
{
  "make": "Honda",
  "model": "Civic",
  "year_range": "2012-2015",
  "component": "alternator",
  "insight": "Use 10mm deep socket for hidden bolt access",
  "confidence": "high",
  "validation_count": 8
}
```

**3. TSB Discoveries (Community-Found)**
```json
{
  "make": "Toyota",
  "model": "Camry",
  "year": 2018,
  "tsb_number": "TSB-0123456",
  "issue": "Transmission shift delay in cold weather",
  "solution": "ECU reflash available",
  "source_url": "https://example.com/tsb"
}
```

---

### 🔒 **Keep Local (Never Sync)**

**1. Private Session Data**
- Individual repair conversations
- User-specific preferences
- API keys and credentials
- Personal notes

**2. VIN-Specific Machine Profiles**
- Exact VIN numbers (privacy)
- Owner information
- Service history timestamps
- Location data

**3. Unvalidated Hacks**
- Single-user discoveries (validation_count < 3)
- Experimental techniques
- Failed attempts

---

## P2P Sync Architecture

### **Option 1: IPFS + libp2p (Recommended)**

**Why IPFS:**
- Content-addressed storage (immutable hacks)
- Built-in P2P networking
- No central server required
- Go library available

**Architecture:**
```
┌─────────────────────────────────────────────────┐
│              PicoClaw Instance A                │
├─────────────────────────────────────────────────┤
│                                                 │
│  Local Workspace                                │
│  ├── sessions.db (private)                      │
│  └── validated_hacks/ (public)                  │
│      └── hack_123.json                          │
│          ↓                                       │
│      IPFS Node                                  │
│      ├── Pin hack_123.json                      │
│      ├── Announce to DHT                        │
│      └── CID: QmXyz...                           │
│                                                 │
└─────────────────────────────────────────────────┘
                    ↓ P2P Network
┌─────────────────────────────────────────────────┐
│              PicoClaw Instance B                │
├─────────────────────────────────────────────────┤
│                                                 │
│  IPFS Node                                      │
│  ├── Discover CID: QmXyz...                     │
│  ├── Fetch hack_123.json                        │
│  └── Verify signature                           │
│      ↓                                           │
│  Local Workspace                                │
│  └── validated_hacks/ (synced)                  │
│      └── hack_123.json                          │
│                                                 │
└─────────────────────────────────────────────────┘
```

**Implementation:**
```go
// pkg/p2p/ipfs_sync.go
package p2p

import (
    "github.com/ipfs/go-ipfs-api"
    "github.com/libp2p/go-libp2p"
)

type IPFSSync struct {
    node   *ipfs.Shell
    pubsub *libp2p.PubSub
}

func (s *IPFSSync) PublishHack(hack *ValidatedHack) (string, error) {
    // 1. Serialize hack to JSON
    data, _ := json.Marshal(hack)
    
    // 2. Add to IPFS (returns CID)
    cid, err := s.node.Add(bytes.NewReader(data))
    
    // 3. Pin locally
    s.node.Pin(cid)
    
    // 4. Announce to pubsub topic
    s.pubsub.Publish("picoclaw-hacks", []byte(cid))
    
    return cid, nil
}

func (s *IPFSSync) SubscribeToHacks() {
    sub, _ := s.pubsub.Subscribe("picoclaw-hacks")
    
    for {
        msg, _ := sub.Next(context.Background())
        cid := string(msg.Data)
        
        // Fetch and verify hack
        data, _ := s.node.Cat(cid)
        hack := &ValidatedHack{}
        json.Unmarshal(data, hack)
        
        // Import if valid
        if hack.ValidationCount >= 3 {
            s.ImportHack(hack)
        }
    }
}
```

---

### **Option 2: Custom P2P Protocol (Lightweight)**

**Why Custom:**
- Smaller binary size
- Full control over sync logic
- No IPFS dependency

**Architecture:**
```
┌─────────────────────────────────────────────────┐
│              PicoClaw Instance A                │
├─────────────────────────────────────────────────┤
│                                                 │
│  P2P Node (TCP/UDP)                             │
│  ├── Listen on port 7777                        │
│  ├── Maintain peer list                         │
│  └── Broadcast validated hacks                  │
│      ↓                                           │
│  Sync Protocol                                  │
│  ├── ANNOUNCE: New hack available               │
│  ├── REQUEST: Fetch hack by ID                  │
│  └── VERIFY: Signature check                    │
│                                                 │
└─────────────────────────────────────────────────┘
                    ↓ Direct P2P
┌─────────────────────────────────────────────────┐
│              PicoClaw Instance B                │
├─────────────────────────────────────────────────┤
│                                                 │
│  P2P Node (TCP/UDP)                             │
│  ├── Receive ANNOUNCE                           │
│  ├── Send REQUEST                               │
│  └── Verify and import                          │
│                                                 │
└─────────────────────────────────────────────────┘
```

**Protocol Messages:**
```json
// ANNOUNCE
{
  "type": "announce",
  "hack_id": "hack_buick_1999_bracket_trim",
  "make": "Buick",
  "model": "LeSabre",
  "year": 1999,
  "validation_count": 5,
  "timestamp": "2024-03-10T14:30:00Z"
}

// REQUEST
{
  "type": "request",
  "hack_id": "hack_buick_1999_bracket_trim"
}

// RESPONSE
{
  "type": "response",
  "hack_id": "hack_buick_1999_bracket_trim",
  "data": { /* full hack JSON */ },
  "signature": "0x123abc..."
}
```

---

### **Option 3: Hybrid (IPFS + Local Sync)**

**Best of both worlds:**
- IPFS for global discovery
- Direct P2P for local garage network
- Fallback to manual export/import

**Use Cases:**
- **Global:** Discover hacks from worldwide mechanics
- **Local:** Fast sync within same garage/shop
- **Offline:** Export/import via USB drive

---

## Data Schema for Sync

### **Validated Hack Format**
```json
{
  "hack_id": "hack_buick_1999_bracket_trim",
  "version": 1,
  "created_at": "2024-03-10T14:30:00Z",
  "updated_at": "2024-03-15T10:00:00Z",
  
  "vehicle": {
    "make": "Buick",
    "model": "LeSabre",
    "year": 1999,
    "vin_pattern": "1G4HP54K*",  // Wildcard for privacy
    "build_date_range": "1999-10-01 to 2000-12-31"
  },
  
  "repair": {
    "component": "alternator",
    "issue": "Difficult access due to bracket",
    "solution": "Trim 2mm from bracket edge",
    "tools_required": ["file", "10mm deep socket"],
    "time_saved": "30 minutes"
  },
  
  "validation": {
    "count": 5,
    "garages": ["garage_a", "garage_b", "garage_c"],
    "confidence": "high",
    "verified_by_librarian": true,
    "manual_citation": "1999 Buick LeSabre Service Manual, p.5-12"
  },
  
  "metadata": {
    "language": "en",
    "region": "US",
    "tags": ["alternator", "bracket", "time-saver"],
    "difficulty": "intermediate"
  },
  
  "signature": {
    "algorithm": "ed25519",
    "public_key": "0xabc123...",
    "signature": "0xdef456..."
  }
}
```

---

## Conflict Resolution

### **Scenario 1: Same Hack, Different Validation Counts**

**Device A:** validation_count = 3  
**Device B:** validation_count = 5

**Resolution:** Merge and use higher count
```json
{
  "validation_count": 5,
  "garages": ["a", "b", "c", "d", "e"]  // Union of validators
}
```

---

### **Scenario 2: Contradictory Hacks**

**Device A:** "Trim 2mm from bracket"  
**Device B:** "Trim 5mm from bracket"

**Resolution:** Keep both, mark as variants
```json
{
  "hack_id": "hack_buick_1999_bracket_trim",
  "variants": [
    {
      "description": "Trim 2mm",
      "validation_count": 3,
      "preferred": true
    },
    {
      "description": "Trim 5mm",
      "validation_count": 2,
      "preferred": false
    }
  ]
}
```

---

### **Scenario 3: Outdated Information**

**Device A:** Last sync 30 days ago  
**Device B:** Fresh data

**Resolution:** Use timestamp + version
```json
{
  "version": 2,
  "updated_at": "2024-03-15T10:00:00Z",
  "supersedes": "version 1"
}
```

---

## Privacy & Security

### **Privacy Protection**

**1. VIN Anonymization**
```
Real VIN: 1G4HP54K9XH123456
Shared:   1G4HP54K*  (wildcard last 8 digits)
```

**2. Location Removal**
```json
{
  "garage_id": "hash(garage_name + salt)",  // Anonymous ID
  "location": null  // Never shared
}
```

**3. User Opt-In**
```json
{
  "p2p_sync": {
    "enabled": true,
    "share_validated_hacks": true,
    "share_machine_profiles": false,  // Keep VINs private
    "require_validation_count": 3
  }
}
```

---

### **Security Measures**

**1. Digital Signatures**
- Each hack signed with garage's private key
- Prevents tampering
- Verifiable authenticity

**2. Reputation System**
```json
{
  "garage_id": "garage_a",
  "reputation_score": 95,
  "total_contributions": 50,
  "validated_contributions": 45,
  "rejected_contributions": 5
}
```

**3. Content Moderation**
- Community flagging
- Automatic rejection of low-reputation sources
- Manual review for disputed hacks

---

## Implementation Phases

### **Phase 1: Local Export/Import (MVP)**
```bash
# Export validated hacks
picoclaw export-hacks --min-validations 3 --output hacks.json

# Import from another garage
picoclaw import-hacks --input hacks.json --verify
```

**Benefits:**
- No network required
- USB drive transfer
- Full control

---

### **Phase 2: Local Network Sync**
```bash
# Start P2P node on local network
picoclaw p2p start --local-only

# Discover peers
picoclaw p2p peers

# Sync with peer
picoclaw p2p sync --peer 192.168.1.100
```

**Benefits:**
- Fast sync within garage
- No internet required
- Automatic discovery

---

### **Phase 3: Global P2P Network**
```bash
# Join global network
picoclaw p2p start --global

# Subscribe to topics
picoclaw p2p subscribe --make Buick --model LeSabre

# Publish hack
picoclaw p2p publish --hack-id hack_123
```

**Benefits:**
- Worldwide knowledge sharing
- Automatic updates
- Decentralized

---

## Database Schema Updates

### **Add P2P Sync Tables**
```sql
-- Track synced hacks
CREATE TABLE synced_hacks (
  hack_id TEXT PRIMARY KEY,
  source_garage_id TEXT,
  synced_at TIMESTAMP,
  ipfs_cid TEXT,
  signature TEXT,
  verified BOOLEAN,
  FOREIGN KEY (hack_id) REFERENCES validated_hacks(hack_id)
);

-- Track peer garages
CREATE TABLE peer_garages (
  garage_id TEXT PRIMARY KEY,
  public_key TEXT,
  reputation_score INTEGER,
  last_seen TIMESTAMP,
  total_contributions INTEGER
);

-- Track sync status
CREATE TABLE sync_log (
  sync_id TEXT PRIMARY KEY,
  sync_type TEXT,  -- 'export', 'import', 'p2p'
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  hacks_synced INTEGER,
  errors TEXT
);
```

---

## Configuration

### **config.json**
```json
{
  "p2p": {
    "enabled": false,
    "mode": "local",  // "local", "global", "hybrid"
    "port": 7777,
    "ipfs": {
      "enabled": false,
      "api_url": "http://localhost:5001"
    },
    "sync": {
      "auto_sync": true,
      "sync_interval": 3600,  // 1 hour
      "min_validation_count": 3,
      "share_machine_profiles": false,
      "share_validated_hacks": true
    },
    "privacy": {
      "anonymize_vins": true,
      "remove_location": true,
      "require_signature": true
    }
  }
}
```

---

## Future Enhancements

### **1. Blockchain for Validation**
- Immutable validation records
- Transparent reputation system
- Incentive mechanism (tokens for contributions)

### **2. Machine Learning**
- Detect patterns across hacks
- Suggest related repairs
- Predict common issues

### **3. Visual Sharing**
- Share whiteboard annotations
- Sync schematics with highlights
- Video tutorials (IPFS-hosted)

---

## Recommendation

**Start with Phase 1 (Export/Import):**
- Simple to implement
- No network complexity
- Proves the concept
- USB drive transfer works offline

**Then Phase 2 (Local Network):**
- Automatic sync within garage
- No internet required
- Fast and reliable

**Finally Phase 3 (Global P2P):**
- IPFS integration
- Worldwide knowledge sharing
- Full decentralization

---

## Summary

**P2P Sync enables:**
- ✅ Share validated hacks across garages
- ✅ Preserve privacy (VINs stay local)
- ✅ No central server required
- ✅ Offline-first with USB export
- ✅ Gradual rollout (local → global)

**Next steps:**
1. Implement export/import commands
2. Add P2P sync tables to sessions.db
3. Create hack signing/verification
4. Test with 2 garage instances
5. Document for users

**The P2P network makes P2P Garage OS truly peer-to-peer!** 🚀
