"""DESi Paper Review - a deterministic, offline reviewer ASSISTANT.

Grounding principle: this is a reviewer ASSISTANT, not a peer reviewer.
It never accepts or rejects a paper, never replaces a human reviewer,
never determines truth, and never guarantees correctness. Its single
verdict is REVIEW_ASSISTANCE_ONLY.

The pipeline is built on the real desi-governance public API:
  * desi.scientific_rendering.forbidden_hits   (hype / forbidden-term scan)
  * desi.core.replay_kernel.replay_hash         (byte-stable artifacts)
  * desi.core.replay_kernel.canonical_json
  * desi.core.governance_core.core_identity     (protected-core gate)
  * desi.reviewer.reviewer_port.AUDIT_FRAMING   (audit framing statement)
"""
from __future__ import annotations

__version__ = "0.1.0a0"

# The only verdict this tool will ever emit.
VERDICT = "REVIEW_ASSISTANCE_ONLY"

__all__ = ["VERDICT", "__version__"]
