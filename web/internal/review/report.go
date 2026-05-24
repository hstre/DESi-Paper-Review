package review

import (
	"fmt"
	"strings"
)

var mainClaimCategories = map[string]bool{
	CatMain: true, CatNovelty: true, CatGeneralization: true, CatResult: true,
}

var limitationsText = []string{
	"- This tool is a reviewer ASSISTANT, not a peer reviewer. Its only verdict is `REVIEW_ASSISTANCE_ONLY`.",
	"- It never accepts or rejects a paper, never replaces a human reviewer, never determines truth, and never guarantees correctness.",
	"- All findings are produced by deterministic keyword and structural heuristics on the submitted text. They may contain false positives and false negatives.",
	"- Absence of a flag is not evidence of quality; presence of a flag is not evidence of a defect. Every item requires human judgement.",
	"- The output is offline and reproducible; it reflects only the text provided and incorporates no external knowledge.",
}

func fmtTerms(terms []string) string {
	if len(terms) == 0 {
		return "none"
	}
	parts := make([]string, len(terms))
	for i, t := range terms {
		parts[i] = "`" + t + "`"
	}
	return strings.Join(parts, ", ")
}

// RenderReport renders the artifact to a deterministic Markdown report.
func RenderReport(a Artifact) string {
	gov := a.Governance
	mode := gov.Mode
	var b strings.Builder
	w := func(s string) { b.WriteString(s); b.WriteByte('\n') }

	w("# DESi Paper Review - Assistance Report")
	w("")
	w(fmt.Sprintf("**Paper:** %s", a.PaperTitle))
	w(fmt.Sprintf("**Verdict:** `%s`", a.Verdict))
	w(fmt.Sprintf("**Tool:** %s %s", a.Tool, a.ToolVersion))
	w(fmt.Sprintf("**Replay hash:** `%s`", a.ReplayHash))
	w("")

	w("## Scope")
	w("")
	w(gov.Disclaimer)
	w("")
	w("> " + gov.AuditFraming)
	w("")
	w(fmt.Sprintf("- Governance library: `%s`", gov.Library))
	w(fmt.Sprintf("- Protected-core identity: `%s`", gov.CoreIdentity.String()))
	w(fmt.Sprintf("- Hype / forbidden-term hits (DESi scan): %s", fmtTerms(gov.ForbiddenTermHits)))
	w(fmt.Sprintf("- Mode: offline_mode=%t, allow_live_llm_calls=%t, live_calls_enabled=%t",
		mode.OfflineMode, mode.AllowLiveLLMCalls, mode.LiveCallsEnabled))
	w("")

	w("## Main Claims")
	w("")
	mainCount := 0
	for _, c := range a.Claims {
		if mainClaimCategories[c.Category] {
			w(fmt.Sprintf("- **[%s / %s]** (%s) %s", c.ClaimID, c.Category, c.Section, c.Text))
			mainCount++
		}
	}
	if mainCount == 0 {
		w("_No main claims were detected._")
	}
	w("")

	w("## Evidence Gaps")
	w("")
	if len(a.EvidenceGaps) > 0 {
		for _, g := range a.EvidenceGaps {
			w(fmt.Sprintf("- **[%s]** (%s) %s", g.ClaimID, g.Section, g.Note))
		}
	} else {
		w("_No evidence gaps were flagged._")
	}
	w("")

	w("## Overclaim Risks")
	w("")
	if len(a.Overclaims) > 0 {
		for _, oc := range a.Overclaims {
			w(fmt.Sprintf("- **[%s]** terms %s in (%s): %s",
				oc.ClaimID, fmtTerms(oc.Terms), oc.Section, oc.Text))
		}
	} else {
		w("_No overclaim language was detected._")
	}
	w("")

	w("## Reproducibility Risks")
	w("")
	if len(a.ReproducibilityRisks) > 0 {
		for _, r := range a.ReproducibilityRisks {
			w(fmt.Sprintf("- **%s**: %s", r.RiskType, r.Detail))
		}
	} else {
		w("_No reproducibility risks were flagged._")
	}
	w("")

	w("## Questions for Human Reviewer")
	w("")
	for i, q := range a.ReviewerQuestions {
		w(fmt.Sprintf("%d. %s", i+1, q))
	}
	w("")

	w("## Limitations of This Review")
	w("")
	for _, l := range limitationsText {
		w(l)
	}
	w("")

	return b.String()
}
