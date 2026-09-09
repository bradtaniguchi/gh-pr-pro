## Description
<!-- Provide a brief explanation of the changes introduced in this PR. Link relevant issues below. -->
Fixes #

## Type of Change
<!-- Please check the relevant option(s) that describe this pull request. -->
- [ ] Bug fix (non-breaking change fixing an issue)
- [ ] New feature / metric (non-breaking addition to functionality)
- [ ] Breaking change (fix or feature modifying existing public CLI behavior, flags, or output schemas)
- [ ] Documentation update
- [ ] Maintenance / chore / CI

## Contributor Verification Checklist
<!-- Please complete the following checklist before requesting a review. -->
- [ ] I have run `make check` locally and all checks (vet, test, audit-readme, audit-schema) pass.
- [ ] All unit tests pass with race detector enabled (`make test` or `go test -race ./...`).
- [ ] If changing or adding CLI commands/flags, I have updated `README.md` and verified with `make audit-readme`.
- [ ] If changing output serialization schemas, I have verified with `make audit-schema`.
- [ ] I have added appropriate tests covering bug fixes or new functionality.
- [ ] I have reviewed the [Code of Conduct](CODE_OF_CONDUCT.md) and agree to follow its guidelines.

## Additional Notes
<!-- Include any extra testing details (e.g. commands run against public repos with -R), design rationale, or screenshots. -->
