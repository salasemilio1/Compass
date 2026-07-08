# check existing learnings before "fixing" something that looks like a typo

A doubled letter (`graphifyy`) reads as an obvious typo on sight. An audit agent flagged it as one and
it was "fixed" to `graphify` in install.sh -- but `graphifyy` is the real PyPI package name (it installs
a `graphify` command), already documented independently in two places:
[[graphify-cli-quirks]] and [[verify-flagged-installs-directly]].

**How to apply:** before "correcting" something that looks wrong by inspection alone (unusual spelling,
inconsistent casing, a name that doesn't match its command), grep `.compass/learnings/` for it first.
If a past session already investigated and documented the answer, trust that over a fresh visual
impression.
