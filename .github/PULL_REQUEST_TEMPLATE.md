## What does this change?

<!-- A clear description of the change and why it is needed. -->

## Related issue

<!-- Fixes #123 -->

## Type of change

- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation
- [ ] Refactor / internal cleanup
- [ ] Security fix

## How was this tested?

<!-- Describe what you ran, and on what. Include distribution and Go version. -->

- [ ] `make test` passes
- [ ] `make vet` passes
- [ ] `make fmt` produces no changes
- [ ] Added tests covering the new behaviour, including failure cases
- [ ] Tested manually against a real server (describe how)

## Project rules

Confirm this change does not break the architectural commitments in
[CONTRIBUTING.md](CONTRIBUTING.md):

- [ ] No Docker, containers or container-based deployment introduced
- [ ] Nothing requires root
- [ ] tmux is not treated as authoritative process state or used for lifecycle control
- [ ] No shell exposure; no concatenated shell strings
- [ ] Minecraft files remain the filesystem's source of truth
- [ ] Paths are stored relative to `BONGHOS_HOME` and validated canonically
- [ ] Permissions are enforced in the backend
- [ ] No secrets are logged
- [ ] No required telemetry

## Documentation

- [ ] `Tutorial.txt` updated (if commands, options or behaviour changed)
- [ ] `README.md` updated (if features or setup changed)
- [ ] `source/docs/CHANGELOG.md` updated

## Anything else?

<!-- Screenshots, trade-offs, follow-up work, known gaps. -->
