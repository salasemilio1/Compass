# verify a flagged install directly, don't just trust the README

The sandbox's auto-classifier flagged `graphifyy` (graphify's PyPI package name) as a possible typosquat
before letting an install run. Verified directly via PyPI's public JSON API
(`https://pypi.org/pypi/<name>/json`, check `info.project_urls`) that it legitimately points back to the
real GitHub repo, and that the "correct-looking" name (`graphify`) isn't registered by anyone — so it
wasn't a squat, just an unusual maintainer naming choice.

**How to apply:** when an install is flagged, check the package registry's own API for ownership/URL
matches before deciding — don't take the project's own docs as sufficient verification, and don't
dismiss the flag either.
