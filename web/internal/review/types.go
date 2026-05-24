// Package review is a deterministic, offline Go re-implementation of the
// DESi Paper Review pipeline. It assists a human reviewer and never
// accepts, rejects, or validates a paper. Its only verdict is
// REVIEW_ASSISTANCE_ONLY.
//
// The governance primitives (hype scan, replay hash, canonical JSON,
// protected-core identity, audit framing) are NOT re-implemented here.
// They are obtained from the real desi-governance library via the DESi
// microservice (see package desi).
package review

import "encoding/json"

const (
	// Verdict is the only verdict this tool will ever emit.
	Verdict = "REVIEW_ASSISTANCE_ONLY"
	// SchemaVersion of the emitted artifact.
	SchemaVersion = "1.0"
	// Tool identifier.
	Tool = "desi-paper-review-web"
	// ToolVersion of this Go implementation.
	ToolVersion = "0.1.0a0"
)

// Section is one parsed section of a paper.
type Section struct {
	Title string
	Level int
	Text  string
}

// ParsedPaper is the structured form of an input paper.
type ParsedPaper struct {
	Title    string
	Sections []Section
	RawText  string
}

// Claim is one extracted, classified sentence.
type Claim struct {
	ClaimID  string `json:"claim_id"`
	Category string `json:"category"`
	Section  string `json:"section"`
	Text     string `json:"text"`
}

// Overclaim is a claim containing one or more overclaim terms.
type Overclaim struct {
	ClaimID  string   `json:"claim_id"`
	Category string   `json:"category"`
	Section  string   `json:"section"`
	Text     string   `json:"text"`
	Terms    []string `json:"terms"`
}

// UnsupportedClaim is a strong claim lacking inline support.
type UnsupportedClaim struct {
	ClaimID  string `json:"claim_id"`
	Category string `json:"category"`
	Section  string `json:"section"`
	Text     string `json:"text"`
}

// EvidenceGap describes missing support for a claim.
type EvidenceGap struct {
	ClaimID string `json:"claim_id"`
	Section string `json:"section"`
	Missing string `json:"missing"`
	Note    string `json:"note"`
}

// ReproRisk is one reproducibility risk.
type ReproRisk struct {
	RiskType string `json:"risk_type"`
	Detail   string `json:"detail"`
}

// Mode records the offline/live gating flags.
type Mode struct {
	OfflineMode       bool `json:"offline_mode"`
	AllowLiveLLMCalls bool `json:"allow_live_llm_calls"`
	LiveCallsEnabled  bool `json:"live_calls_enabled"`
}

// Governance is the DESi governance stamp. CoreIdentity is a json.Number
// so the exact DESi value (e.g. 1.0) is preserved byte-for-byte.
type Governance struct {
	Library           string      `json:"library"`
	CoreIdentity      json.Number `json:"core_identity"`
	AuditFraming      string      `json:"audit_framing"`
	ForbiddenTermHits []string    `json:"forbidden_term_hits"`
	Mode              Mode        `json:"mode"`
	Disclaimer        string      `json:"disclaimer"`
}

// Artifact is the full review artifact. ReplayHash is omitted while the
// body is hashed, then attached.
type Artifact struct {
	SchemaVersion        string             `json:"schema_version"`
	Tool                 string             `json:"tool"`
	ToolVersion          string             `json:"tool_version"`
	PaperTitle           string             `json:"paper_title"`
	Claims               []Claim            `json:"claims"`
	UnsupportedClaims    []UnsupportedClaim `json:"unsupported_claims"`
	Overclaims           []Overclaim        `json:"overclaims"`
	EvidenceGaps         []EvidenceGap      `json:"evidence_gaps"`
	ReproducibilityRisks []ReproRisk        `json:"reproducibility_risks"`
	ReviewerQuestions    []string           `json:"reviewer_questions"`
	Verdict              string             `json:"verdict"`
	Governance           Governance         `json:"governance"`
	ReplayHash           string             `json:"replay_hash,omitempty"`
}

// Config holds the offline/live gating configuration.
type Config struct {
	OfflineMode       bool
	AllowLiveLLMCalls bool
}

// DefaultConfig is offline with no live calls.
func DefaultConfig() Config {
	return Config{OfflineMode: true, AllowLiveLLMCalls: false}
}

// LiveCallsEnabled requires BOTH gates: not offline AND explicitly allowed.
func (c Config) LiveCallsEnabled() bool {
	return !c.OfflineMode && c.AllowLiveLLMCalls
}

const disclaimer = "This artifact assists human review only. It never accepts, rejects, " +
	"or validates a paper, never replaces a human reviewer, and never " +
	"determines truth or guarantees correctness."

// Categories of claims (parallel to the Python implementation).
const (
	CatMain           = "main_claim"
	CatEvidence       = "evidence_claim"
	CatMethod         = "method_claim"
	CatResult         = "result_claim"
	CatLimitation     = "limitation_claim"
	CatNovelty        = "novelty_claim"
	CatGeneralization = "generalization_claim"
)
