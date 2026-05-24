from __future__ import annotations

from pathlib import Path

import pytest

from paper_review.parser import parse_paper, parse_text

SAMPLE = Path(__file__).resolve().parents[1] / "examples" / "sample_paper.md"


def test_sample_parses_title_and_sections():
    paper = parse_paper(SAMPLE)
    assert "Solves Generalization" in paper.title
    titles = {t.lower() for t in paper.section_titles()}
    for expected in ("abstract", "introduction", "method", "results", "conclusion"):
        assert expected in titles
    assert paper.full_text().strip()


def test_parse_text_detects_markdown_headers():
    paper = parse_text("# Title\n\n## Abstract\nHello world.\n\n## Results\nDone.\n")
    assert paper.title == "Title"
    assert paper.section_titles() == ("Title", "Abstract", "Results")
    abstract = next(s for s in paper.sections if s.title == "Abstract")
    assert "Hello world." in abstract.text


def test_parse_plain_text_known_headings():
    paper = parse_text("My Paper\n\nAbstract\nSome text.\n\nResults\nMore text.\n")
    titles = [t.lower() for t in paper.section_titles()]
    assert "abstract" in titles
    assert "results" in titles


def test_unsupported_suffix_raises(tmp_path):
    bad = tmp_path / "paper.pdf"
    bad.write_text("x", encoding="utf-8")
    with pytest.raises(ValueError):
        parse_paper(bad)
