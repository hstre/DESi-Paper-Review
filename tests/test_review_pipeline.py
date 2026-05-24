from __future__ import annotations

import json
from pathlib import Path

from paper_review import VERDICT
from paper_review.claim_extractor import OVERCLAIM_TERMS, overclaim_terms_in
from paper_review.config import Config, load_config
from paper_review.parser import parse_paper
from paper_review.report_renderer import render_report
from paper_review.review_pipeline import (
    artifact_json,
    detect_reproducibility_risks,
    review_file,
)

REPO = Path(__file__).resolve().parents[1]
SAMPLE = REPO / "examples" / "sample_paper.md"
GOLDEN_JSON = REPO / "examples" / "sample_review_output.json"
GOLDEN_REPORT = REPO / "examples" / "sample_review_report.md"


def test_desi_governance_is_importable():
    import desi  # noqa: F401
    from desi.core.governance_core import core_identity
    from desi.core.replay_kernel import canonical_json, replay_hash
    from desi.reviewer.reviewer_port import AUDIT_FRAMING
    from desi.scientific_rendering import forbidden_hits

    assert core_identity() == 1.0
    assert callable(forbidden_hits)
    assert callable(replay_hash)
    assert callable(canonical_json)
    assert isinstance(AUDIT_FRAMING, str) and AUDIT_FRAMING


def test_verdict_is_assistance_only():
    artifact = review_file(SAMPLE, Config())
    assert artifact["verdict"] == VERDICT == "REVIEW_ASSISTANCE_ONLY"


def test_all_overclaim_terms_detected_in_sample():
    text = parse_paper(SAMPLE).full_text()
    found = set(overclaim_terms_in(text))
    assert set(OVERCLAIM_TERMS) <= found, f"missing: {set(OVERCLAIM_TERMS) - found}"


def test_overclaims_present_in_artifact():
    artifact = review_file(SAMPLE, Config())
    assert artifact["overclaims"], "expected overclaims to be flagged"
    flagged = {t for oc in artifact["overclaims"] for t in oc["terms"]}
    # Several distinct overclaim terms should surface across claims.
    assert len(flagged) >= 5


def test_evidence_gaps_present():
    artifact = review_file(SAMPLE, Config())
    assert artifact["evidence_gaps"], "expected evidence gaps"
    assert artifact["unsupported_claims"]


def test_reproducibility_risks_detected():
    text = parse_paper(SAMPLE).full_text()
    types = {r["risk_type"] for r in detect_reproducibility_risks(text)}
    expected = {
        "missing_code",
        "missing_data",
        "missing_baselines",
        "missing_parameters",
        "unclear_dataset",
        "unsupported_metrics",
    }
    assert expected <= types, f"missing: {expected - types}"


def test_json_output_is_byte_stable():
    a1 = artifact_json(review_file(SAMPLE, Config()))
    a2 = artifact_json(review_file(SAMPLE, Config()))
    assert a1 == a2


def test_markdown_output_is_byte_stable():
    r1 = render_report(review_file(SAMPLE, Config()))
    r2 = render_report(review_file(SAMPLE, Config()))
    assert r1 == r2


def test_replay_hash_is_stable_and_recorded():
    artifact = review_file(SAMPLE, Config())
    assert "replay_hash" in artifact
    assert artifact["replay_hash"] == review_file(SAMPLE, Config())["replay_hash"]


def test_governance_stamp_uses_real_desi_api():
    artifact = review_file(SAMPLE, Config())
    gov = artifact["governance"]
    assert gov["library"] == "desi-governance"
    assert gov["core_identity"] == 1.0
    assert "does not validate itself" in gov["audit_framing"]
    # The sample is free of DESi hype/forbidden terms.
    assert gov["forbidden_term_hits"] == []


def test_matches_committed_golden_artifact():
    if not GOLDEN_JSON.exists():
        return  # golden generated as a build step; skip if absent
    fresh = artifact_json(review_file(SAMPLE, load_config()))
    assert fresh == GOLDEN_JSON.read_text(encoding="utf-8")
    # Sanity: the committed golden is valid JSON with the right verdict.
    assert json.loads(fresh)["verdict"] == "REVIEW_ASSISTANCE_ONLY"


def test_matches_committed_golden_report():
    if not GOLDEN_REPORT.exists():
        return
    fresh = render_report(review_file(SAMPLE, load_config()))
    assert fresh == GOLDEN_REPORT.read_text(encoding="utf-8")
