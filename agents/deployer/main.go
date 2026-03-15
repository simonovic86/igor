// SPDX-License-Identifier: Apache-2.0

//go:build tinygo || wasip1

// deployer: a demo agent that pays for and deploys itself to a compute provider.
//
// The agent demonstrates Igor's self-provisioning capabilities:
//   - Checks its own budget to see if it can afford compute
//   - Encounters HTTP 402 from the provider (payment required)
//   - Pays from its budget via wallet_pay hostcall
//   - Deploys itself with the payment receipt
//   - Monitors deployment status until running
//   - Effect lifecycle ensures crash-safe multi-step workflow
//
// The compute provider is a mock — no real cloud involved.
// The point: an agent can pay for and provision its own infrastructure.
package main

import (
	"encoding/binary"

	"github.com/simonovic86/igor/sdk/igor"
)

const (
	providerURL = "http://localhost:8500"
	deployURL   = providerURL + "/v1/deploy"
	deployCost  = int64(500_000) // matches mockcloud
)

// Intent type prefixes to distinguish payment vs deploy intents in EffectLog.
const (
	intentPayment    byte = 0x01
	intentDeployment byte = 0x02
)

// Phases of the agent.
const (
	phaseCheckBudget uint8 = 0 // assess if budget is sufficient
	phasePayProvider uint8 = 1 // pay compute provider
	phaseDeploy      uint8 = 2 // deploy with receipt
	phaseMonitor     uint8 = 3 // poll deployment status
	phaseReconciling uint8 = 4 // handling unresolved intents on resume
)

// Deployer implements a demo agent that self-provisions compute.
type Deployer struct {
	TickCount    uint64
	BirthNano    int64
	LastNano     int64
	Phase        uint8
	DeploymentID [64]byte // fixed-size deployment ID
	DeployIDLen  uint32
	DeployStatus uint8 // 0=none, 1=pending, 2=provisioning, 3=running, 4=terminated
	PaidCount    uint32
	TotalPaid    int64
	LastReceipt  []byte

	// Effect tracking for crash-safe multi-step workflow.
	Effects igor.EffectLog
}

func (d *Deployer) Init() {}

func (d *Deployer) Tick() bool {
	d.TickCount++
	now := igor.ClockNow()
	if d.BirthNano == 0 {
		d.BirthNano = now
		igor.Logf("[deployer] Agent initialized. Self-provisioning compute.")
	}
	d.LastNano = now
	ageSec := (d.LastNano - d.BirthNano) / 1_000_000_000

	// On resume, handle unresolved intents first.
	unresolved := d.Effects.Unresolved()
	if len(unresolved) > 0 {
		d.Phase = phaseReconciling
		for _, intent := range unresolved {
			d.reconcile(intent)
		}
		d.Effects.Prune()
		// After reconciliation, decide where to go.
		if d.DeployIDLen > 0 {
			d.Phase = phaseMonitor
		} else {
			d.Phase = phaseCheckBudget
		}
		return true
	}

	// Check for pending (Recorded) intents that need execution.
	pending := d.Effects.Pending()
	for _, intent := range pending {
		if intent.State == igor.Recorded {
			if len(intent.Data) > 0 && intent.Data[0] == intentPayment {
				return d.executePayment(intent)
			}
			if len(intent.Data) > 0 && intent.Data[0] == intentDeployment {
				return d.executeDeploy(intent)
			}
		}
	}

	// Phase-based normal operation.
	switch d.Phase {
	case phaseCheckBudget:
		return d.tickCheckBudget(ageSec)
	case phasePayProvider:
		return d.tickPayProvider()
	case phaseDeploy:
		return d.tickDeploy()
	case phaseMonitor:
		return d.tickMonitor(ageSec)
	}

	return false
}

// --- Phase: Check Budget ---

func (d *Deployer) tickCheckBudget(ageSec int64) bool {
	// Already have a deployment? Go to monitor.
	if d.DeployIDLen > 0 {
		igor.Logf("[deployer] tick=%d age=%ds Have deployment, resuming monitoring.",
			d.TickCount, ageSec)
		d.Phase = phaseMonitor
		return true
	}

	balance := igor.WalletBalance()
	igor.Logf("[deployer] tick=%d age=%ds budget=%d deploy_cost=%d",
		d.TickCount, ageSec, balance, deployCost)

	if balance >= deployCost {
		igor.Logf("[deployer] Budget sufficient. Proceeding to pay compute provider.")
		d.Phase = phasePayProvider
		return true
	}

	igor.Logf("[deployer] Insufficient budget. Waiting...")
	return false
}

// --- Phase: Pay Provider ---

func (d *Deployer) tickPayProvider() bool {
	// First, probe the provider to get payment terms.
	status, body, err := igor.HTTPPost(deployURL, "", nil)
	if err != nil {
		igor.Logf("[deployer] HTTP error probing provider: %s", err.Error())
		return false
	}

	if status != 402 {
		igor.Logf("[deployer] Expected 402, got %d", status)
		return false
	}

	amount, recipient, memo := parsePaymentTerms(body)
	if amount <= 0 {
		igor.Logf("[deployer] Invalid payment terms in 402 response")
		return false
	}

	igor.Logf("[deployer] 402 PAYMENT REQUIRED: %d microcents to %s (%s)",
		amount, recipient, memo)

	return d.recordPaymentIntent(amount, recipient, memo)
}

func (d *Deployer) recordPaymentIntent(amount int64, recipient, memo string) bool {
	var key [16]byte
	_ = igor.RandBytes(key[:])

	data := igor.NewEncoder(128).
		Raw([]byte{intentPayment}). // type prefix
		Int64(amount).
		Bytes([]byte(recipient)).
		Bytes([]byte(memo)).
		Finish()

	if err := d.Effects.Record(key[:], data); err != nil {
		igor.Logf("[deployer] ERROR: failed to record payment intent: %s", err.Error())
		return false
	}

	igor.Logf("[deployer] Payment intent RECORDED (key=%x...). Waiting for checkpoint...",
		key[:4])
	return false // Wait for checkpoint before execution.
}

func (d *Deployer) executePayment(intent igor.Intent) bool {
	// Decode: [type:1][amount:8][recipient][memo]
	dd := igor.NewDecoder(intent.Data[1:]) // skip type prefix
	amount := dd.Int64()
	recipient := string(dd.Bytes())
	memo := string(dd.Bytes())
	if err := dd.Err(); err != nil {
		igor.Logf("[deployer] ERROR: corrupt payment intent: %s", err.Error())
		_ = d.Effects.Compensate(intent.ID)
		return false
	}

	if err := d.Effects.Begin(intent.ID); err != nil {
		igor.Logf("[deployer] ERROR: failed to begin payment: %s", err.Error())
		return false
	}

	igor.Logf("[deployer] Payment IN-FLIGHT: %d microcents to %s (key=%x...)",
		amount, recipient, intent.ID[:4])

	// === DANGER ZONE: crash between Begin and Confirm → Unresolved ===

	receipt, err := igor.WalletPay(amount, recipient, memo)
	if err != nil {
		igor.Logf("[deployer] Payment FAILED: %s", err.Error())
		_ = d.Effects.Compensate(intent.ID)
		d.Effects.Prune()
		return false
	}

	// Payment succeeded.
	d.PaidCount++
	d.TotalPaid += amount
	d.LastReceipt = receipt

	if err := d.Effects.Confirm(intent.ID); err != nil {
		igor.Logf("[deployer] ERROR: failed to confirm payment: %s", err.Error())
		return false
	}

	igor.Logf("[deployer] Payment CONFIRMED: %d microcents (receipt=%d bytes). Proceeding to deploy.",
		amount, len(receipt))
	d.Effects.Prune()
	d.Phase = phaseDeploy
	return true
}

// --- Phase: Deploy ---

func (d *Deployer) tickDeploy() bool {
	if len(d.LastReceipt) == 0 {
		igor.Logf("[deployer] No receipt available. Restarting payment.")
		d.Phase = phasePayProvider
		return true
	}
	return d.recordDeployIntent()
}

func (d *Deployer) recordDeployIntent() bool {
	var key [16]byte
	_ = igor.RandBytes(key[:])

	data := igor.NewEncoder(128).
		Raw([]byte{intentDeployment}). // type prefix
		Bytes(d.LastReceipt).
		Finish()

	if err := d.Effects.Record(key[:], data); err != nil {
		igor.Logf("[deployer] ERROR: failed to record deploy intent: %s", err.Error())
		return false
	}

	igor.Logf("[deployer] Deploy intent RECORDED (key=%x...). Waiting for checkpoint...",
		key[:4])
	return false
}

func (d *Deployer) executeDeploy(intent igor.Intent) bool {
	// Decode: [type:1][receipt]
	dd := igor.NewDecoder(intent.Data[1:])
	receipt := dd.Bytes()
	if err := dd.Err(); err != nil {
		igor.Logf("[deployer] ERROR: corrupt deploy intent: %s", err.Error())
		_ = d.Effects.Compensate(intent.ID)
		return false
	}

	if err := d.Effects.Begin(intent.ID); err != nil {
		igor.Logf("[deployer] ERROR: failed to begin deploy: %s", err.Error())
		return false
	}

	igor.Logf("[deployer] Deployment IN-FLIGHT (key=%x...)", intent.ID[:4])

	// === DANGER ZONE ===

	headers := map[string]string{
		"X-Payment":    encodeBytesHex(receipt),
		"Content-Type": "application/json",
	}
	body := []byte(`{"wasm_hash":"0000000000000000000000000000000000000000000000000000000000000000","budget_offer":500000}`)

	status, resp, err := igor.HTTPRequest("POST", deployURL, headers, body)
	if err != nil {
		igor.Logf("[deployer] Deploy HTTP error: %s", err.Error())
		_ = d.Effects.Compensate(intent.ID)
		d.Effects.Prune()
		return false
	}

	if status != 201 {
		igor.Logf("[deployer] Deploy failed: status=%d body=%s", status, string(resp))
		_ = d.Effects.Compensate(intent.ID)
		d.Effects.Prune()
		return false
	}

	// Parse deployment ID from JSON response.
	depID := extractJSONString(resp, "deployment_id")
	if depID == "" {
		igor.Logf("[deployer] Deploy response missing deployment_id")
		_ = d.Effects.Compensate(intent.ID)
		d.Effects.Prune()
		return false
	}

	d.setDeploymentID(depID)
	d.DeployStatus = 1  // pending
	d.LastReceipt = nil // consumed

	if err := d.Effects.Confirm(intent.ID); err != nil {
		igor.Logf("[deployer] ERROR: failed to confirm deploy: %s", err.Error())
		return false
	}

	igor.Logf("[deployer] DEPLOYED: id=%s status=pending. Now monitoring.", depID)
	d.Effects.Prune()
	d.Phase = phaseMonitor
	return true
}

// --- Phase: Monitor ---

func (d *Deployer) tickMonitor(ageSec int64) bool {
	if d.DeployIDLen == 0 {
		igor.Logf("[deployer] No deployment to monitor. Restarting.")
		d.Phase = phaseCheckBudget
		return true
	}

	depID := d.getDeploymentID()

	// Poll every 3 ticks.
	if d.TickCount%3 != 0 {
		if d.TickCount%5 == 0 {
			igor.Logf("[deployer] tick=%d age=%ds monitoring deployment=%s status=%s payments=%d total_paid=%d",
				d.TickCount, ageSec, depID, statusName(d.DeployStatus), d.PaidCount, d.TotalPaid)
		}
		return false
	}

	statusURL := deployURL + "/" + depID
	status, resp, err := igor.HTTPGet(statusURL)
	if err != nil {
		igor.Logf("[deployer] Status check error: %s", err.Error())
		return false
	}

	if status != 200 {
		igor.Logf("[deployer] Status check failed: %d", status)
		return false
	}

	depStatus := extractJSONString(resp, "status")
	switch depStatus {
	case "pending":
		d.DeployStatus = 1
	case "provisioning":
		d.DeployStatus = 2
	case "running":
		if d.DeployStatus != 3 {
			igor.Logf("[deployer] DEPLOYMENT RUNNING! id=%s — self-provisioning complete.", depID)
		}
		d.DeployStatus = 3
	case "terminated":
		d.DeployStatus = 4
	}

	igor.Logf("[deployer] tick=%d deployment=%s status=%s",
		d.TickCount, depID, depStatus)

	return false
}

// --- Reconciliation ---

func (d *Deployer) reconcile(intent igor.Intent) {
	if len(intent.Data) == 0 {
		_ = d.Effects.Compensate(intent.ID)
		return
	}

	switch intent.Data[0] {
	case intentPayment:
		d.reconcilePayment(intent)
	case intentDeployment:
		d.reconcileDeploy(intent)
	default:
		igor.Logf("[deployer] RECONCILING: unknown intent type %d, compensating", intent.Data[0])
		_ = d.Effects.Compensate(intent.ID)
	}
}

func (d *Deployer) reconcilePayment(intent igor.Intent) {
	igor.Logf("[deployer] RECONCILING: unresolved payment (key=%x...)", intent.ID[:4])

	dd := igor.NewDecoder(intent.Data[1:])
	amount := dd.Int64()
	recipient := string(dd.Bytes())
	_ = dd.Bytes() // memo

	// In production: check on-chain settlement.
	// Simulate: if first byte of key is even, payment completed.
	paymentCompleted := (intent.ID[0] % 2) == 0

	if paymentCompleted {
		d.PaidCount++
		d.TotalPaid += amount
		if err := d.Effects.Confirm(intent.ID); err != nil {
			igor.Logf("[deployer] ERROR: reconcile confirm failed: %s", err.Error())
			return
		}
		igor.Logf("[deployer] Reconciled: payment of %d to %s COMPLETED before crash",
			amount, recipient)
	} else {
		if err := d.Effects.Compensate(intent.ID); err != nil {
			igor.Logf("[deployer] ERROR: reconcile compensate failed: %s", err.Error())
			return
		}
		igor.Logf("[deployer] Reconciled: payment of %d to %s DID NOT complete — will retry",
			amount, recipient)
	}
}

func (d *Deployer) reconcileDeploy(intent igor.Intent) {
	igor.Logf("[deployer] RECONCILING: unresolved deployment (key=%x...)", intent.ID[:4])

	// Check if we already have a deployment ID (set before crash).
	if d.DeployIDLen > 0 {
		depID := d.getDeploymentID()
		// Query provider for status.
		statusURL := deployURL + "/" + depID
		status, _, err := igor.HTTPGet(statusURL)
		if err == nil && status == 200 {
			if err := d.Effects.Confirm(intent.ID); err != nil {
				igor.Logf("[deployer] ERROR: reconcile confirm failed: %s", err.Error())
				return
			}
			igor.Logf("[deployer] Reconciled: deployment %s EXISTS — confirming", depID)
			return
		}
	}

	// Deployment didn't complete — compensate.
	if err := d.Effects.Compensate(intent.ID); err != nil {
		igor.Logf("[deployer] ERROR: reconcile compensate failed: %s", err.Error())
		return
	}
	igor.Logf("[deployer] Reconciled: deployment DID NOT complete — will retry")
}

// --- Checkpoint Serialization ---

func (d *Deployer) Marshal() []byte {
	return igor.NewEncoder(512).
		Uint64(d.TickCount).
		Int64(d.BirthNano).
		Int64(d.LastNano).
		Raw([]byte{d.Phase}).
		Raw(d.DeploymentID[:]).
		Uint32(d.DeployIDLen).
		Raw([]byte{d.DeployStatus}).
		Uint32(d.PaidCount).
		Int64(d.TotalPaid).
		Bytes(d.LastReceipt).
		Bytes(d.Effects.Marshal()).
		Finish()
}

func (d *Deployer) Unmarshal(data []byte) {
	dd := igor.NewDecoder(data)
	d.TickCount = dd.Uint64()
	d.BirthNano = dd.Int64()
	d.LastNano = dd.Int64()

	phaseBuf := dd.FixedBytes(1)
	if len(phaseBuf) > 0 {
		d.Phase = phaseBuf[0]
	}

	dd.ReadInto(d.DeploymentID[:])
	d.DeployIDLen = dd.Uint32()

	statusBuf := dd.FixedBytes(1)
	if len(statusBuf) > 0 {
		d.DeployStatus = statusBuf[0]
	}

	d.PaidCount = dd.Uint32()
	d.TotalPaid = dd.Int64()
	d.LastReceipt = dd.Bytes()
	d.Effects.Unmarshal(dd.Bytes()) // THE RESUME RULE: InFlight → Unresolved
	if err := dd.Err(); err != nil {
		panic("unmarshal checkpoint: " + err.Error())
	}
}

// --- Helpers ---

func (d *Deployer) setDeploymentID(id string) {
	n := len(id)
	if n > 64 {
		n = 64
	}
	copy(d.DeploymentID[:], id[:n])
	d.DeployIDLen = uint32(n)
}

func (d *Deployer) getDeploymentID() string {
	return string(d.DeploymentID[:d.DeployIDLen])
}

func statusName(s uint8) string {
	switch s {
	case 0:
		return "none"
	case 1:
		return "pending"
	case 2:
		return "provisioning"
	case 3:
		return "running"
	case 4:
		return "terminated"
	default:
		return "unknown"
	}
}

// parsePaymentTerms extracts amount, recipient, and memo from a 402 response body.
// Format: [amount:8 LE][recipient_len:4 LE][recipient][memo_len:4 LE][memo]
func parsePaymentTerms(body []byte) (amount int64, recipient, memo string) {
	if len(body) < 12 {
		return 0, "", ""
	}
	amount = int64(binary.LittleEndian.Uint64(body[:8]))
	off := 8
	recipientLen := int(binary.LittleEndian.Uint32(body[off:]))
	off += 4
	if off+recipientLen > len(body) {
		return 0, "", ""
	}
	recipient = string(body[off : off+recipientLen])
	off += recipientLen
	if off+4 > len(body) {
		return amount, recipient, ""
	}
	memoLen := int(binary.LittleEndian.Uint32(body[off:]))
	off += 4
	if off+memoLen > len(body) {
		return amount, recipient, ""
	}
	memo = string(body[off : off+memoLen])
	return amount, recipient, memo
}

// extractJSONString does minimal JSON extraction without encoding/json (keeps WASM small).
// Looks for "key":"value" and returns value.
func extractJSONString(data []byte, key string) string {
	s := string(data)
	needle := `"` + key + `":"`
	idx := indexOf(s, needle)
	if idx < 0 {
		return ""
	}
	start := idx + len(needle)
	end := indexOfFrom(s, `"`, start)
	if end < 0 {
		return ""
	}
	return s[start:end]
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func indexOfFrom(s, sub string, from int) int {
	for i := from; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// encodeBytesHex encodes bytes as hex string (no dependencies).
func encodeBytesHex(data []byte) string {
	const hex = "0123456789abcdef"
	buf := make([]byte, len(data)*2)
	for i, b := range data {
		buf[i*2] = hex[b>>4]
		buf[i*2+1] = hex[b&0x0f]
	}
	return string(buf)
}

func init() { igor.Run(&Deployer{}) }
func main() {}
