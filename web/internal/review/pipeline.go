package review

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hstre/desi-paper-review-web/internal/desi"
)

// ErrGovernance is returned when the protected-core identity gate fails.
var ErrGovernance = errors.New("DESi protected-core identity gate failed")

var riskQuestion = map[string]string{
	"missing_code":        "Can the authors provide a link to the source code required to reproduce the reported results?",
	"missing_data":        "Are the underlying data available for independent inspection?",
	"missing_baselines":   "Which baselines were used, and how does the method compare against prior work?",
	"missing_parameters":  "What exact hyperparameters, seeds, and training/configuration settings were used?",
	"unclear_dataset":     "Which dataset (name, size, and splits) was used, and how was it constructed?",
	"unsupported_metrics": "How were the reported metrics computed (evaluation protocol, test set, and variance or significance)?",
}

// Pipeline runs the offline review and delegates governance primitives to
// the DESi microservice.
type Pipeline struct {
	Desi   *desi.Client
	Config Config
}

// Result bundles the structured artifact and its DESi-canonical JSON.
type Result struct {
	Artifact Artifact
	JSON     string
}

// Review runs the full pipeline over the given paper text.
func (pl *Pipeline) Review(ctx context.Context, text string) (*Result, error) {
	paper := ParseText(text)
	claims := ExtractClaims(paper)

	overclaims := buildOverclaims(claims)
	overclaimIDs := map[string]bool{}
	for _, oc := range overclaims {
		overclaimIDs[oc.ClaimID] = true
	}
	unsupported, gaps := buildUnsupported(claims, overclaimIDs)
	risks := DetectReproducibilityRisks(paper.RawText)
	questions := buildReviewerQuestions(overclaims, gaps, risks)

	// --- Real DESi governance via the microservice ---
	health, err := pl.Desi.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("governance health check failed: %w", err)
	}
	if id, _ := health.CoreIdentity.Float64(); id != 1.0 {
		return nil, fmt.Errorf("%w (core_identity=%s); refusing to emit a review artifact",
			ErrGovernance, health.CoreIdentity.String())
	}
	forbidden, err := pl.Desi.ForbiddenHits(ctx, paper.RawText)
	if err != nil {
		return nil, fmt.Errorf("forbidden-term scan failed: %w", err)
	}

	artifact := Artifact{
		SchemaVersion:        SchemaVersion,
		Tool:                 Tool,
		ToolVersion:          ToolVersion,
		PaperTitle:           paper.Title,
		Claims:               claims,
		UnsupportedClaims:    unsupported,
		Overclaims:           overclaims,
		EvidenceGaps:         gaps,
		ReproducibilityRisks: risks,
		ReviewerQuestions:    questions,
		Verdict:              Verdict,
		Governance: Governance{
			Library:           health.Library,
			CoreIdentity:      health.CoreIdentity,
			AuditFraming:      health.AuditFraming,
			ForbiddenTermHits: forbidden,
			Mode: Mode{
				OfflineMode:       pl.Config.OfflineMode,
				AllowLiveLLMCalls: pl.Config.AllowLiveLLMCalls,
				LiveCallsEnabled:  pl.Config.LiveCallsEnabled(),
			},
			Disclaimer: disclaimer,
		},
	}

	// Hash the body (replay_hash omitted), then attach the real hash.
	hash, err := pl.Desi.ReplayHash(ctx, artifact)
	if err != nil {
		return nil, fmt.Errorf("replay hashing failed: %w", err)
	}
	artifact.ReplayHash = hash

	// Final byte-stable serialization is produced by the real DESi
	// canonical_json primitive.
	canonical, err := pl.Desi.CanonicalJSON(ctx, artifact)
	if err != nil {
		return nil, fmt.Errorf("canonical serialization failed: %w", err)
	}

	return &Result{Artifact: artifact, JSON: canonical}, nil
}

func buildOverclaims(claims []Claim) []Overclaim {
	out := []Overclaim{}
	for _, c := range claims {
		terms := OverclaimTermsIn(c.Text)
		if len(terms) > 0 {
			out = append(out, Overclaim{
				ClaimID:  c.ClaimID,
				Category: c.Category,
				Section:  c.Section,
				Text:     c.Text,
				Terms:    terms,
			})
		}
	}
	return out
}

func buildUnsupported(claims []Claim, overclaimIDs map[string]bool) ([]UnsupportedClaim, []EvidenceGap) {
	unsupported := []UnsupportedClaim{}
	gaps := []EvidenceGap{}
	for _, c := range claims {
		strong := strongCategories[c.Category] || overclaimIDs[c.ClaimID]
		if strong && !HasInlineSupport(c.Text) {
			unsupported = append(unsupported, UnsupportedClaim{
				ClaimID:  c.ClaimID,
				Category: c.Category,
				Section:  c.Section,
				Text:     c.Text,
			})
			gaps = append(gaps, EvidenceGap{
				ClaimID: c.ClaimID,
				Section: c.Section,
				Missing: "inline_support",
				Note:    "Strong claim presented without inline data, a citation, or a table/figure reference.",
			})
		}
	}
	return unsupported, gaps
}

func buildReviewerQuestions(overclaims []Overclaim, gaps []EvidenceGap, risks []ReproRisk) []string {
	questions := []string{}
	for _, oc := range overclaims {
		questions = append(questions, fmt.Sprintf(
			"Claim %s uses strong language (%s). Does the evidence justify it, or should the authors soften or substantiate the wording?",
			oc.ClaimID, strings.Join(oc.Terms, ", ")))
	}
	for _, g := range gaps {
		questions = append(questions, fmt.Sprintf(
			"What specific evidence supports claim %s? It currently lacks inline data, a citation, or a table/figure reference.",
			g.ClaimID))
	}
	seen := map[string]bool{}
	for _, r := range risks {
		if q, ok := riskQuestion[r.RiskType]; ok && !seen[r.RiskType] {
			questions = append(questions, q)
			seen[r.RiskType] = true
		}
	}
	if len(questions) == 0 {
		questions = append(questions, "No automated concerns were flagged; please assess the paper on its scientific merits.")
	}
	return questions
}
