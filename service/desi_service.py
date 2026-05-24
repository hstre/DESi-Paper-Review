"""DESi governance microservice.

A tiny stdlib HTTP service that exposes the REAL desi-governance public
API so that a non-Python client (here: the Go pipeline) can use the
genuine governance primitives instead of re-implementing (faking) them.

Nothing is faked: every endpoint delegates to the installed
desi-governance library.

Endpoints:
  GET  /health          -> core_identity, audit_framing, library, version
  POST /forbidden-hits  {"text": "..."}  -> {"hits": [...]}
  POST /replay-hash     {"obj": <json>}  -> {"replay_hash": "..."}
  POST /canonical-json  {"obj": <json>}  -> {"canonical_json": "..."}

The service is offline: it binds to localhost and performs no outbound
network access.
"""
from __future__ import annotations

import argparse
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from importlib import metadata

# Real desi-governance public API (never faked).
from desi.core.governance_core import core_identity
from desi.core.replay_kernel import canonical_json, replay_hash
from desi.reviewer.reviewer_port import AUDIT_FRAMING
from desi.scientific_rendering import forbidden_hits

try:
    _DESI_VERSION = metadata.version("desi-governance")
except metadata.PackageNotFoundError:  # pragma: no cover
    _DESI_VERSION = "unknown"

LIBRARY = "desi-governance"
MAX_BODY_BYTES = 8 * 1024 * 1024


class _Handler(BaseHTTPRequestHandler):
    server_version = "DESiGovernanceService/1.0"
    protocol_version = "HTTP/1.1"

    def log_message(self, *args, **kwargs):  # noqa: D401 - silence default log
        return

    def _send(self, code: int, payload: dict) -> None:
        body = json.dumps(payload).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _read_json(self) -> dict:
        length = int(self.headers.get("Content-Length", 0))
        if length <= 0:
            return {}
        if length > MAX_BODY_BYTES:
            raise ValueError("request body too large")
        raw = self.rfile.read(length)
        return json.loads(raw.decode("utf-8"))

    def do_GET(self) -> None:
        if self.path == "/health":
            self._send(
                200,
                {
                    "library": LIBRARY,
                    "version": _DESI_VERSION,
                    "desi_available": True,
                    "core_identity": core_identity(),
                    "audit_framing": AUDIT_FRAMING,
                },
            )
            return
        self._send(404, {"error": "not found", "path": self.path})

    def do_POST(self) -> None:
        try:
            data = self._read_json()
        except (ValueError, json.JSONDecodeError) as exc:
            self._send(400, {"error": f"invalid request body: {exc}"})
            return

        if self.path == "/forbidden-hits":
            text = data.get("text", "")
            if not isinstance(text, str):
                self._send(400, {"error": "'text' must be a string"})
                return
            self._send(200, {"hits": list(forbidden_hits(text))})
            return

        if self.path == "/replay-hash":
            if "obj" not in data:
                self._send(400, {"error": "missing 'obj'"})
                return
            self._send(200, {"replay_hash": replay_hash(data["obj"])})
            return

        if self.path == "/canonical-json":
            if "obj" not in data:
                self._send(400, {"error": "missing 'obj'"})
                return
            self._send(200, {"canonical_json": canonical_json(data["obj"])})
            return

        self._send(404, {"error": "not found", "path": self.path})


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="DESi governance microservice")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=8765)
    args = parser.parse_args(argv)

    # Fail fast if the protected core is not intact.
    identity = core_identity()
    if identity != 1.0:
        raise SystemExit(
            f"refusing to start: core_identity={identity!r} (expected 1.0)"
        )

    server = ThreadingHTTPServer((args.host, args.port), _Handler)
    print(
        f"DESi governance service ({LIBRARY} {_DESI_VERSION}) listening on "
        f"http://{args.host}:{args.port}  core_identity={identity}",
        flush=True,
    )
    try:
        server.serve_forever()
    except KeyboardInterrupt:  # pragma: no cover
        pass
    finally:
        server.server_close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
