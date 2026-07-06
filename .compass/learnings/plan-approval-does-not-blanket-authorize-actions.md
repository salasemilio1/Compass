# plan approval doesn't blanket-authorize every action named in it

Approving a plan via ExitPlanMode doesn't make every subsequent tool call in that plan pre-authorized --
the auto-mode permission classifier can still independently deny a specific action (writing
`.git/hooks/pre-commit`, in this case) even though it was explicitly named and approved in the plan
text. Treat each risky action as needing its own confirmation at execution time; don't try to route
around a denial with a different tool -- surface it and ask again.
