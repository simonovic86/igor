// SPDX-License-Identifier: Apache-2.0

// Mock compute provider server for the deployer demo.
//
// Simulates a decentralized compute marketplace (like Akash or Golem).
// Agents pay for compute and deploy themselves.
//
// Endpoints:
//
//	GET  /health                — health check
//	POST /v1/deploy             — request deployment (requires X-Payment header)
//	GET  /v1/deploy/{id}        — check deployment status
//	POST /v1/deploy/{id}/terminate — terminate deployment
//
// Without X-Payment header, POST /v1/deploy returns 402 with payment terms.
// Payment terms are binary-encoded:
//
//	[amount:8 LE][recipient_len:4 LE][recipient][memo_len:4 LE][memo]
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	deploymentCost   int64  = 500_000 // 0.50 currency units
	paymentRecipient string = "mockcloud-provider"
	paymentMemo      string = "compute-deployment"
)

type deployment struct {
	ID         string `json:"deployment_id"`
	WASMHash   string `json:"wasm_hash"`
	Status     string `json:"status"`
	queryCount int
}

type server struct {
	mu          sync.Mutex
	deployments map[string]*deployment
	counter     int
}

func newServer() *server {
	return &server{
		deployments: make(map[string]*deployment),
	}
}

func main() {
	addr := ":8500"
	if envAddr := os.Getenv("MOCKCLOUD_ADDR"); envAddr != "" {
		addr = envAddr
	}

	s := newServer()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})
	mux.HandleFunc("/v1/deploy", s.handleDeploy)
	mux.HandleFunc("/v1/deploy/", s.handleDeployByID)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("[mockcloud] Listening on %s", addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("[mockcloud] Server error: %v", err)
		}
	}()

	<-done
	log.Println("[mockcloud] Shutting down...")
	srv.Close()
}

func (s *server) handleDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payment := r.Header.Get("X-Payment")
	if payment == "" {
		// No payment — return 402 with payment terms.
		log.Printf("[mockcloud] 402 → %s (no payment header)", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write(encodePaymentTerms(deploymentCost, paymentRecipient, paymentMemo))
		return
	}

	// Parse request body for wasm_hash.
	var req struct {
		WASMHash    string `json:"wasm_hash"`
		BudgetOffer int64  `json:"budget_offer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.counter++
	id := fmt.Sprintf("dep-%04d", s.counter)
	dep := &deployment{
		ID:       id,
		WASMHash: req.WASMHash,
		Status:   "pending",
	}
	s.deployments[id] = dep
	s.mu.Unlock()

	log.Printf("[mockcloud] 201 → %s deployment=%s wasm=%s budget=%d",
		r.RemoteAddr, id, req.WASMHash, req.BudgetOffer)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(dep)
}

func (s *server) handleDeployByID(w http.ResponseWriter, r *http.Request) {
	// Extract deployment ID from path: /v1/deploy/{id} or /v1/deploy/{id}/terminate
	path := strings.TrimPrefix(r.URL.Path, "/v1/deploy/")
	parts := strings.SplitN(path, "/", 2)
	id := parts[0]

	if id == "" {
		http.Error(w, "missing deployment id", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	dep, ok := s.deployments[id]
	if !ok {
		s.mu.Unlock()
		http.Error(w, "deployment not found", http.StatusNotFound)
		return
	}

	// Check for terminate action.
	if len(parts) > 1 && parts[1] == "terminate" && r.Method == http.MethodPost {
		dep.Status = "terminated"
		s.mu.Unlock()
		log.Printf("[mockcloud] TERMINATED deployment=%s", id)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(dep)
		return
	}

	if r.Method != http.MethodGet {
		s.mu.Unlock()
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Progress status on each query.
	dep.queryCount++
	switch {
	case dep.queryCount <= 1:
		dep.Status = "pending"
	case dep.queryCount <= 2:
		dep.Status = "provisioning"
	default:
		dep.Status = "running"
	}
	status := dep.Status
	s.mu.Unlock()

	log.Printf("[mockcloud] STATUS deployment=%s status=%s query=%d", id, status, dep.queryCount)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dep)
}

// encodePaymentTerms encodes payment parameters for the 402 response body.
// Format: [amount:8 LE][recipient_len:4 LE][recipient][memo_len:4 LE][memo]
func encodePaymentTerms(amount int64, recipient, memo string) []byte {
	size := 8 + 4 + len(recipient) + 4 + len(memo)
	buf := make([]byte, size)
	off := 0
	binary.LittleEndian.PutUint64(buf[off:], uint64(amount))
	off += 8
	binary.LittleEndian.PutUint32(buf[off:], uint32(len(recipient)))
	off += 4
	copy(buf[off:], recipient)
	off += len(recipient)
	binary.LittleEndian.PutUint32(buf[off:], uint32(len(memo)))
	off += 4
	copy(buf[off:], memo)
	return buf
}
