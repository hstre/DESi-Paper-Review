package review

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/hstre/desi-paper-review-web/internal/desi"
)

// mockDesi returns a test double of the DESi governance microservice. It
// canonicalizes via Go's key-sorted marshaling so replay hashes are
// deterministic, mirroring the contract the real service satisfies.
func mockDesi(t *testing.T, coreIdentity float64, forbidden []string) *httptest.Server {
	t.Helper()
	canon := func(raw json.RawMessage) []byte {
		var v any
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		if err := dec.Decode(&v); err != nil {
			t.Fatalf("mock canon decode: %v", err)
		}
		b, err := json.Marshal(v) // Go sorts map keys deterministically
		if err != nil {
			t.Fatalf("mock canon marshal: %v", err)
		}
		return append(b, '\n')
	}
	readObj := func(r *http.Request) json.RawMessage {
		var req struct {
			Obj json.RawMessage `json:"obj"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("mock read obj: %v", err)
		}
		return req.Obj
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		ci := strconv.FormatFloat(coreIdentity, 'f', 1, 64)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"library":"desi-governance","version":"mock",` +
			`"desi_available":true,"core_identity":` + ci +
			`,"audit_framing":"DESi does not validate itself."}`))
	})
	mux.HandleFunc("/forbidden-hits", func(w http.ResponseWriter, _ *http.Request) {
		out, _ := json.Marshal(map[string]any{"hits": forbidden})
		w.Write(out)
	})
	mux.HandleFunc("/replay-hash", func(w http.ResponseWriter, r *http.Request) {
		sum := sha256.Sum256(canon(readObj(r)))
		json.NewEncoder(w).Encode(map[string]string{"replay_hash": hex.EncodeToString(sum[:])})
	})
	mux.HandleFunc("/canonical-json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"canonical_json": string(canon(readObj(r)))})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestPipelineAgainstMockService(t *testing.T) {
	srv := mockDesi(t, 1.0, []string{"Breakthrough"})
	pl := &Pipeline{Desi: desi.New(srv.URL), Config: DefaultConfig()}

	res, err := pl.Review(context.Background(), sample)
	if err != nil {
		t.Fatal(err)
	}
	if res.Artifact.Verdict != Verdict {
		t.Fatalf("verdict = %q", res.Artifact.Verdict)
	}
	if len(res.Artifact.Overclaims) == 0 {
		t.Error("expected overclaims")
	}
	if len(res.Artifact.ReproducibilityRisks) == 0 {
		t.Error("expected reproducibility risks")
	}
	if res.Artifact.ReplayHash == "" {
		t.Error("expected a replay hash from the service")
	}
	if res.JSON == "" {
		t.Error("expected canonical JSON from the service")
	}
	if got := res.Artifact.Governance.CoreIdentity.String(); got != "1.0" {
		t.Errorf("core_identity = %s, want 1.0", got)
	}
	hits := res.Artifact.Governance.ForbiddenTermHits
	if len(hits) != 1 || hits[0] != "Breakthrough" {
		t.Errorf("forbidden hits = %v, want [Breakthrough]", hits)
	}
}

func TestGateRefusesWhenCoreIdentityNotOne(t *testing.T) {
	srv := mockDesi(t, 0.0, []string{})
	pl := &Pipeline{Desi: desi.New(srv.URL), Config: DefaultConfig()}

	_, err := pl.Review(context.Background(), sample)
	if err == nil {
		t.Fatal("expected the protected-core gate to refuse output")
	}
	if !errors.Is(err, ErrGovernance) {
		t.Fatalf("expected ErrGovernance, got %v", err)
	}
}

func TestDeterministicAgainstMock(t *testing.T) {
	srv := mockDesi(t, 1.0, []string{})
	pl := &Pipeline{Desi: desi.New(srv.URL), Config: DefaultConfig()}

	r1, err := pl.Review(context.Background(), sample)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := pl.Review(context.Background(), sample)
	if err != nil {
		t.Fatal(err)
	}
	if r1.JSON != r2.JSON {
		t.Error("canonical JSON is not stable across runs")
	}
	if r1.Artifact.ReplayHash != r2.Artifact.ReplayHash {
		t.Error("replay hash is not stable across runs")
	}
}

func TestServiceUnreachableSurfacesError(t *testing.T) {
	// Point at a closed port; Review must return an error, not panic.
	pl := &Pipeline{Desi: desi.New("http://127.0.0.1:59999"), Config: DefaultConfig()}
	if _, err := pl.Review(context.Background(), sample); err == nil {
		t.Fatal("expected an error when the service is unreachable")
	}
}
