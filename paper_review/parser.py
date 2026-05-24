"""Deterministic paper parser for .md / .txt sources.

Splits a paper into a title and an ordered list of sections. Section
boundaries are markdown headers (``#`` .. ``######``) or, for plain
text, lines that are exactly a well-known section heading. No network,
no randomness, no timestamps - parsing the same bytes always yields the
same structure.
"""
from __future__ import annotations

import re
from dataclasses import dataclass
from pathlib import Path

SUPPORTED_SUFFIXES = (".md", ".markdown", ".txt")

KNOWN_SECTIONS = (
    "abstract",
    "introduction",
    "background",
    "related work",
    "method",
    "methods",
    "methodology",
    "approach",
    "experimental setup",
    "experiments",
    "evaluation",
    "results",
    "analysis",
    "discussion",
    "conclusion",
    "conclusions",
    "limitations",
    "future work",
    "references",
    "acknowledgements",
    "acknowledgments",
)

_HEADER_RE = re.compile(r"^(#{1,6})\s+(.*\S)\s*$")


@dataclass(frozen=True)
class Section:
    title: str
    level: int
    text: str


@dataclass(frozen=True)
class ParsedPaper:
    title: str
    sections: tuple[Section, ...]
    raw_text: str

    def section_titles(self) -> tuple[str, ...]:
        return tuple(s.title for s in self.sections)

    def full_text(self) -> str:
        return self.raw_text


def parse_paper(path: str | Path) -> ParsedPaper:
    p = Path(path)
    if p.suffix.lower() not in SUPPORTED_SUFFIXES:
        raise ValueError(
            f"unsupported file type {p.suffix!r}; expected one of "
            f"{', '.join(SUPPORTED_SUFFIXES)}"
        )
    return parse_text(p.read_text(encoding="utf-8"))


def parse_text(text: str) -> ParsedPaper:
    norm = text.replace("\r\n", "\n").replace("\r", "\n")
    lines = norm.split("\n")
    title = _detect_title(lines)
    sections = _split_sections(lines)
    return ParsedPaper(title=title, sections=tuple(sections), raw_text=norm)


def _detect_title(lines: list[str]) -> str:
    for ln in lines:
        m = _HEADER_RE.match(ln)
        if m and len(m.group(1)) == 1:
            return m.group(2).strip()
    for ln in lines:
        if ln.strip():
            return ln.strip().lstrip("#").strip()
    return "Untitled"


def _heading_of(line: str) -> tuple[int, str] | None:
    m = _HEADER_RE.match(line)
    if m:
        return len(m.group(1)), m.group(2).strip()
    s = line.strip()
    if s and len(s) <= 60 and s.rstrip(":").strip().lower() in KNOWN_SECTIONS:
        return 2, s.rstrip(":").strip()
    return None


def _split_sections(lines: list[str]) -> list[Section]:
    sections: list[Section] = []
    cur_title: str | None = None
    cur_level = 1
    cur_body: list[str] = []

    def flush() -> None:
        text = "\n".join(cur_body).strip()
        if cur_title is not None:
            sections.append(Section(cur_title, cur_level, text))
        elif text:
            sections.append(Section("Preamble", 1, text))

    for ln in lines:
        heading = _heading_of(ln)
        if heading is not None:
            flush()
            cur_level, cur_title = heading
            cur_body = []
        else:
            cur_body.append(ln)
    flush()
    return sections


__all__ = [
    "KNOWN_SECTIONS",
    "ParsedPaper",
    "Section",
    "SUPPORTED_SUFFIXES",
    "parse_paper",
    "parse_text",
]
