# Development in Sandboxed Environments

This document records recurring development issues observed in sandboxed runs on 2026-02-25.

## Known Issues

- Default Go and golangci-lint cache locations may be read-only or blocked.
- Test, lint, and schema checks can fail even when code is correct if cache paths are not writable.
- Repository-local module caches can include read-only files that block cleanup.

## Recommended Local Cache Overrides

`Taskfile.yml` configures repo-local cache directories by default, so normal task commands already use:

- `GOCACHE=.cache/go-build`
- `GOMODCACHE=.cache/gomod`
- `GOPATH=.cache/gopath`
- `GOLANGCI_LINT_CACHE=.cache/golangci-lint`

Use Task commands as-is in sandboxed environments:

```bash
task check
```

If you run `go test` or `golangci-lint` directly without Task, set the same env vars manually:

```bash
ROOT_DIR="$(pwd)"
GOCACHE="$ROOT_DIR/.cache/go-build" \
GOMODCACHE="$ROOT_DIR/.cache/gomod" \
GOPATH="$ROOT_DIR/.cache/gopath" \
GOLANGCI_LINT_CACHE="$ROOT_DIR/.cache/golangci-lint" \
go test ./...
```

Cache deletion is not recommended in normal workflows because it resets warm caches.

## Cleanup Tip

If cache cleanup is unavoidable and fails because files are read-only, normalize permissions before retrying cleanup:

```bash
chmod -R u+w .cache
```
