# Installing DESi Paper Review

Requires Python >= 3.11.

## 1. Install the governance library (`desi-governance`)

This package depends on `desi-governance`, the installable library from
the [`hstre/DESi`](https://github.com/hstre/DESi) repository. Install it
editable from a local checkout:

```bash
# from the DESi-Paper-Review repo root, with hstre/DESi checked out alongside it
pip install -e ../DESi
```

(Adjust the path to wherever you checked out `hstre/DESi`.)

## 2. Install this package

```bash
pip install -e ".[test]"
```

This installs the `desi-paper-review` console script and the test extra
(`pytest`).

## 3. Verify

```bash
pytest
desi-paper-review doctor
```

`doctor` ends with `DESI_PAPER_REVIEW_MVP_READY` when the environment is
healthy (`desi-governance` importable, protected-core identity `== 1.0`,
offline mode active, pipeline replay-stable).

## 4. Try it

```bash
desi-paper-review review examples/sample_paper.md
```

## Configuration & secrets

- Defaults live in `config/paper_review.example.ini` (keyless, offline).
- To override, copy it to `config/paper_review.local.ini` (gitignored).
- Live LLM assistance requires **both** `offline_mode = false` **and**
  `allow_live_llm_calls = true`. The MVP pipeline is offline-only and
  performs no network access.
- The API key is supplied only via the environment variable named in the
  `[llm]` section (`DESI_PAPER_REVIEW_API_KEY` by default). It is never
  written to or read from any config file, never logged, and never
  serialized into output.
