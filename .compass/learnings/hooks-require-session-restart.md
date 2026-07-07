Claude Code loads hook configuration (`settings.json` `hooks`) once when a session starts and cannot
hot-swap it. Adding, removing, or editing a hook registration mid-session has no effect on that
session — only on ones started afterward. When installing or changing a hook (e.g. via `install.sh`),
tell the user a new session is required before verifying it fires, rather than expecting it to activate
immediately in the current one.
