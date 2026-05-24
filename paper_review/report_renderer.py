"""Deterministic Markdown rendering of a review artifact.

Report sections (fixed order): Scope, Main Claims, Evidence Gaps,
Overclaim Risks, Reproducibility Risks, Questions for Human Reviewer,
Limitations of This Review. The same artifact always renders to the same
bytes.
"""
from __future__ import annotations

_MAIN_CLAIM_CATEGORIES = (
    "main_claim",
    "novelty_claim",
    "generalization_claim",
    "result_claim",
)

_LIMITATIONS_TEXT = (
    "- This tool is a reviewer ASSISTANT, not a peer reviewer. Its only "
    "verdict is `REVIEW_ASSISTANCE_ONLY`.",
    "- It never accepts or rejects a paper, never replaces a human "
    "reviewer, never determines truth, and never guarantees correctness.",
    "- All findings are produced by deterministic keyword and structural "
    "heuristics on the submitted text. They may contain false positives "
    "and false negatives.",
    "- Absence of a flag is not evidence of quality; presence of a flag "
    "is not evidence of a defect. Every item requires human judgement.",
    "- The output is offline and reproducible; it reflects only the text "
    "provided and incorporates no external knowledge.",
)


def _fmt_terms(terms: list[str]) -> str:
    return ", ".join(f"`{t}`" for t in terms) if terms else "none"


def render_report(artifact: dict) -> str:
    gov = artifact["governance"]
    mode = gov["mode"]
    lines: list[str] = []

    lines.append("# DESi Paper Review - Assistance Report")
    lines.append("")
    lines.append(f"**Paper:** {artifact['paper_title']}")
    lines.append(f"**Verdict:** `{artifact['verdict']}`")
    lines.append(
        f"**Tool:** {artifact['tool']} {artifact['tool_version']}"
    )
    lines.append(f"**Replay hash:** `{artifact['replay_hash']}`")
    lines.append("")

    # --- Scope -------------------------------------------------------
    lines.append("## Scope")
    lines.append("")
    lines.append(gov["disclaimer"])
    lines.append("")
    lines.append(f"> {gov['audit_framing']}")
    lines.append("")
    lines.append(f"- Governance library: `{gov['library']}`")
    lines.append(f"- Protected-core identity: `{gov['core_identity']}`")
    lines.append(
        "- Hype / forbidden-term hits (DESi scan): "
        f"{_fmt_terms(gov['forbidden_term_hits'])}"
    )
    lines.append(
        "- Mode: offline_mode="
        f"{str(mode['offline_mode']).lower()}, allow_live_llm_calls="
        f"{str(mode['allow_live_llm_calls']).lower()}, live_calls_enabled="
        f"{str(mode['live_calls_enabled']).lower()}"
    )
    lines.append("")

    # --- Main Claims -------------------------------------------------
    lines.append("## Main Claims")
    lines.append("")
    main_claims = [
        c for c in artifact["claims"]
        if c["category"] in _MAIN_CLAIM_CATEGORIES
    ]
    if main_claims:
        for c in main_claims:
            lines.append(
                f"- **[{c['claim_id']} / {c['category']}]** "
                f"({c['section']}) {c['text']}"
            )
    else:
        lines.append("_No main claims were detected._")
    lines.append("")

    # --- Evidence Gaps -----------------------------------------------
    lines.append("## Evidence Gaps")
    lines.append("")
    if artifact["evidence_gaps"]:
        for gap in artifact["evidence_gaps"]:
            lines.append(
                f"- **[{gap['claim_id']}]** ({gap['section']}) {gap['note']}"
            )
    else:
        lines.append("_No evidence gaps were flagged._")
    lines.append("")

    # --- Overclaim Risks ---------------------------------------------
    lines.append("## Overclaim Risks")
    lines.append("")
    if artifact["overclaims"]:
        for oc in artifact["overclaims"]:
            lines.append(
                f"- **[{oc['claim_id']}]** terms {_fmt_terms(oc['terms'])} "
                f"in ({oc['section']}): {oc['text']}"
            )
    else:
        lines.append("_No overclaim language was detected._")
    lines.append("")

    # --- Reproducibility Risks ---------------------------------------
    lines.append("## Reproducibility Risks")
    lines.append("")
    if artifact["reproducibility_risks"]:
        for risk in artifact["reproducibility_risks"]:
            lines.append(f"- **{risk['risk_type']}**: {risk['detail']}")
    else:
        lines.append("_No reproducibility risks were flagged._")
    lines.append("")

    # --- Questions for Human Reviewer --------------------------------
    lines.append("## Questions for Human Reviewer")
    lines.append("")
    for idx, question in enumerate(artifact["reviewer_questions"], start=1):
        lines.append(f"{idx}. {question}")
    lines.append("")

    # --- Limitations of This Review ----------------------------------
    lines.append("## Limitations of This Review")
    lines.append("")
    lines.extend(_LIMITATIONS_TEXT)
    lines.append("")

    return "\n".join(lines)


__all__ = ["render_report"]
