# DESi Paper Review - Go web UI

A browser front-end and a deterministic, offline **Go re-implementation**
of the DESi Paper Review pipeline.

It is the same reviewer **assistant** as the Python tool (single verdict
`REVIEW_ASSISTANCE_ONLY`), but the pipeline (parsing, claim extraction,
overclaim detection, evidence gaps, reproducibility risks, reviewer
questions) is written in Go.

## Architecture: Go pipeline + real DESi via a microservice

The governance primitives are **not** re-implemented (faked) in Go. They
come from the genuine `desi-governance` library, exposed over HTTP by a
tiny Python microservice ([`../service/desi_service.py`](../service/desi_service.py)):

| Primitive | Source |
|---|---|
| `forbidden_hits` (hype scan) | real `desi.scientific_rendering.forbidden_hits` |
| `replay_hash` (byte-stable hash) | real `desi.core.replay_kernel.replay_hash` |
| `canonical_json` (byte-stable JSON) | real `desi.core.replay_kernel.canonical_json` |
| `core_identity` (protected-core gate) | real `desi.core.governance_core.core_identity` |
| `AUDIT_FRAMING` | real `desi.reviewer.reviewer_port.AUDIT_FRAMING` |

```
browser  ──HTTP──▶  Go web UI + Go pipeline  ──HTTP──▶  Python DESi service ──▶ desi-governance
```

The Go pipeline builds the artifact body, then asks the service for the
real `replay_hash` and `canonical_json`, and refuses to emit anything if
`core_identity() != 1.0`.

## Run it

Quickest path (from the repo root):

```bash
make dev      # starts the DESi service + web UI together; Ctrl-C stops both
make check    # full health gate: service + python doctor + go self-check
make test     # run Python and Go tests
```

Or manually:

```bash
# 1. install the governance library and start the DESi microservice
pip install -e ..          # installs desi-paper-review (and pulls desi-governance)
pip install -e ../../DESi  # or wherever hstre/DESi is checked out
python ../service/desi_service.py --port 8765

# 2. build and start the web UI (in another shell)
cd web
go build -o dpr-web .
./dpr-web -addr :8080 -desi http://127.0.0.1:8765
# open http://localhost:8080
```

Click **Load sample** to fill the form with a paper that contains
deliberate overclaims and reproducibility gaps.

## Self-check (headless)

```bash
./dpr-web -check -desi http://127.0.0.1:8765
```

Pings the DESi service, runs the pipeline on the built-in sample twice,
and verifies byte-stable output. Ends with `DESI_PAPER_REVIEW_WEB_READY`
or `DESI_PAPER_REVIEW_WEB_NOT_READY`.

## Layout

```
web/
  main.go                     entrypoint, flags, -check self-test
  internal/desi/client.go     HTTP client for the DESi microservice
  internal/review/            Go pipeline (parser, claims, repro, pipeline, report)
  internal/server/            web server + embedded HTML UI
```

## Offline & secrets

Offline by default (`offline_mode=true`, `allow_live_llm_calls=false`).
The Go pipeline talks only to the local DESi service; it makes no other
network calls and never handles API keys.
