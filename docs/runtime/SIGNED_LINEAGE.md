# Signed Checkpoint Lineage

**Status:** Implemented (Task 13)
**Derives from:** EI-3 (Checkpoint Lineage Integrity), EI-11 (Divergent Lineage Detection), OA-1 (Canonical Logical Identity)

## Purpose

Signed checkpoint lineage provides cryptographic proof that a sequence of checkpoints was produced by a single agent identity. Each checkpoint is signed by the agent's Ed25519 private key and includes the hash of the previous checkpoint, forming a tamper-evident hash chain. This enables:

- Verifiable checkpoint authenticity without trusting the executing node
- Divergent lineage detection through conflicting checkpoint hashes (EI-11)
- Foundation for trustless multi-node operation

## Agent Identity

Each agent has an Ed25519 keypair that serves as its canonical cryptographic identity (OA-1). The identity is:

- **Generated** when an agent is first loaded on a node
- **Persisted** as `<agentID>.identity` alongside checkpoints
- **Transferred** during migration as part of the `AgentPackage`
- **Deleted** from the source node after successful migration

The private key signs checkpoints. The public key is embedded in each checkpoint for independent verification.

**Package:** `pkg/identity/`

### Serialization

```
[privkey_len:4 LE][private_key:64 bytes]
```

The public key is derived from the private key on load.

## Checkpoint Format v0x04

Extends v0x03 (81-byte header) with 128 bytes of lineage metadata:

```
Offset  Field            Type       Size   Notes
──────  ──────────────── ────────── ────── ──────────────────────────
0       Version          uint8      1      0x04
1-8     Budget           int64 LE   8      Microcents
9-16    PricePerSecond   int64 LE   8      Microcents/second
17-24   TickNumber       uint64 LE  8      Sequential counter
25-56   WASMHash         [32]byte   32     SHA-256 of WASM binary
57-64   MajorVersion     uint64 LE  8      Authority epoch major
65-72   LeaseGeneration  uint64 LE  8      Authority epoch lease gen
73-80   LeaseExpiry      uint64 LE  8      Unix nanoseconds
81-112  PrevHash         [32]byte   32     SHA-256 of previous checkpoint
113-144 AgentPubKey      [32]byte   32     Agent's Ed25519 public key
145-208 Signature        [64]byte   64     Ed25519 signature
209+    AgentState       []byte     N      WASM-provided state
```

**Total header size:** 209 bytes

### Backward Compatibility

- v0x02 and v0x03 checkpoints are readable by the updated parser
- New checkpoints are written as v0x04 when agent identity is set
- If agent identity is nil (disabled), checkpoints are written as v0x03

## Signing Protocol

### Signing Domain

The signature covers everything in the checkpoint except the 64-byte signature field itself:

```
[version..agentPubKey] || [state]
= checkpoint[0:145] || checkpoint[209:]
```

This ensures the signature covers the version, budget, tick number, WASM hash, epoch, previous hash, public key, and the full agent state.

### Sign Flow (SaveCheckpointToStorage)

1. Build checkpoint with all header fields, signature slot zeroed
2. Construct signing domain: `checkpoint[0:145] || state`
3. Sign with `ed25519.Sign(agentPrivateKey, signingDomain)`
4. Write signature into `checkpoint[145:209]`
5. Copy state into `checkpoint[209:]`
6. Compute `SHA-256(fullCheckpoint)` → store as `PrevCheckpointHash` for next save

### Verify Flow (LoadCheckpointFromStorage)

1. Parse header, extract `AgentPubKey` and `Signature`
2. Reconstruct signing domain: `checkpoint[0:145] || state`
3. Verify with `ed25519.Verify(AgentPubKey, signingDomain, Signature)`
4. Reject checkpoint if verification fails
5. Compute `SHA-256(fullCheckpoint)` → set as `PrevCheckpointHash`

**Package:** `pkg/lineage/`

## Lineage Chain

Each checkpoint includes `PrevHash` — the SHA-256 of the previous complete checkpoint blob. This creates a hash chain:

```
Genesis(PrevHash=0x00..00) → CP1(PrevHash=SHA256(Genesis)) → CP2(PrevHash=SHA256(CP1)) → ...
```

- **Genesis checkpoint:** `PrevHash` is all zeros (`[0x00; 32]`)
- **Fork detection:** Two checkpoints at the same tick with different `PrevHash` values indicate a lineage fork (EI-11), triggering `RECOVERY_REQUIRED`

## Content Addressing

The SHA-256 hash of a complete checkpoint blob serves as a content address compatible with IPFS/CID (raw codec + sha2-256 multihash). Computed by `lineage.ContentHash()`.

## Migration

During migration:

1. **Source** serializes `AgentIdentity` via `MarshalBinary()` and includes it in `AgentPackage.AgentIdentity`
2. **Target** deserializes the identity and passes it to `LoadAgentFromBytes`
3. **Target** persists the identity locally via `storage.SaveIdentity()`
4. **Source** deletes the local identity file after confirmed migration

The agent's `PrevCheckpointHash` chains correctly across migrations because the target loads the migrated checkpoint and computes its content hash.

## Invariant Enforcement

| Invariant | Enforcement |
|-----------|-------------|
| EI-3 (Checkpoint Lineage Integrity) | PrevHash chain ensures linear ordering; signature prevents forgery |
| EI-11 (Divergent Lineage Detection) | Conflicting PrevHash values between nodes prove a lineage fork |
| OA-1 (Canonical Logical Identity) | Agent public key in each checkpoint binds identity to lineage |
| MC-3 (Migration Lineage Preservation) | Identity and PrevHash travel with the agent during migration |
| RE-1 (Atomic Checkpoints) | Identity file uses same atomic write pattern as checkpoints |

## Security Considerations

- **Key compromise:** If the private key is extracted from storage, an attacker could forge checkpoints. Mitigation: identity files should have restricted filesystem permissions.
- **No replay protection:** Signed lineage proves authorship but does not prevent a compromised node from replaying old checkpoints. Byzantine fault tolerance is out of scope for v0 (see THREAT_MODEL.md A2).
- **Identity loss:** If the identity file is lost but checkpoints remain, the agent cannot produce checkpoints that chain to the existing lineage. This is treated as a lineage break.
