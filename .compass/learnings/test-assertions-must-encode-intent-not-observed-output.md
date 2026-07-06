# assert what the code should do, not what it currently does

While writing calibration tests for `check_similarity.py` (real-content dedup hard cases), several
`FakeEmbedder` mock vectors were initially chosen just to make a specific code path execute (e.g. "any
values that reach the embedding branch"), rather than to represent what a correctly-behaving embedder
*should* return for that input pair given the pair's actual ground-truth similarity. The user caught
this immediately and called it a hard antipattern: a test that bends its expected/mocked values to match
implementation behavior can pass while the underlying logic is wrong, and a green suite then lies about
correctness.

**How to apply:** decide the correct expected output from the spec/intent *before* looking at what the
code produces. If a mock feeds an assertion (e.g. injected vectors, fake API responses), set the mock to
what a correctly-behaving real dependency would return for that specific input — not to whatever value
happens to make the assertion pass. If a test then fails, that is a signal to go fix the code, not to
loosen or reshape the assertion. Encoded as rule 6 in the `new-test`/`audit-tests` skills.
