# Test Layout

`ms-go-validation-orchestrator` uses a hybrid Go test layout:

- `test/unit/` for service-level unit suites
- `internal/.../*_test.go` for package-local tests that need unexported engine wiring helpers

Integration, contract, and e2e suites should also live under this `test/` tree.
