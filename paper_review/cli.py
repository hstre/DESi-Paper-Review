"""Command-line interface for DESi Paper Review.

Commands:
  desi-paper-review review <file> [--output report.md] [--json out.json]
  desi-paper-review doctor
  desi-paper-review config
"""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

from . import VERDICT, __version__
from .config import load_config
from .report_renderer import render_report
from .review_pipeline import GovernanceError, artifact_json, review_file, review_text

# A tiny built-in paper used by `doctor` to exercise the pipeline offline.
_DOCTOR_SAMPLE = (
    "# Probe Paper\n\n## Abstract\nWe present the first method that "
    "solves the task and proves robust across settings.\n\n## Results\n"
    "Our approach reaches high accuracy on a dataset.\n"
)


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="desi-paper-review",
        description=(
            "Reviewer ASSISTANT (not a peer reviewer). Offline, "
            "deterministic. Only verdict: REVIEW_ASSISTANCE_ONLY."
        ),
    )
    parser.add_argument(
        "--version", action="version", version=f"desi-paper-review {__version__}"
    )
    sub = parser.add_subparsers(dest="command")

    p_review = sub.add_parser(
        "review", help="review a paper (.md/.txt) and emit a report"
    )
    p_review.add_argument("file", help="path to the paper (.md/.txt)")
    p_review.add_argument(
        "--output", "-o", help="write the Markdown report to this path"
    )
    p_review.add_argument(
        "--json", help="write the JSON artifact to this path"
    )
    p_review.add_argument(
        "--config", help="path to a config .ini (defaults to repo config)"
    )

    p_doctor = sub.add_parser(
        "doctor", help="run environment / self checks"
    )
    p_doctor.add_argument("--config", help="path to a config .ini")

    p_config = sub.add_parser(
        "config", help="print the resolved, secret-free configuration"
    )
    p_config.add_argument("--config", help="path to a config .ini")

    return parser


def _cmd_review(args: argparse.Namespace) -> int:
    cfg = load_config(args.config)
    try:
        artifact = review_file(args.file, cfg)
    except FileNotFoundError:
        sys.stderr.write(f"error: file not found: {args.file}\n")
        return 1
    except ValueError as exc:
        sys.stderr.write(f"error: {exc}\n")
        return 1
    except GovernanceError as exc:
        sys.stderr.write(f"governance error: {exc}\n")
        return 1

    report = render_report(artifact)

    if args.json:
        Path(args.json).write_text(artifact_json(artifact), encoding="utf-8")
    if args.output:
        Path(args.output).write_text(report, encoding="utf-8")
        sys.stdout.write(f"Wrote report to {args.output}\n")
        if args.json:
            sys.stdout.write(f"Wrote JSON artifact to {args.json}\n")
    else:
        sys.stdout.write(report)
        if not report.endswith("\n"):
            sys.stdout.write("\n")
        if args.json:
            sys.stdout.write(f"Wrote JSON artifact to {args.json}\n")
    return 0


def _cmd_config(args: argparse.Namespace) -> int:
    cfg = load_config(args.config)
    # safe_dict contains NO secret - only the env-var name and a presence flag.
    sys.stdout.write(artifact_json(cfg.safe_dict()))
    return 0


def _cmd_doctor(args: argparse.Namespace) -> int:
    ok = True
    out: list[str] = []

    try:
        import desi  # noqa: F401
        from desi.core.governance_core import core_identity

        out.append("[ok] desi-governance is importable")
    except Exception as exc:  # pragma: no cover - import failure path
        out.append(f"[FAIL] cannot import desi-governance: {exc}")
        for line in out:
            sys.stdout.write(line + "\n")
        sys.stdout.write("\nDESI_PAPER_REVIEW_MVP_NOT_READY\n")
        return 1

    identity = core_identity()
    if identity == 1.0:
        out.append(f"[ok] protected-core identity = {identity}")
    else:
        ok = False
        out.append(f"[FAIL] core_identity={identity} (expected 1.0)")

    cfg = load_config(args.config)
    if cfg.live_calls_enabled:
        out.append(
            "[warn] live LLM calls are ENABLED by config "
            "(both gates open)"
        )
    else:
        out.append("[ok] offline mode active (no live LLM calls)")

    try:
        a1 = review_text(_DOCTOR_SAMPLE, cfg)
        a2 = review_text(_DOCTOR_SAMPLE, cfg)
        if a1["verdict"] != VERDICT:
            ok = False
            out.append(
                f"[FAIL] unexpected verdict {a1['verdict']!r}"
            )
        elif artifact_json(a1) != artifact_json(a2):
            ok = False
            out.append("[FAIL] pipeline output is not replay-stable")
        else:
            out.append(
                "[ok] pipeline runs offline and is replay-stable "
                f"(hash {a1['replay_hash'][:12]}...)"
            )
    except Exception as exc:  # pragma: no cover - defensive
        ok = False
        out.append(f"[FAIL] pipeline error: {exc}")

    for line in out:
        sys.stdout.write(line + "\n")
    sys.stdout.write("\n")
    if ok:
        sys.stdout.write("DESI_PAPER_REVIEW_MVP_READY\n")
        return 0
    sys.stdout.write("DESI_PAPER_REVIEW_MVP_NOT_READY\n")
    return 1


def main(argv: list[str] | None = None) -> int:
    parser = _build_parser()
    args = parser.parse_args(argv)
    if args.command == "review":
        return _cmd_review(args)
    if args.command == "doctor":
        return _cmd_doctor(args)
    if args.command == "config":
        return _cmd_config(args)
    parser.print_help()
    return 2


if __name__ == "__main__":  # pragma: no cover
    raise SystemExit(main())
