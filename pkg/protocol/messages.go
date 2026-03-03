package protocol

// AgentPackage contains all data needed to transfer an agent.
type AgentPackage struct {
	AgentID        string
	WASMBinary     []byte
	WASMHash       []byte // SHA-256 of WASMBinary for integrity verification
	Checkpoint     []byte
	ManifestData   []byte
	Budget         int64
	PricePerSecond int64
	ReplayData     *ReplayData `json:",omitempty"`
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

// AgentStarted is the confirmation message sent by the target node after
// receiving a migration. Sent for both success and failure.
type AgentStarted struct {
	AgentID   string
	NodeID    string
	StartTime int64
	Success   bool
	Error     string
}
