# guard command substitutions that may legitimately return empty under `set -euo pipefail`

With `pipefail` set, `var=$(grep ... | awk ...)` exits the whole script if `grep` finds nothing --
even though `awk` on empty input exits 0 -- because the assignment's exit status is the pipeline's,
and pipefail makes that the *first* nonzero exit in the pipe, not the last command's. Hit this writing
the `Commit:` staleness check in `hooks/compass-session-start.sh`. Guard with `|| true` on the whole
assignment whenever "not found" is an expected, non-error outcome.
