# Example Igor Agent

A minimal autonomous agent that demonstrates the Igor lifecycle.

## Functionality

This agent maintains a simple counter that:
- Initializes to 0
- Increments on each tick
- Can be checkpointed to save state
- Can be resumed from saved state

## Building

Requires TinyGo:

```bash
make build
```

This produces `agent.wasm`.

## Agent Lifecycle

The agent implements the required Igor lifecycle functions:

### `init()`
Called when the agent first starts. Initializes counter to 0.

### `tick()`
Called periodically by the runtime. Increments and prints the counter.

### `checkpoint() -> (ptr, len)`
Serializes the current counter value to 8 bytes (uint64 little-endian).
Returns pointer and length to the serialized state.

### `resume(ptr, len)`
Restores the counter from previously checkpointed state.

### `malloc(size) -> ptr`
Allocates memory for state restoration.

## Running

```bash
igord --run-agent agents/example/agent.wasm
```

The agent will:
1. Initialize
2. Tick every second
3. Checkpoint periodically
4. Survive restarts by resuming from checkpoint
