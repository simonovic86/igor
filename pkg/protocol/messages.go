package protocol

// MigrationRequest represents a request from an agent to migrate
// to another node.
type MigrationRequest struct {
	AgentID             string
	RequiredCapabilities []string
	Budget              float64
}

// MigrationAccept represents a node's acceptance of a migration request.
type MigrationAccept struct {
	TargetNodeID    string
	AcceptedPrice   float64
	TargetAddress   string
}

// AgentTransfer represents the payload of an agent being transferred
// between nodes.
type AgentTransfer struct {
	AgentID         string
	WASMBinary      []byte
	State           []byte
	ManifestData    []byte
}

// AgentStarted is emitted when an agent successfully starts on a node.
type AgentStarted struct {
	AgentID       string
	NodeID        string
	StartTime     int64
}

// AgentTerminated is emitted when an agent stops execution on a node.
type AgentTerminated struct {
	AgentID       string
	NodeID        string
	EndTime       int64
	Reason        string
}
