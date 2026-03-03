# Survivor Agent

An autonomous agent that demonstrates Igor's survival capabilities: checkpointing, migration, resume, and continuity across infrastructure. Built with the Igor Agent SDK.

## What It Does

The survivor agent maintains persistent state across ticks, checkpoints, and migrations:

- **TickCount** — total ticks ever executed (survives restarts)
- **BirthNano** — timestamp of the agent's first tick (persists through migrations)
- **LastNano** — timestamp of the most recent tick (for uptime calculation)
- **Luck** — running XOR of random bytes (demonstrates rand hostcall)

Each tick it logs a narrative line showing its tick count, age, and luck value. Every 10 ticks it logs a milestone. After a restart or migration, the tick counter and age continue from where they left off.

## SDK Usage

This agent uses the Igor SDK (`sdk/igor`), which handles all WASM lifecycle exports and memory management. The agent implements the `igor.Agent` interface:

```go
type Survivor struct { ... }
func (s *Survivor) Init()                 // one-time initialization
func (s *Survivor) Tick()                 // per-tick logic using igor.ClockNow(), igor.RandBytes(), igor.Logf()
func (s *Survivor) Marshal() []byte       // serialize state for checkpointing
func (s *Survivor) Unmarshal(data []byte)  // restore state on resume
func init() { igor.Run(&Survivor{}) }
```

The SDK provides the five required WASM exports (`agent_init`, `agent_tick`, `agent_checkpoint`, `agent_checkpoint_ptr`, `agent_resume`) automatically.

## State Format

28 bytes, little-endian binary:

```
[0:8]   TickCount uint64
[8:16]  BirthNano int64
[16:24] LastNano  int64
[24:28] Luck      uint32
```

## Replay Compatibility

The agent is designed for CM-4 (Observation Determinism) compliance:
- Only `Tick()` calls observation hostcalls (clock, rand, log)
- `Init()`, `Marshal()`, and `Unmarshal()` are pure — no hostcalls
- `Unmarshal()` performs pure state restoration with no side effects

This ensures replay verification works correctly both locally and during migration.

## Hostcalls Used

All three observation hostcalls (declared in `agent.manifest.json`):

- `igor.ClockNow()` — read wall clock for age tracking
- `igor.RandBytes()` — generate random bytes for luck accumulation
- `igor.Logf()` — emit narrative log messages

## Building

Requires TinyGo:

```bash
make agent    # from repo root
```

## Running

```bash
make run-agent                    # default budget 1.0
./bin/igord --run-agent agents/example/agent.wasm --budget 10.0
```

## Demonstrating Survival

```bash
# Start the agent
./bin/igord --run-agent agents/example/agent.wasm --budget 10.0

# Let it tick a few times, then Ctrl-C (checkpoints on shutdown)

# Restart — it resumes from checkpoint, tick count and age continue
./bin/igord --run-agent agents/example/agent.wasm --budget 10.0
```
