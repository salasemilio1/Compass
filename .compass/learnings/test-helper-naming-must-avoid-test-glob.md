# name shared test helpers outside the `test_*.sh` glob

`tests/run_tests.sh` discovers tests via `test_*.sh`. A shared assertion-helper file named
`test_helpers.sh` matched that glob and was "run" as its own test -- sourcing it standalone has no
assertions, so it silently reported PASS with zero checks, no error. Renamed to `helpers.sh`. Any
future shared bash test fixture/helper file must avoid the `test_*.sh` prefix, or the runner will
silently treat it as a passing test.
