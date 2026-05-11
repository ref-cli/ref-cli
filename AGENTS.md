# ref-cli

GitHub: https://github.com/ref-cli/ref-cli

The `ref` binary — written in Go. Single static binary, cross-compiles for macOS and Linux.

Full design spec: [`docs/spec.md`](docs/spec.md)

## Build & Test

```bash
make build        # build for current platform → bin/ref
make install      # install to $(GOPATH)/bin
make test         # go test ./...
make lint         # golangci-lint run
make build-all    # cross-compile: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
make release      # build all platform tarballs into dist/
make bundle-examples  # fetch latest ref-examples tag into bundled-examples/
```

## Key Architecture Notes

- **Bundled examples**: `ref-examples` is embedded at build time via `go:embed bundled-examples/`. Run `make bundle-examples` before building a release to refresh the snapshot. `ref init` extracts these to `~/.config/ref/examples/` with no network call.
- **Dev mode**: when `../ref-examples/` exists relative to the repo root (i.e. `~/ref-cli/ref-examples/`), `ref` loads examples from there directly. Force it with `REF_EXAMPLES_DIR=../ref-examples ref <cmd>`.
- **Update check**: on every invocation, reads `last_update_check` from `conf.yml` — no network if fewer than 7 days have passed. If stale, calls the GitHub releases API synchronously with a 3-second timeout. Silent on failure.
- **AI fallback**: shells out to the user's installed `claude` or `codex` CLI via `os/exec`. No API key. Preference stored in `~/.config/ref/conf.yml`.
- **No `ref "use case"` syntax**: content search is only `ref -s <phrase>` (non-interactive). Multi-word arguments without `-s` are not treated as searches.

## Pre-commit Hooks (lefthook)

```bash
brew install lefthook
lefthook install   # run from repo root
```

Hooks: `gofmt`, `go vet`, `golangci-lint`, `go build`, `go test`, plus `example-lint` for any staged `bundled-examples/*.txt`.
