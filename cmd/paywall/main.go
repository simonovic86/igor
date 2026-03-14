// SPDX-License-Identifier: Apache-2.0

// Mock x402 paywall server for the x402buyer demo.
//
// Endpoints:
//
//	GET /api/premium-data
//	  - No X-Payment header → 402 with payment terms
//	  - X-Payment header present → 200 with premium data
//
// Payment terms are binary-encoded in the response body:
//
//	[amount:8 LE][recipient_len:4 LE][recipient][memo_len:4 LE][memo]
package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	paymentAmount    int64  = 100_000 // 0.10 currency units (100,000 microcents)
	paymentRecipient string = "paywall-provider"
	paymentMemo      string = "premium-data-access"
)

func main() {
	addr := ":8402"
	if envAddr := os.Getenv("PAYWALL_ADDR"); envAddr != "" {
		addr = envAddr
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/premium-data", handlePremiumData)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("[paywall] Listening on %s", addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("[paywall] Server error: %v", err)
		}
	}()

	<-done
	log.Println("[paywall] Shutting down...")
	srv.Close()
}

func handlePremiumData(w http.ResponseWriter, r *http.Request) {
	payment := r.Header.Get("X-Payment")

	if payment == "" {
		// No payment — return 402 with payment terms.
		log.Printf("[paywall] 402 → %s (no payment header)", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write(encodePaymentTerms(paymentAmount, paymentRecipient, paymentMemo))
		return
	}

	// Payment present — return premium data.
	log.Printf("[paywall] 200 → %s (payment received, %d bytes)", r.RemoteAddr, len(payment))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)

	// Simulate premium data: a randomized market insight.
	insights := []string{
		"BTC dominance rising to 58.3%, altcoin rotation expected",
		"ETH gas fees averaging 12 gwei, L2 adoption accelerating",
		"DeFi TVL crossed $95B, new ATH since 2022",
		"Stablecoin supply expanding, USDC market cap up 15% MoM",
		"On-chain whale activity: 3 wallets accumulated 12,000 BTC this week",
	}
	insight := insights[rand.IntN(len(insights))]
	fmt.Fprintf(w, `{"insight":"%s","timestamp":%d,"tier":"premium"}`, insight, time.Now().Unix())
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
