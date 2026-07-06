# graphify CLI quirks (for if the semantic layer is ever revisited)

Not used by compass's current wiring (deterministic-only, see decisions/0001), but worth recording since
we hit both directly:

- The Ollama backend needs `uv tool install "graphifyy[ollama]" --force` — a plain `graphifyy` install
  fails on `--backend ollama` with a missing `openai` package error, even though Ollama itself needs no
  such package conceptually.
- Flag syntax is inconsistent across subcommands: `extract`/`update` take `--backend <name>`
  (space-separated); `cluster-only`/`label` require `--backend=<name>` (equals-sign only). Using the
  wrong form doesn't error clearly — it silently falls back to auto-detect.
