package protocol

// MigrationRequest represents a request from an agent to migrate
// to another node.
type MigrationRequest struct {
	AgentID              string
	RequiredCapabilities []string
	Budget               int64
}

// MigrationOffer represents a node offering to host an agent.
type MigrationOffer struct {
	NodeID         string
	PricePerSecond int64
	AvailableUntil int64
}

// MigrationAccept represents a node's acceptance of a migration request.
type MigrationAccept struct {
	AgentID       string
	TargetNodeID  string
	AcceptedPrice int64
	Success       bool
	ErrorMessage  string
}

// AgentPackage contains all data needed to transfer an agent.
type AgentPackage struct {
	AgentID        string
	WASMBinary     []byte
	Checkpoint     []byte
	ManifestData   []byte
	Budget         int64
	PricePerSecond int64
	ReplayData     *ReplayData `json:",omitempty"` // nil when no tick has been executed (backward compatible)
}

// ReplayData contains replay verification data for a single tick.
// Included in migration packages so the target node can verify checkpoint
// integrity by re-executing the last tick and comparing results (CM-4).
type ReplayData struct {
	PreTickState []byte
	TickNumber   uint64
	Entries      []ReplayEntry
}

// ReplayEntry is a single observation recorded during a tick.
// Protocol-level mirror of eventlog.Entry to keep pkg/protocol dependency-free.
type ReplayEntry struct {
	HostcallID uint16
	Payload    []byte
}

// AgentTransfer represents the payload of an agent being transferred
// between nodes over libp2p stream.
type AgentTransfer struct {
	Package      AgentPackage
	SourceNodeID string
}

// AgentStarted is emitted when an agent successfully starts on a node.
type AgentStarted struct {
	AgentID   string
	NodeID    string
	StartTime int64
	Success   bool
	Error     string
}

// AgentTerminated is emitted when an agent stops execution on a node.
type AgentTerminated struct {
	AgentID string
	NodeID  string
	EndTime int64
	Reason  string
}
