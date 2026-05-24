# DESi Paper Review

A deterministic, offline **reviewer ASSISTANT** built on the
[`desi-governance`](https://github.com/hstre/DESi) library.

> **Grounding principle: this is a reviewer assistant, not a peer
> reviewer.** It never accepts or rejects a paper, never replaces a human
> reviewer, never determines truth, and never guarantees correctness. Its
> single verdict is `REVIEW_ASSISTANCE_ONLY`.

## What it does

Given a paper (`.md` / `.txt`), the pipeline runs fully offline and
deterministically:

1. **Parse** the paper into title + sections.
2. **Extract & classify claims** into
   `main_claim`, `evidence_claim`, `method_claim`, `result_claim`,
   `limitation_claim`, `novelty_claim`, `generalization_claim`.
3. **Flag overclaims** (terms: `first`, `novel`, `robust`, `significant`,
   `generalizes`, `solves`, `proves`).
4. **Find evidence gaps** — strong claims with no inline support.
5. **Detect reproducibility risks** —
   `missing_data`, `missing_code`, `missing_baselines`,
   `missing_parameters`, `unclear_dataset`, `unsupported_metrics`.
6. **Generate reviewer questions** for the human reviewer.
7. **Emit a replay-stable artifact** (JSON + Markdown report).

## Built on the real DESi public API

The pipeline uses the genuine `desi-governance` public API — nothing is
faked:

| DESi API | Use here |
|---|---|
| `desi.scientific_rendering.forbidden_hits` | hype / forbidden-term scan over the paper |
| `desi.core.replay_kernel.canonical_json` | byte-stable artifact serialization |
| `desi.core.replay_kernel.replay_hash` | byte-stable artifact hash |
| `desi.core.governance_core.core_identity` | protected-core gate; the pipeline **refuses to emit** an artifact if `core_identity() != 1.0` |
| `desi.reviewer.reviewer_port.AUDIT_FRAMING` | audit-framing statement embedded in every artifact |

## Usage

```bash
desi-paper-review review examples/sample_paper.md
desi-paper-review review paper.md --output report.md --json out.json
desi-paper-review doctor
desi-paper-review config
```

`doctor` ends with `DESI_PAPER_REVIEW_MVP_READY` or
`DESI_PAPER_REVIEW_MVP_NOT_READY`.

## Output

The JSON artifact contains: `paper_title`, `claims`,
`unsupported_claims`, `overclaims`, `evidence_gaps`,
`reproducibility_risks`, `reviewer_questions`,
`verdict = "REVIEW_ASSISTANCE_ONLY"`, a DESi governance stamp, and a
byte-stable `replay_hash`. The Markdown report has the sections: Scope /
Main Claims / Evidence Gaps / Overclaim Risks / Reproducibility Risks /
Questions for Human Reviewer / Limitations of This Review.

See [`examples/`](examples/) for `sample_paper.md` and the generated
`sample_review_output.json` / `sample_review_report.md`.

## Offline & secrets

- Offline by default: `offline_mode = true`, `allow_live_llm_calls = false`.
  Live LLM calls require **both** flags flipped; the MVP pipeline makes no
  network calls at all.
- API keys are **never** committed, logged, or serialized. The example
  config is keyless; `config/paper_review.local.ini`, `.env`, `*.key`, and
  `secrets/` are gitignored.

## Install

See [INSTALL.md](INSTALL.md).
