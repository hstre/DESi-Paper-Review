// Command desi-paper-review-web serves the browser UI for DESi Paper
// Review and runs the offline Go pipeline. Governance primitives come
// from the real desi-governance library via the DESi microservice.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/hstre/desi-paper-review-web/internal/desi"
	"github.com/hstre/desi-paper-review-web/internal/review"
	"github.com/hstre/desi-paper-review-web/internal/server"
)

func main() {
	addr := flag.String("addr", ":8080", "address for the web UI to listen on")
	desiURL := flag.String("desi", "http://127.0.0.1:8765", "base URL of the DESi governance microservice")
	check := flag.Bool("check", false, "run a one-shot self-check and exit (no server)")
	allowLive := flag.Bool("allow-live", false, "allow live LLM calls (requires --offline=false; MVP makes none)")
	offline := flag.Bool("offline", true, "offline mode (no network beyond the local DESi service)")
	flag.Parse()

	cfg := review.Config{OfflineMode: *offline, AllowLiveLLMCalls: *allowLive}

	if *check {
		os.Exit(runCheck(*desiURL, cfg))
	}

	srv := server.New(*desiURL, cfg)
	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	fmt.Printf("DESi Paper Review web UI on http://localhost%s  (DESi service: %s)\n", *addr, *desiURL)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "server error:", err)
		os.Exit(1)
	}
}

// runCheck verifies the DESi service link, runs the pipeline on the
// built-in sample twice, and confirms byte-stable output.
func runCheck(desiURL string, cfg review.Config) int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := desi.New(desiURL)
	pl := &review.Pipeline{Desi: client, Config: cfg}

	h, err := client.Health(ctx)
	if err != nil {
		fmt.Println("[FAIL] DESi service unreachable:", err)
		fmt.Println("\nDESI_PAPER_REVIEW_WEB_NOT_READY")
		return 1
	}
	id, _ := h.CoreIdentity.Float64()
	if id != 1.0 {
		fmt.Printf("[FAIL] core_identity=%s (expected 1.0)\n", h.CoreIdentity.String())
		fmt.Println("\nDESI_PAPER_REVIEW_WEB_NOT_READY")
		return 1
	}
	fmt.Printf("[ok] DESi governance online (%s %s, core_identity=%s)\n", h.Library, h.Version, h.CoreIdentity.String())

	r1, err := pl.Review(ctx, server.SamplePaper)
	if err != nil {
		fmt.Println("[FAIL] pipeline error:", err)
		fmt.Println("\nDESI_PAPER_REVIEW_WEB_NOT_READY")
		return 1
	}
	if r1.Artifact.Verdict != review.Verdict {
		fmt.Printf("[FAIL] unexpected verdict %q\n", r1.Artifact.Verdict)
		fmt.Println("\nDESI_PAPER_REVIEW_WEB_NOT_READY")
		return 1
	}
	r2, err := pl.Review(ctx, server.SamplePaper)
	if err != nil {
		fmt.Println("[FAIL] pipeline error on second run:", err)
		fmt.Println("\nDESI_PAPER_REVIEW_WEB_NOT_READY")
		return 1
	}
	if r1.JSON != r2.JSON || review.RenderReport(r1.Artifact) != review.RenderReport(r2.Artifact) {
		fmt.Println("[FAIL] output is not replay-stable across runs")
		fmt.Println("\nDESI_PAPER_REVIEW_WEB_NOT_READY")
		return 1
	}
	fmt.Printf("[ok] pipeline replay-stable (hash %s...)\n", short(r1.Artifact.ReplayHash))
	fmt.Printf("[ok] sample: %d claims, %d overclaims, %d evidence gaps, %d reproducibility risks\n",
		len(r1.Artifact.Claims), len(r1.Artifact.Overclaims),
		len(r1.Artifact.EvidenceGaps), len(r1.Artifact.ReproducibilityRisks))

	fmt.Println("\nDESI_PAPER_REVIEW_WEB_READY")
	return 0
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}
