"""Configuration loading for DESi Paper Review.

Two independent gates protect against accidental live LLM calls:
``offline_mode`` must be False AND ``allow_live_llm_calls`` must be True
before any network use would even be considered. The MVP pipeline is
offline-only, so no network call is ever made regardless.

API keys are never read into the config object, never serialized, and
never logged. Only the NAME of the environment variable holding a key is
recorded, plus a boolean indicating whether such a variable is set.
"""
from __future__ import annotations

import configparser
import os
from dataclasses import dataclass
from pathlib import Path

DEFAULT_API_KEY_ENV = "DESI_PAPER_REVIEW_API_KEY"

_PACKAGE_ROOT = Path(__file__).resolve().parent
_PROJECT_ROOT = _PACKAGE_ROOT.parent
_CONFIG_DIR = _PROJECT_ROOT / "config"
LOCAL_CONFIG = _CONFIG_DIR / "paper_review.local.ini"
EXAMPLE_CONFIG = _CONFIG_DIR / "paper_review.example.ini"

_TRUE = {"1", "true", "yes", "on"}


def _to_bool(value: str | bool | None, default: bool) -> bool:
    if value is None:
        return default
    if isinstance(value, bool):
        return value
    return str(value).strip().lower() in _TRUE


@dataclass(frozen=True)
class Config:
    """Resolved, secret-free configuration."""

    offline_mode: bool = True
    allow_live_llm_calls: bool = False
    api_key_env: str = DEFAULT_API_KEY_ENV
    provider: str = "none"
    model: str = "none"

    @property
    def live_calls_enabled(self) -> bool:
        """Live LLM calls require BOTH gates: not offline AND allowed."""
        return (not self.offline_mode) and self.allow_live_llm_calls

    @property
    def api_key_present(self) -> bool:
        """Whether a key is available in the environment.

        The value is never returned or stored - only its presence.
        """
        return bool(os.environ.get(self.api_key_env))

    def safe_dict(self) -> dict:
        """A view that is SAFE to serialize/log: contains no secret."""
        return {
            "offline_mode": self.offline_mode,
            "allow_live_llm_calls": self.allow_live_llm_calls,
            "live_calls_enabled": self.live_calls_enabled,
            "api_key_env": self.api_key_env,
            "api_key_present": self.api_key_present,
            "provider": self.provider,
            "model": self.model,
        }


def load_config(path: str | os.PathLike[str] | None = None) -> Config:
    """Load configuration.

    Resolution order: explicit ``path`` if given, else the gitignored
    local config, else the committed keyless example, else built-in
    offline defaults.
    """
    candidates: list[Path] = []
    if path is not None:
        candidates.append(Path(path))
    else:
        candidates.append(LOCAL_CONFIG)
        candidates.append(EXAMPLE_CONFIG)
    for cand in candidates:
        if cand.is_file():
            return _parse_config_file(cand)
    return Config()


def _parse_config_file(path: Path) -> Config:
    cp = configparser.ConfigParser()
    cp.read(path, encoding="utf-8")
    defaults = Config()

    pr = cp["paper_review"] if cp.has_section("paper_review") else {}
    llm = cp["llm"] if cp.has_section("llm") else {}

    # NOTE: we deliberately read only api_key_env (the variable NAME).
    # Any literal key placed in a config file is ignored, never loaded.
    return Config(
        offline_mode=_to_bool(pr.get("offline_mode"), defaults.offline_mode),
        allow_live_llm_calls=_to_bool(
            pr.get("allow_live_llm_calls"), defaults.allow_live_llm_calls
        ),
        api_key_env=(llm.get("api_key_env") or defaults.api_key_env).strip(),
        provider=(llm.get("provider") or defaults.provider).strip(),
        model=(llm.get("model") or defaults.model).strip(),
    )


__all__ = [
    "Config",
    "DEFAULT_API_KEY_ENV",
    "EXAMPLE_CONFIG",
    "LOCAL_CONFIG",
    "load_config",
]
