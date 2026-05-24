"""Guarantees: no network by default, and no API-key leakage."""
from __future__ import annotations

import json
import socket
from pathlib import Path

from paper_review.cli import main
from paper_review.config import DEFAULT_API_KEY_ENV, Config, load_config
from paper_review.report_renderer import render_report
from paper_review.review_pipeline import artifact_json, review_file

SAMPLE = Path(__file__).resolve().parents[1] / "examples" / "sample_paper.md"
SENTINEL = "sk-LEAKCANARY-DO-NOT-EMIT-0xDEADBEEF"


def test_review_makes_no_network_calls(monkeypatch):
    def deny(*args, **kwargs):  # pragma: no cover - must never run
        raise AssertionError("network access attempted during offline review")

    monkeypatch.setattr(socket, "socket", deny)
    monkeypatch.setattr(socket, "create_connection", deny)

    artifact = review_file(SAMPLE, Config())
    assert artifact["verdict"] == "REVIEW_ASSISTANCE_ONLY"
    assert artifact["governance"]["mode"]["live_calls_enabled"] is False


def test_offline_is_the_default():
    cfg = load_config()
    assert cfg.offline_mode is True
    assert cfg.allow_live_llm_calls is False
    assert cfg.live_calls_enabled is False


def test_api_key_never_leaks_into_outputs(monkeypatch):
    monkeypatch.setenv(DEFAULT_API_KEY_ENV, SENTINEL)

    artifact = review_file(SAMPLE, Config())
    assert SENTINEL not in artifact_json(artifact)
    assert SENTINEL not in render_report(artifact)

    safe = Config().safe_dict()
    assert SENTINEL not in json.dumps(safe)
    # Presence is reported as a boolean, never the value.
    assert safe["api_key_present"] is True


def test_api_key_never_leaks_via_cli(monkeypatch, capsys):
    monkeypatch.setenv(DEFAULT_API_KEY_ENV, SENTINEL)

    assert main(["config"]) == 0
    assert SENTINEL not in capsys.readouterr().out

    assert main(["review", str(SAMPLE)]) == 0
    assert SENTINEL not in capsys.readouterr().out
