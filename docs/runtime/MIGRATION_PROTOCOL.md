# Migration Protocol

## Overview

The Igor migration protocol enables agents to move between nodes while preserving state and budget. Migration occurs over a dedicated libp2p stream protocol.

## Protocol Specification

**Protocol ID:** `/igor/migrate/1.0.0`

**Transport:** libp2p stream (bidirectional)

**Encoding:** JSON

## Message Types

### AgentPackage

Complete agent data for transfer.

```go
type AgentPackage struct {
    AgentID        string      // Unique agent identifier
    WASMBinary     []byte      // Compiled WASM module
    WASMHash       []byte      // SHA-256 of WASMBinary for integrity verification
    Checkpoint     []byte      // Serialized state + budget metadata (209-byte header, v0x04)
    ManifestData   []byte      // Capability manifest JSON
    Budget         int64       // Remaining budget in microcents
    PricePerSecond int64       // Cost per second in microcents
    ReplayData     *ReplayData // Replay verification data (nil if no tick executed)
}
```

### AgentTransfer

Stream message sent from source to target.

```go
type AgentTransfer struct {
    Package      AgentPackage
    SourceNodeID string       // Origin peer ID
}
```

```go

// ReplayData contains replay verification data for a single tick.
// Included in migration packages so the target node can verify checkpoint
// integrity by re-executing the last tick and comparing results (CM-4).
type ReplayData struct {
    PreTickState []byte        // Agent state before the tick
    TickNumber   uint64        // Tick that was executed
    Entries      []ReplayEntry // Ordered observation hostcall records
}

// ReplayEntry is a single observation recorded during a tick.
type ReplayEntry struct {
    HostcallID uint16 // Identifies which hostcall (clock_now=1, rand_bytes=2, log_emit=3)
    Payload    []byte // Serialized return value
}
```

### AgentStarted

Confirmation sent from target back to source.

```go
type AgentStarted struct {
    AgentID string
    NodeID  string  // Target peer ID
    Success bool
    Error   string  // If Success=false
}
```

## Migration Flow

### Complete Sequence

```
Source Node                              Target Node
     │                                        │
     │ 1. Load checkpoint                     │
     │    from storage                        │
     │                                        │
     │ 2. Load WASM binary                    │
     │    from filesystem                     │
     │                                        │
     │ 3. Extract budget from                 │
     │    checkpoint metadata                 │
     │                                        │
     │ 4. Connect to target peer              │
     ├───────────────────────────────────────>│
     │                                        │
     │ 5. Open /igor/migrate/1.0.0 stream     │
     ├═══════════════════════════════════════>│
     │                                        │
     │ 6. Send AgentTransfer                  │
     ├───────────────────────────────────────>│
     │                                        │
     │                                        │ 7. Decode transfer
     │                                        │ 8. Verify WASM hash
     │                                        │ 9. Verify replay (if data present)
     │                                        │ 10. Save checkpoint
     │                                        │ 11. Load agent from bytes
     │                                        │ 12. Initialize
     │                                        │ 13. Resume from checkpoint
     │                                        │ 14. Register as active
     │                                        │
     │ 15. Receive AgentStarted               │
     │<───────────────────────────────────────┤
     │                                        │
     │ 16. Verify success                     │
     │                                        │
     │ 17. Terminate local instance           │
     │                                        │
     │ 18. Delete local checkpoint            │
     │                                        │
     X (Agent terminated)               ● (Agent running)
```

## Implementation Details

### Source Node (Outgoing)

**Function:** `MigrateAgent(ctx, agentID, wasmPath, targetPeerAddr)`

**Steps:**

1. **Parse target address**
   ```go
   maddr := multiaddr.NewMultiaddr(targetPeerAddr)
   addrInfo := peer.AddrInfoFromP2pAddr(maddr)
   ```

2. **Connect to peer**
   ```go
   host.Connect(ctx, *addrInfo)
   ```

3. **Load checkpoint**
   ```go
   checkpoint := storage.LoadCheckpoint(ctx, agentID)
   ```

4. **Load WASM binary**
   ```go
   wasmBinary := os.ReadFile(wasmPath)
   ```

5. **Extract budget metadata**
   ```go
   budgetVal, pricePerSecond, _, _, _, _ := agent.ParseCheckpointHeader(checkpoint)
   ```

6. **Create package**
   ```go
   wasmHash := sha256.Sum256(wasmBinary)
   pkg := AgentPackage{
       AgentID:        agentID,
       WASMBinary:     wasmBinary,
       WASMHash:       wasmHash[:],
       Checkpoint:     checkpoint,
       ManifestData:   manifestData,
       Budget:         budgetVal,
       PricePerSecond: pricePerSecond,
       ReplayData:     replayDataFromInstance(instance, checkpoint),
   }
   ```

7. **Open stream**
   ```go
   stream := host.NewStream(ctx, peerID, MigrateProtocol)
   ```

8. **Send transfer**
   ```go
   json.NewEncoder(stream).Encode(AgentTransfer{
       Package:      pkg,
       SourceNodeID: sourceID,
   })
   ```

9. **Wait for confirmation**
   ```go
   var started AgentStarted
   json.NewDecoder(stream).Decode(&started)
   ```

10. **Handle success**
    ```go
    if started.Success {
        // Terminate local instance
        activeAgents[agentID].Close(ctx)
        delete(activeAgents, agentID)
        
        // Delete checkpoint
        storage.DeleteCheckpoint(ctx, agentID)
    }
    ```

### Target Node (Incoming)

**Handler:** `handleIncomingMigration(stream)`

**Steps:**

1. **Decode transfer**
   ```go
   var transfer AgentTransfer
   json.NewDecoder(stream).Decode(&transfer)
   ```

2. **Extract package**
   ```go
   pkg := transfer.Package
   ```

3. **Verify WASM hash**
   ```go
   computed := sha256.Sum256(pkg.WASMBinary)
   if computed != [32]byte(pkg.WASMHash) {
       // Reject migration
   }
   ```

4. **Verify replay data** (if present)
   ```go
   if pkg.ReplayData != nil {
       result := replayEngine.ReplayTick(ctx, pkg.WASMBinary, ...)
       if !result.Verified {
           // Reject migration
       }
   }
   ```

5. **Save checkpoint**
   ```go
   storage.SaveCheckpoint(ctx, pkg.AgentID, pkg.Checkpoint)
   ```

6. **Load agent from bytes**
   ```go
   instance := agent.LoadAgentFromBytes(
       ctx, engine, pkg.WASMBinary, pkg.AgentID,
       storage, pkg.Budget, pkg.PricePerSecond, pkg.ManifestData, logger,
   )
   ```

7. **Initialize and resume**
   ```go
   instance.Init(ctx)
   instance.LoadCheckpointFromStorage(ctx)
   ```

8. **Register as active**
   ```go
   activeAgents[pkg.AgentID] = instance
   ```

9. **Send confirmation**
   ```go
   json.NewEncoder(stream).Encode(AgentStarted{
       AgentID: pkg.AgentID,
       NodeID:  localNodeID,
       Success: true,
   })
   ```

## CLI Usage

### Migrate Agent Command

```bash
./bin/igord \
  --migrate-agent <agentID> \
  --to <target_multiaddr> \
  --wasm <wasm_binary_path>
```

**Example:**
```bash
./bin/igord \
  --migrate-agent local-agent \
  --to /ip4/127.0.0.1/tcp/4002/p2p/12D3KooW... \
  --wasm agents/research/example/agent.wasm
```

### Requirements

- Checkpoint must exist for the agent
- Target node must be reachable
- WASM binary must be accessible
- Sufficient budget must remain

## Error Handling

### Connection Failures

If source cannot connect to target:
- Log error
- Keep local instance running
- Checkpoint remains intact
- Agent continues on source

### Transfer Failures

If transfer encoding fails:
- Close stream
- Log error
- Keep local instance running
- Agent continues on source

### Target Failures

If target cannot start agent:
- Target sends `AgentStarted{Success: false, Error: "reason"}`
- Source receives failure
- Source keeps local instance
- Agent continues on source

### Confirmation Timeout

If no confirmation received:
- Source stream timeout
- Log error
- Keep local instance
- Agent continues on source

**Safety:** Migration is conservative. On any error, agent stays on source node.

## Protocol Guarantees

### Single Instance Invariant

**Guarantee:** At most one active instance exists at any time.

**Implementation:**
- Source terminates only after confirmation
- Target starts before sending confirmation
- Atomic checkpoint operations
- No optimistic migration

### State Consistency

**Guarantee:** Agent state is never lost or corrupted.

**Implementation:**
- Atomic checkpoint writes (temp → fsync → rename)
- Checkpoint validated before deletion
- Resume errors abort startup
- Original checkpoint remains until confirmed

### Budget Conservation

**Guarantee:** Budget is never created or duplicated.

**Implementation:**
- Budget loaded from checkpoint
- Transferred in AgentPackage
- Source checkpoint deleted after confirmation
- No double-spending possible

## Performance Characteristics

### Migration Latency

Typical migration time (local network):
- Connect: ~10ms
- Transfer: ~50ms (for 190KB WASM + 65B checkpoint)
- Resume: ~300ms (WASM compilation)
- Total: ~360ms

### Checkpoint Size

- Header: 209 bytes (version + budget + price + tick + wasmHash + majorVersion + leaseGeneration + leaseExpiry + prevHash + agentPubKey + signature)
- State: Agent-dependent
- Example agent: 237 bytes total (209 header + 28 state)

### WASM Transfer Size

- Counter agent: ~190KB
- Actual size depends on agent complexity
- Transferred once per migration

## Migration Logging

### Source Node

```
Starting agent migration agent_id=local-agent target=/ip4/.../p2p/...
Checkpoint loaded for migration checkpoint_size=65
Budget metadata extracted budget=0.999999 price_per_second=0.001000
Transfer sent agent_id=local-agent
Agent started on target target_node=12D3KooW...
Local agent instance terminated agent_id=local-agent
Local checkpoint deleted agent_id=local-agent
Migration completed successfully agent_id=local-agent
```

### Target Node

```
Receiving agent migration from_peer=12D3KooW...
Agent package received agent_id=local-agent wasm_size=188404 checkpoint_size=65
Checkpoint saved agent_id=local-agent path=checkpoints/local-agent.checkpoint
Agent loaded successfully agent_id=local-agent
Budget restored from checkpoint budget=0.999999
[agent] Resumed with counter=7
Agent migration accepted and started agent_id=local-agent from_node=12D3KooW...
```

## Protocol Extensions (Future)

Not yet implemented:

- Multi-hop migration routing
- Migration negotiation (price/resources)
- Capability matching
- Cryptographic agent identity
- Receipt signing for payment proof

These may be added in future phases beyond v0.
