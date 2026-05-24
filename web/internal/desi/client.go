// Package desi is an HTTP client for the DESi governance microservice.
//
// The Go pipeline never re-implements (fakes) the DESi governance
// primitives. Instead it calls this client, which talks to the Python
// service that wraps the genuine desi-governance public API:
// forbidden_hits, replay_hash, canonical_json, core_identity and
// AUDIT_FRAMING.
package desi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to the DESi governance microservice.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New returns a client for the service at baseURL.
func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

// Health is the /health response. CoreIdentity is kept as json.Number so
// the exact value reported by DESi (e.g. 1.0) survives re-serialization
// byte-for-byte.
type Health struct {
	Library       string      `json:"library"`
	Version       string      `json:"version"`
	DesiAvailable bool        `json:"desi_available"`
	CoreIdentity  json.Number `json:"core_identity"`
	AuditFraming  string      `json:"audit_framing"`
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) postJSON(ctx context.Context, path string, payload any) ([]byte, error) {
	buf, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("desi service unreachable: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("desi service returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// Health fetches the protected-core identity and audit framing.
func (c *Client) Health(ctx context.Context) (*Health, error) {
	body, err := c.get(ctx, "/health")
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var h Health
	if err := dec.Decode(&h); err != nil {
		return nil, err
	}
	return &h, nil
}

// ForbiddenHits runs the real desi forbidden/hype-term scan over text.
func (c *Client) ForbiddenHits(ctx context.Context, text string) ([]string, error) {
	body, err := c.postJSON(ctx, "/forbidden-hits", map[string]any{"text": text})
	if err != nil {
		return nil, err
	}
	var out struct {
		Hits []string `json:"hits"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.Hits == nil {
		out.Hits = []string{}
	}
	return out.Hits, nil
}

// ReplayHash returns the real desi replay_hash of obj.
func (c *Client) ReplayHash(ctx context.Context, obj any) (string, error) {
	body, err := c.postJSON(ctx, "/replay-hash", map[string]any{"obj": obj})
	if err != nil {
		return "", err
	}
	var out struct {
		ReplayHash string `json:"replay_hash"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	return out.ReplayHash, nil
}

// CanonicalJSON returns the real desi byte-stable canonical JSON of obj.
func (c *Client) CanonicalJSON(ctx context.Context, obj any) (string, error) {
	body, err := c.postJSON(ctx, "/canonical-json", map[string]any{"obj": obj})
	if err != nil {
		return "", err
	}
	var out struct {
		CanonicalJSON string `json:"canonical_json"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	return out.CanonicalJSON, nil
}
