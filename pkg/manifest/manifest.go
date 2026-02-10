package manifest

// Manifest describes an agent's identity, requirements, and policies.
type Manifest struct {
	// AgentID is the unique identifier for this agent.
	AgentID string

	// OwnerPublicKey is the cryptographic public key of the agent's owner.
	OwnerPublicKey []byte

	// CapabilitiesRequired lists the runtime capabilities this agent needs.
	CapabilitiesRequired []string

	// MigrationPolicy defines rules for when and how the agent can migrate.
	MigrationPolicy MigrationPolicy

	// ResourceLimits defines the maximum resources the agent may consume.
	ResourceLimits ResourceLimits
}

// MigrationPolicy specifies migration behavior constraints.
type MigrationPolicy struct {
	// Enabled indicates whether migration is allowed.
	Enabled bool

	// MaxPricePerSecond is the maximum price the agent is willing to pay.
	MaxPricePerSecond float64

	// PreferredRegions is a list of geographic preferences (optional).
	PreferredRegions []string
}

// ResourceLimits defines runtime resource constraints.
type ResourceLimits struct {
	// MaxMemoryBytes is the maximum memory the agent may use.
	MaxMemoryBytes uint64

	// MaxCPUMillis is the maximum CPU time per tick in milliseconds.
	MaxCPUMillis uint64

	// MaxStorageBytes is the maximum persistent storage allowed.
	MaxStorageBytes uint64
}
