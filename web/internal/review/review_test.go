package review

import (
	"encoding/json"
	"testing"
)

const sample = `# A Novel Method That Solves Generalization and Proves Robust Across Tasks

## Abstract
We present the first method that solves the long-standing problem of
generalization in learning systems. Our novel approach generalizes to
any task and proves robust under arbitrary conditions. We claim
significant gains and argue that the method establishes a new standard.

## Method
Our method applies a sequence of transformations to the input. The
procedure is simple and broadly applicable across any domain.

## Results
Our approach achieves high accuracy and strong performance on a dataset.
The method generalizes and remains robust throughout.

## Conclusion
We have presented the first approach that solves generalization, stays
robust, and proves significant in practice.
`

func TestParseDetectsTitleAndSections(t *testing.T) {
	p := ParseText(sample)
	if p.Title == "" || p.Title[:8] != "A Novel " {
		t.Fatalf("unexpected title: %q", p.Title)
	}
	want := map[string]bool{"Abstract": true, "Method": true, "Results": true, "Conclusion": true}
	got := map[string]bool{}
	for _, s := range p.Sections {
		got[s.Title] = true
	}
	for w := range want {
		if !got[w] {
			t.Errorf("missing section %q", w)
		}
	}
}

func TestAllOverclaimTermsDetected(t *testing.T) {
	found := map[string]bool{}
	for _, term := range OverclaimTermsIn(sample) {
		found[term] = true
	}
	for _, term := range OverclaimTerms {
		if !found[term] {
			t.Errorf("overclaim term not detected: %q", term)
		}
	}
}

func TestReproducibilityRisks(t *testing.T) {
	types := map[string]bool{}
	for _, r := range DetectReproducibilityRisks(sample) {
		types[r.RiskType] = true
	}
	for _, want := range []string{
		"missing_code", "missing_data", "missing_baselines",
		"missing_parameters", "unclear_dataset", "unsupported_metrics",
	} {
		if !types[want] {
			t.Errorf("reproducibility risk not detected: %q", want)
		}
	}
}

func TestExtractClaimsAndOverclaims(t *testing.T) {
	p := ParseText(sample)
	claims := ExtractClaims(p)
	if len(claims) == 0 {
		t.Fatal("expected claims")
	}
	ocs := buildOverclaims(claims)
	if len(ocs) == 0 {
		t.Fatal("expected overclaims")
	}
	for _, c := range claims {
		if c.ClaimID == "" || c.Category == "" {
			t.Errorf("malformed claim: %+v", c)
		}
	}
}

func TestReportIsDeterministic(t *testing.T) {
	a := Artifact{
		PaperTitle: "T",
		Tool:       Tool,
		Verdict:    Verdict,
		Governance: Governance{
			Library:           "desi-governance",
			CoreIdentity:      json.Number("1.0"),
			AuditFraming:      "framing",
			ForbiddenTermHits: []string{},
			Mode:              Mode{OfflineMode: true},
		},
		Claims:               []Claim{{ClaimID: "C001", Category: CatMain, Section: "Abstract", Text: "x"}},
		Overclaims:           []Overclaim{},
		EvidenceGaps:         []EvidenceGap{},
		UnsupportedClaims:    []UnsupportedClaim{},
		ReproducibilityRisks: []ReproRisk{},
		ReviewerQuestions:    []string{"q1"},
	}
	if RenderReport(a) != RenderReport(a) {
		t.Fatal("report rendering is not deterministic")
	}
}

func TestClaimIDPadding(t *testing.T) {
	if claimID(1) != "C001" || claimID(42) != "C042" || claimID(123) != "C123" {
		t.Fatalf("bad claim ids: %s %s %s", claimID(1), claimID(42), claimID(123))
	}
}
