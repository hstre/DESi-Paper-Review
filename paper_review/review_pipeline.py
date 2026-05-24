"""The offline, deterministic review pipeline.

Stages: parse -> extract & classify claims -> flag overclaims ->
find evidence gaps -> detect reproducibility risks -> generate reviewer
questions -> assemble a replay-stable artifact.

Real desi-governance public API is used (never faked):
  * forbidden_hits   - hype / forbidden-term scan over the paper text
  * core_identity    - protected-core gate; the pipeline REFUSES to emit
                       an artifact if this is not exactly 1.0
  * canonical_json   - byte-stable serialization
  * replay_hash      - byte-stable artifact hash
  * AUDIT_FRAMING    - the audit-framing statement embedded in the stamp
"""
from __future__ import annotations

import re
from pathlib import Path

from desi.core.governance_core import core_identity
from desi.core.replay_kernel import canonical_json, replay_hash
from desi.reviewer.reviewer_port import AUDIT_FRAMING
from desi.scientific_rendering import forbidden_hits

from . import VERDICT, __version__
from .claim_extractor import (
    STRONG_CATEGORIES,
    Claim,
    extract_claims,
    has_inline_support,
    overclaim_terms_in,
)
from .config import Config, load_config
from .parser import ParsedPaper, parse_paper, parse_text

REPRODUCIBILITY_RISK_TYPES = (
    "missing_data",
    "missing_code",
    "missing_baselines",
    "missing_parameters",
    "unclear_dataset",
    "unsupported_metrics",
)

_DISCLAIMER = (
    "This artifact assists human review only. It never accepts, rejects, "
    "or validates a paper, never replaces a human reviewer, and never "
    "determines truth or guarantees correctness."
)

_REVIEWER_QUESTION_FOR_RISK = {
    "missing_code": (
        "Can the authors provide a link to the source code required to "
        "reproduce the reported results?"
    ),
    "missing_data": (
        "Are the underlying data available for independent inspection?"
    ),
    "missing_baselines": (
        "Which baselines were used, and how does the method compare "
        "against prior work?"
    ),
    "missing_parameters": (
        "What exact hyperparameters, seeds, and training/configuration "
        "settings were used?"
    ),
    "unclear_dataset": (
        "Which dataset (name, size, and splits) was used, and how was it "
        "constructed?"
    ),
    "unsupported_metrics": (
        "How were the reported metrics computed (evaluation protocol, "
        "test set, and variance or significance)?"
    ),
}


class GovernanceError(RuntimeError):
    """Raised when the protected-core identity gate fails."""


def _has_any(haystack: str, *needles: str) -> bool:
    return any(n in haystack for n in needles)


def detect_reproducibility_risks(text: str) -> list[dict]:
    """Deterministic reproducibility-risk scan over the full paper text."""
    low = text.lower()
    risks: list[dict] = []

    if not _has_any(
        low, "github", "gitlab", "code is available", "code available",
        "source code", "open-source", "open source", "code repository",
        "implementation is available", "zenodo",
    ):
        risks.append(
            {
                "risk_type": "missing_code",
                "detail": "No link or statement indicating the source "
                "code is available for reproduction.",
            }
        )

    if not _has_any(
        low, "data is available", "data are available", "data available",
        "dataset is available", "dataset available", "publicly available",
        "supplementary data", "data repository", "zenodo",
    ):
        risks.append(
            {
                "risk_type": "missing_data",
                "detail": "No statement that the underlying data are "
                "available for inspection.",
            }
        )

    if not _has_any(
        low, "baseline", "compared to", "compared with", "comparison",
        "state-of-the-art", "state of the art", "prior work", "we compare",
    ):
        risks.append(
            {
                "risk_type": "missing_baselines",
                "detail": "No baseline or comparison against prior work "
                "is described.",
            }
        )

    if not _has_any(
        low, "hyperparameter", "learning rate", "epochs", "batch size",
        "random seed", "parameter settings", "training details",
        "we set", "configuration",
    ):
        risks.append(
            {
                "risk_type": "missing_parameters",
                "detail": "No hyperparameters, seeds, or training/"
                "configuration details are reported.",
            }
        )

    mentions_dataset = _has_any(low, "dataset", "data set", "corpus")
    named = _has_any(
        low, "imagenet", "cifar", "mnist", "glue", "squad", "wikitext",
        "coco", "penn treebank", "wmt",
    )
    sized = bool(
        re.search(
            r"\b\d[\d,\.]*\s*(?:samples|examples|images|documents|"
            r"sentences|instances|rows|records)\b",
            low,
        )
    )
    if mentions_dataset and not (named or sized):
        risks.append(
            {
                "risk_type": "unclear_dataset",
                "detail": "A dataset is mentioned but not identified by "
                "name, size, or construction.",
            }
        )
    elif not mentions_dataset:
        risks.append(
            {
                "risk_type": "unclear_dataset",
                "detail": "No dataset is identified anywhere in the paper.",
            }
        )

    has_metric_words = _has_any(
        low, "accuracy", "f1", "precision", "recall", "auc", "score",
        "performance", "%", "error rate",
    )
    has_metric_method = _has_any(
        low, "evaluation protocol", "we measure", "computed as",
        "cross-validation", "cross validation", "test set", "held-out",
        "held out", "confidence interval", "standard deviation",
        "p-value", "p <", "significance test",
    )
    if has_metric_words and not has_metric_method:
        risks.append(
            {
                "risk_type": "unsupported_metrics",
                "detail": "Performance metrics are reported without a "
                "described measurement protocol or variance/significance.",
            }
        )

    return risks


def _build_overclaims(claims: list[Claim]) -> list[dict]:
    overclaims: list[dict] = []
    for claim in claims:
        terms = overclaim_terms_in(claim.text)
        if terms:
            overclaims.append(
                {
                    "claim_id": claim.claim_id,
                    "category": claim.category,
                    "section": claim.section,
                    "text": claim.text,
                    "terms": list(terms),
                }
            )
    return overclaims


def _build_unsupported(
    claims: list[Claim], overclaim_ids: set[str]
) -> tuple[list[dict], list[dict]]:
    unsupported: list[dict] = []
    evidence_gaps: list[dict] = []
    for claim in claims:
        strong = (
            claim.category in STRONG_CATEGORIES
            or claim.claim_id in overclaim_ids
        )
        if strong and not has_inline_support(claim.text):
            unsupported.append(
                {
                    "claim_id": claim.claim_id,
                    "category": claim.category,
                    "section": claim.section,
                    "text": claim.text,
                }
            )
            evidence_gaps.append(
                {
                    "claim_id": claim.claim_id,
                    "section": claim.section,
                    "missing": "inline_support",
                    "note": "Strong claim presented without inline data, "
                    "a citation, or a table/figure reference.",
                }
            )
    return unsupported, evidence_gaps


def _build_reviewer_questions(
    overclaims: list[dict],
    evidence_gaps: list[dict],
    repro_risks: list[dict],
) -> list[str]:
    questions: list[str] = []
    for oc in overclaims:
        terms = ", ".join(oc["terms"])
        questions.append(
            f"Claim {oc['claim_id']} uses strong language ({terms}). Does "
            f"the evidence justify it, or should the authors soften or "
            f"substantiate the wording?"
        )
    for gap in evidence_gaps:
        questions.append(
            f"What specific evidence supports claim {gap['claim_id']}? It "
            f"currently lacks inline data, a citation, or a table/figure "
            f"reference."
        )
    seen: set[str] = set()
    for risk in repro_risks:
        rt = risk["risk_type"]
        if rt in _REVIEWER_QUESTION_FOR_RISK and rt not in seen:
            questions.append(_REVIEWER_QUESTION_FOR_RISK[rt])
            seen.add(rt)
    if not questions:
        questions.append(
            "No automated concerns were flagged; please assess the paper "
            "on its scientific merits."
        )
    return questions


def _assemble(paper: ParsedPaper, config: Config) -> dict:
    claims = extract_claims(paper)
    full_text = paper.full_text()

    overclaims = _build_overclaims(claims)
    overclaim_ids = {oc["claim_id"] for oc in overclaims}
    unsupported, evidence_gaps = _build_unsupported(claims, overclaim_ids)
    repro_risks = detect_reproducibility_risks(full_text)
    questions = _build_reviewer_questions(
        overclaims, evidence_gaps, repro_risks
    )

    # Real DESi hype / forbidden-term scan over the paper text.
    forbidden = list(forbidden_hits(full_text))

    body = {
        "schema_version": "1.0",
        "tool": "desi-paper-review",
        "tool_version": __version__,
        "paper_title": paper.title,
        "claims": [
            {
                "claim_id": c.claim_id,
                "category": c.category,
                "section": c.section,
                "text": c.text,
            }
            for c in claims
        ],
        "unsupported_claims": unsupported,
        "overclaims": overclaims,
        "evidence_gaps": evidence_gaps,
        "reproducibility_risks": repro_risks,
        "reviewer_questions": questions,
        "verdict": VERDICT,
        "governance": {
            "library": "desi-governance",
            "core_identity": core_identity(),
            "audit_framing": AUDIT_FRAMING,
            "forbidden_term_hits": forbidden,
            "mode": {
                "offline_mode": config.offline_mode,
                "allow_live_llm_calls": config.allow_live_llm_calls,
                "live_calls_enabled": config.live_calls_enabled,
            },
            "disclaimer": _DISCLAIMER,
        },
    }
    artifact = dict(body)
    # The replay hash covers the full artifact body (everything except
    # the hash field itself); deterministic body -> deterministic hash.
    artifact["replay_hash"] = replay_hash(body)
    return artifact


def _review(paper: ParsedPaper, config: Config | None) -> dict:
    cfg = config if config is not None else load_config()

    # Protected-core gate: refuse to emit an artifact if the DESi core
    # identity is not exactly 1.0.
    identity = core_identity()
    if identity != 1.0:
        raise GovernanceError(
            "DESi protected-core identity check failed "
            f"(core_identity={identity!r}); refusing to emit a review "
            "artifact."
        )

    # The MVP pipeline is offline-only; live LLM calls are never made.
    # The two-gate flag is surfaced in the artifact for transparency.
    return _assemble(paper, cfg)


def review_text(text: str, config: Config | None = None) -> dict:
    return _review(parse_text(text), config)


def review_file(path: str | Path, config: Config | None = None) -> dict:
    return _review(parse_paper(path), config)


def artifact_json(artifact: dict) -> str:
    """Byte-stable JSON serialization of an artifact (DESi canonical form)."""
    return canonical_json(artifact)


__all__ = [
    "GovernanceError",
    "REPRODUCIBILITY_RISK_TYPES",
    "artifact_json",
    "detect_reproducibility_risks",
    "review_file",
    "review_text",
]
