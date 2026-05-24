"""Deterministic claim extraction and classification.

Sentences are split deterministically, classified into a fixed taxonomy
by keyword signals checked in a fixed priority order, and scanned for a
fixed set of paper-level overclaim terms. Nothing here makes a judgement
about whether a claim is *true*; it only surfaces structure and language
for a human reviewer.
"""
from __future__ import annotations

import re
from dataclasses import dataclass

from .parser import ParsedPaper

CLAIM_CATEGORIES = (
    "main_claim",
    "evidence_claim",
    "method_claim",
    "result_claim",
    "limitation_claim",
    "novelty_claim",
    "generalization_claim",
)

# Strong assertions that demand evidence.
STRONG_CATEGORIES = (
    "main_claim",
    "result_claim",
    "novelty_claim",
    "generalization_claim",
)

# Paper-level overclaim terms (distinct from DESi's hype/forbidden terms).
OVERCLAIM_TERMS = (
    "first",
    "novel",
    "robust",
    "significant",
    "generalizes",
    "solves",
    "proves",
)

# Precise, inflection-aware patterns. Crafted to avoid false positives
# (e.g. "proves" must not match "provide").
_OVERCLAIM_PATTERNS = {
    "first": r"\bfirst\b",
    "novel": r"\bnovel(?:ty)?\b",
    "robust": r"\brobust(?:ness|ly)?\b",
    "significant": r"\bsignificant(?:ly)?\b",
    "generalizes": r"\bgeneraliz(?:e|es|ed|ing|ation|able)\b",
    "solves": r"\bsolv(?:e|es|ed|ing)\b",
    "proves": r"\bprov(?:e|es|ed|ing|en)\b",
}
_OVERCLAIM_RES = {
    term: re.compile(pat, re.IGNORECASE)
    for term, pat in _OVERCLAIM_PATTERNS.items()
}

# Category signals, checked top to bottom; first match wins.
_CATEGORY_SIGNALS: tuple[tuple[str, tuple[str, ...]], ...] = (
    (
        "novelty_claim",
        (r"\bnovel(?:ty)?\b", r"\bfirst\b", r"\bunprecedented\b",
         r"\bfor the first time\b"),
    ),
    (
        "generalization_claim",
        (r"\bgeneraliz", r"\buniversal", r"\bacross (?:all )?(?:tasks|domains)\b",
         r"\bin general\b", r"\bany (?:task|domain|input)\b"),
    ),
    (
        "limitation_claim",
        (r"\blimitation", r"\bhowever\b", r"\bcaveat", r"\bdoes not\b",
         r"\bcannot\b", r"\bfails?\b", r"\bfuture work\b"),
    ),
    (
        "method_claim",
        (r"\bwe propose\b", r"\bwe introduce\b", r"\bwe present\b",
         r"\bwe design\b", r"\bour (?:method|approach|model|algorithm)\b",
         r"\balgorithm\b", r"\bmethod(?:ology)?\b"),
    ),
    (
        "result_claim",
        (r"\bresults?\b", r"\bwe achieve\b", r"\boutperform",
         r"\baccuracy\b", r"\bperformance\b", r"\bimprov", r"\d+(?:\.\d+)?\s*%"),
    ),
    (
        "evidence_claim",
        (r"\bwe show\b", r"\bwe observe\b", r"\bdemonstrat",
         r"\bexperiments?\b", r"\bevidence\b", r"\bevaluat"),
    ),
    (
        "main_claim",
        (r"\bprov(?:e|es|ed|ing|en)\b", r"\bsolv(?:e|es|ed|ing)\b",
         r"\bestablish", r"\bsignificant", r"\brobust\b"),
    ),
)
_COMPILED_SIGNALS = tuple(
    (cat, tuple(re.compile(p, re.IGNORECASE) for p in pats))
    for cat, pats in _CATEGORY_SIGNALS
)

# Inline support signals: a number, a citation, a table/figure, a stats cue.
_SUPPORT_RE = re.compile(
    r"\d|\btable\b|\bfigure\b|\bfig\.\b|\[\d+\]|\bp\s*[<=]|"
    r"\bconfidence interval\b|\bappendix\b",
    re.IGNORECASE,
)

_ASSERTIVE_SECTIONS = ("abstract", "introduction", "conclusion")
_MIN_SENTENCE_LEN = 12


@dataclass(frozen=True)
class Claim:
    claim_id: str
    text: str
    category: str
    section: str


def split_sentences(text: str) -> list[str]:
    collapsed = re.sub(r"\s+", " ", text).strip()
    if not collapsed:
        return []
    parts = re.split(r"(?<=[.!?])\s+", collapsed)
    return [p.strip() for p in parts if p.strip()]


def classify_sentence(sentence: str) -> str | None:
    for category, patterns in _COMPILED_SIGNALS:
        for pat in patterns:
            if pat.search(sentence):
                return category
    return None


def overclaim_terms_in(text: str) -> tuple[str, ...]:
    """Overclaim terms present in the text, in canonical task order."""
    return tuple(
        term for term in OVERCLAIM_TERMS if _OVERCLAIM_RES[term].search(text)
    )


def has_inline_support(sentence: str) -> bool:
    return bool(_SUPPORT_RE.search(sentence))


def extract_claims(paper: ParsedPaper) -> list[Claim]:
    claims: list[Claim] = []
    counter = 0
    for sec in paper.sections:
        sec_low = sec.title.lower()
        assertive = any(k in sec_low for k in _ASSERTIVE_SECTIONS)
        for sentence in split_sentences(sec.text):
            if len(sentence) < _MIN_SENTENCE_LEN:
                continue
            category = classify_sentence(sentence)
            if category is None:
                if not assertive:
                    continue
                category = "main_claim"
            counter += 1
            claims.append(
                Claim(
                    claim_id=f"C{counter:03d}",
                    text=sentence,
                    category=category,
                    section=sec.title,
                )
            )
    return claims


__all__ = [
    "CLAIM_CATEGORIES",
    "Claim",
    "OVERCLAIM_TERMS",
    "STRONG_CATEGORIES",
    "classify_sentence",
    "extract_claims",
    "has_inline_support",
    "overclaim_terms_in",
    "split_sentences",
]
