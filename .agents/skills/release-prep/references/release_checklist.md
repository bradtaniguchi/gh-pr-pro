# Extension Release Checklist & Reference

Use this checklist during every release cycle for `gh-pr-pro`.

---

## 1. Pre-Release Verification
- [ ] Working tree is clean (`git status -s` produces no output)
- [ ] On `main` branch with latest remote commits pulled
- [ ] `make check` passes (`gofmt`, `go vet`, `go test -race`, `make audit-readme`)
- [ ] `./.agents/skills/schema-regression-test/scripts/verify_schemas.sh` passes (all 7 output schemas verified)
- [ ] Cross-compilation passes for all 5 targets:
  - `linux/amd64`
  - `linux/arm64`
  - `darwin/amd64`
  - `darwin/arm64`
  - `windows/amd64`

---

## 2. Versioning & Notes
- [ ] Semantic version determined (`vMAJOR.MINOR.PATCH` with leading `v`)
- [ ] Release notes generated via `./.agents/skills/release-prep/scripts/generate_release_notes.sh`
- [ ] Breaking changes identified and documented (if major release)

---

## 3. Local Smoke Test
- [ ] Extension builds locally: `make build`
- [ ] Extension installs locally: `gh extension install .`
- [ ] Root help executes: `gh pr-pro --help`
- [ ] Sample command runs: `gh pr-pro time merge --past 7d -R cli/cli` (or `./gh-pr-pro cache list`)

---

## 4. Tag & Distribution
- [ ] Create annotated git tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
- [ ] Push tag: `git push origin vX.Y.Z`
- [ ] Monitor GitHub Actions workflow at `.github/workflows/release.yml`
- [ ] Confirm precompiled binary archives attached to the GitHub Release on GitHub
