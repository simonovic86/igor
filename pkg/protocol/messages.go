package protocol

// MigrationRequest represents a request from an agent to migrate
// to another node.
type MigrationRequest struct {
	AgentID              string
	RequiredCapabilities []string
	Budget               float64
}

// MigrationOffer represents a node offering to host an agent.
type MigrationOffer struct {
	NodeID        string
	PricePerSecond float64
	AvailableUntil int64
}

// MigrationAccept represents a node's acceptance of a migration request.
type MigrationAccept struct {
	AgentID       string
	TargetNodeID  string
	AcceptedPrice float64
	Success       bool
	ErrorMessage  string
}

// AgentPackage contains all data needed to transfer an agent.
type AgentPackage struct {
	AgentID        string
	WASMBinary     []byte
	Checkpoint     []byte
	ManifestData   []byte
	Budget         float64
	PricePerSecond float64
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
	AgentID   string
	NodeID    string
	EndTime   int64
	Reason    string
}
