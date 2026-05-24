from __future__ import annotations

import json
from pathlib import Path

from paper_review.cli import main

SAMPLE = str(Path(__file__).resolve().parents[1] / "examples" / "sample_paper.md")


def test_review_runs_offline_and_prints_report(capsys):
    rc = main(["review", SAMPLE])
    out = capsys.readouterr().out
    assert rc == 0
    assert "REVIEW_ASSISTANCE_ONLY" in out
    assert "## Scope" in out
    assert "## Questions for Human Reviewer" in out


def test_review_writes_json_and_report(tmp_path, capsys):
    json_path = tmp_path / "out.json"
    report_path = tmp_path / "report.md"
    rc = main(["review", SAMPLE, "--json", str(json_path), "--output", str(report_path)])
    assert rc == 0
    capsys.readouterr()

    artifact = json.loads(json_path.read_text(encoding="utf-8"))
    assert artifact["verdict"] == "REVIEW_ASSISTANCE_ONLY"
    assert artifact["reproducibility_risks"]
    assert "replay_hash" in artifact

    report = report_path.read_text(encoding="utf-8")
    for header in (
        "## Scope",
        "## Main Claims",
        "## Evidence Gaps",
        "## Overclaim Risks",
        "## Reproducibility Risks",
        "## Questions for Human Reviewer",
        "## Limitations of This Review",
    ):
        assert header in report


def test_doctor_reports_ready(capsys):
    rc = main(["doctor"])
    out = capsys.readouterr().out
    assert rc == 0
    last = out.strip().splitlines()[-1]
    assert last == "DESI_PAPER_REVIEW_MVP_READY"


def test_config_command_emits_no_secret(capsys):
    rc = main(["config"])
    out = capsys.readouterr().out
    assert rc == 0
    cfg = json.loads(out)
    assert cfg["offline_mode"] is True
    assert cfg["live_calls_enabled"] is False
    # Only the env-var NAME is present, never a key value.
    assert cfg["api_key_env"] == "DESI_PAPER_REVIEW_API_KEY"
    assert "api_key" not in {k for k in cfg if k not in ("api_key_env", "api_key_present")}


def test_missing_file_returns_error(capsys):
    rc = main(["review", "does_not_exist_12345.md"])
    err = capsys.readouterr().err
    assert rc == 1
    assert "not found" in err
