# retrieval-q2

Eval code for `.compass/experiments/2026-10-22-retrieval-q2-keywords.md`, not part of compass.
`hygiene.py` is the frozen hygiene ranker. `frozen.sha256` fingerprints the inputs fixed before
any scoring: the Q2 tasks, queries and keyword lines, the agents' prompts, and the rankers' code.
The inputs themselves paraphrase a private repository, so they stay outside git.
