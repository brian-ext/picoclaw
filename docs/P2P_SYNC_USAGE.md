# P2P Sync - Automatic Knowledge Sharing

## Overview

P2P Sync enables automatic sharing of **validated repair hacks** across garages without manual intervention. When enabled, PicoClaw automatically exports your validated discoveries and imports knowledge from other mechanics via the heartbeat system.

---

## Quick Start

### 1. Enable P2P Sync in Config

Edit `config.json`:

```json
{
  "p2p_sync": {
    "enabled": true,
    "sync_interval": 3600,
    "min_validations": 3,
    "auto_export": true,
    "auto_import": true,
    "export_path": "~/.picoclaw/workspace/p2p_export.json",
    "import_path": "~/.picoclaw/workspace/p2p_import.json"
  },
  "heartbeat": {
    "enabled": true,
    "interval": 3600
  }
}
```

### 2. Start PicoClaw

```bash
picoclaw gateway
```

That's it! PicoClaw will now:
- ✅ Auto-export validated hacks every hour
- ✅ Auto-import hacks from other garages
- ✅ Verify all signatures cryptographically
- ✅ Track peer reputation

---

## How It Works

### Automatic Export (Every Sync Interval)

```
Your Garage
    ↓
Validated Hacks (≥3 confirmations)
    ↓
Sign with private key
    ↓
Export to p2p_export.json
    ↓
Share via USB/network/IPFS
```

### Automatic Import (Every Sync Interval)

```
Other Garage
    ↓
p2p_import.json (received via USB/network)
    ↓
Verify signature
    ↓
Check reputation
    ↓
Import to local database
    ↓
Available to AI
```

---

## Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `false` | Enable/disable P2P sync |
| `sync_interval` | `3600` | Seconds between syncs (1 hour) |
| `min_validations` | `3` | Minimum validations to share |
| `auto_export` | `true` | Automatically export hacks |
| `auto_import` | `true` | Automatically import hacks |
| `export_path` | `~/.picoclaw/workspace/p2p_export.json` | Export file location |
| `import_path` | `~/.picoclaw/workspace/p2p_import.json` | Import file location |

---

## Manual Commands

### Export Validated Hacks

```bash
picoclaw p2p export --output hacks.json --min-validations 3
```

**Output:**
```
✅ Exported 5 validated hacks to hacks.json
📊 Minimum validation count: 3
```

### Import Hacks from Another Garage

```bash
picoclaw p2p import --input hacks.json --verify
```

**Output:**
```
📥 Importing 5 hacks from hacks.json
✅ Imported: 4
❌ Rejected: 1
⚠️  Some hacks were rejected due to signature verification failures
```

---

## Sharing Methods

### Method 1: USB Drive (Offline)

**Garage A:**
```bash
# Export to USB
picoclaw p2p export --output /media/usb/hacks.json
```

**Garage B:**
```bash
# Import from USB
picoclaw p2p import --input /media/usb/hacks.json
```

---

### Method 2: Network Share (Local)

**Garage A:**
```bash
# Export to shared folder
picoclaw p2p export --output /mnt/garage_share/hacks_garage_a.json
```

**Garage B:**
```bash
# Automatic import via heartbeat
# Just copy file to import path:
cp /mnt/garage_share/hacks_garage_a.json ~/.picoclaw/workspace/p2p_import.json
```

---

### Method 3: IPFS (Global) - Future

```bash
# Publish to IPFS
picoclaw p2p publish --ipfs

# Subscribe to IPFS topic
picoclaw p2p subscribe --topic picoclaw-hacks
```

---

## Privacy & Security

### What Gets Shared

**✅ Shared:**
- Validated hacks (≥3 confirmations)
- Make, model, year
- Anonymized VIN pattern (`1G4HP54K*`)
- Repair solution
- Tool requirements
- Validation count

**🔒 Never Shared:**
- Exact VIN numbers
- Session conversations
- User information
- API keys
- Unvalidated experiments

### Cryptographic Verification

Every hack is signed with Ed25519:

```json
{
  "hack_id": "hack_buick_1999_bracket_trim",
  "signature": {
    "algorithm": "ed25519",
    "public_key": "base64_encoded_public_key",
    "signature": "base64_encoded_signature"
  }
}
```

**Import process:**
1. Verify signature matches public key
2. Check peer reputation score
3. Reject if verification fails
4. Import if valid

---

## Reputation System

### How Reputation Works

```sql
-- Peer garage reputation
CREATE TABLE peer_garages (
  garage_id TEXT PRIMARY KEY,
  reputation_score INTEGER,
  total_contributions INTEGER,
  verified_contributions INTEGER,
  rejected_contributions INTEGER
);
```

**Reputation changes:**
- ✅ Valid hack imported: `+1`
- ❌ Invalid signature: `-1`
- 🚫 Reputation < 0: Blocked

### View Peer Reputation

```bash
sqlite3 ~/.picoclaw/workspace/sessions.db "SELECT * FROM peer_garages ORDER BY reputation_score DESC"
```

---

## Example Workflow

### Garage A Discovers a Hack

```
User: "I found a shortcut for the alternator replacement on 1999 Buick LeSabre"

AI: "I'll document this discovery. Please confirm the details."

User: "Trim 2mm from the bracket edge to avoid removing the dashboard"

AI: "Recorded as validated hack. Validation count: 1"
```

**Database:**
```sql
INSERT INTO validated_hacks (hack_id, vin, description, validation_count, shared_to_network)
VALUES ('hack_buick_1999_bracket', '1G4HP54K9XH123456', 'Trim 2mm from bracket', 1, 1);
```

---

### Garage B Confirms the Hack

```
User: "I just tried the bracket trim hack on my 1999 Buick"

AI: "Found existing hack: 'Trim 2mm from bracket'. Did it work?"

User: "Yes, saved me 30 minutes!"

AI: "Validation count increased to 2"
```

---

### Garage C Confirms (Reaches Threshold)

```
AI: "This hack now has 3 validations. It will be shared via P2P sync."
```

**Automatic export (next heartbeat):**
```json
{
  "hack_id": "hack_buick_1999_bracket",
  "vehicle": {
    "make": "Buick",
    "model": "LeSabre",
    "year": 1999,
    "vin_pattern": "1G4HP54K*"
  },
  "repair": {
    "solution": "Trim 2mm from bracket edge to avoid dashboard removal",
    "time_saved": "30 minutes"
  },
  "validation": {
    "count": 3,
    "confidence": "medium"
  }
}
```

---

### Garage D Imports Automatically

**Heartbeat runs:**
1. Checks `p2p_import.json`
2. Finds new hack
3. Verifies signature ✅
4. Imports to database
5. Archives import file

**Now available to AI:**
```
User: "How do I replace the alternator on a 1999 Buick LeSabre?"

AI: "I found a validated hack from the P2P network: Trim 2mm from the bracket edge to avoid dashboard removal. This has been validated by 3 garages and saves approximately 30 minutes."
```

---

## Monitoring

### View Sync Log

```bash
sqlite3 ~/.picoclaw/workspace/sessions.db "SELECT * FROM sync_log ORDER BY started_at DESC LIMIT 10"
```

**Output:**
```
sync_id                              | sync_type    | started_at          | hacks_exported | hacks_imported
-------------------------------------|--------------|---------------------|----------------|---------------
a1b2c3d4-...                         | auto_export  | 2024-03-10 14:30:00 | 5              | 0
e5f6g7h8-...                         | auto_import  | 2024-03-10 14:30:05 | 0              | 3
```

---

### View Synced Hacks

```bash
sqlite3 ~/.picoclaw/workspace/sessions.db "SELECT hack_id, source_garage_id, synced_at FROM synced_hacks"
```

---

## Troubleshooting

### Issue: No hacks exported

**Check validation count:**
```sql
SELECT hack_id, validation_count, shared_to_network 
FROM validated_hacks 
WHERE validation_count >= 3;
```

**Solution:** Ensure `shared_to_network = 1` and `validation_count >= min_validations`

---

### Issue: Import rejected

**Check sync log:**
```sql
SELECT * FROM sync_log WHERE errors IS NOT NULL;
```

**Common causes:**
- Invalid signature
- Low peer reputation
- Duplicate hack

---

### Issue: Heartbeat not running

**Check config:**
```json
{
  "heartbeat": {
    "enabled": true,
    "interval": 3600
  }
}
```

**Check logs:**
```bash
grep "p2p_sync" ~/.picoclaw/logs/picoclaw.log
```

---

## Best Practices

### 1. Set Appropriate Validation Threshold

```json
{
  "p2p_sync": {
    "min_validations": 3  // Recommended: 3-5
  }
}
```

**Too low (1-2):** Risk of sharing unproven techniques  
**Too high (10+):** Delays knowledge sharing

---

### 2. Regular Sync Interval

```json
{
  "p2p_sync": {
    "sync_interval": 3600  // 1 hour recommended
  }
}
```

**Too frequent (<600):** Unnecessary overhead  
**Too infrequent (>86400):** Delayed updates

---

### 3. Monitor Peer Reputation

```bash
# Block low-reputation peers
sqlite3 ~/.picoclaw/workspace/sessions.db "DELETE FROM peer_garages WHERE reputation_score < -5"
```

---

### 4. Backup Before Import

```bash
# Backup database before importing
cp ~/.picoclaw/workspace/sessions.db ~/.picoclaw/workspace/sessions.db.backup
picoclaw p2p import --input hacks.json
```

---

## Opt-Out

### Disable P2P Sync

```json
{
  "p2p_sync": {
    "enabled": false
  }
}
```

### Stop Sharing Specific Hacks

```sql
UPDATE validated_hacks 
SET shared_to_network = 0 
WHERE hack_id = 'hack_xyz';
```

---

## Future Enhancements

### Phase 2: Local Network Discovery

```bash
picoclaw p2p discover --local
picoclaw p2p sync --peer 192.168.1.100
```

### Phase 3: IPFS Integration

```bash
picoclaw p2p publish --ipfs
picoclaw p2p subscribe --topic picoclaw-hacks-buick
```

---

## Summary

**P2P Sync makes knowledge sharing:**
- ✅ **Automatic** - No manual export/import
- ✅ **Secure** - Cryptographic signatures
- ✅ **Private** - VINs anonymized
- ✅ **Opt-in** - Disabled by default
- ✅ **Verified** - Reputation system

**Enable it once, benefit forever!** 🚀
