# Contributing

This repository is a Go CLI project. For day-to-day development, the easiest setup is VS Code plus the Go toolchain and the repo's standard command-line tools.

## Prerequisites

Install these tools on macOS:

```bash
brew install go lefthook golangci-lint make
```

If `make` is already available through Xcode Command Line Tools, you can skip the Homebrew install.

If you do not already have the compiler toolchain for macOS installed, run:

```bash
xcode-select --install
```

## VS Code Setup

1. Open `ref-cli/` in VS Code.
2. If you are also editing examples, add `ref-examples/` as a second folder in the same workspace.
3. Install the Go extension in VS Code.
4. Install the EditorConfig extension in VS Code.
5. Use the integrated terminal for all repo commands.

The Go extension is enough for editing, formatting, test discovery, and debugging Go code in this repo.

## Recommended VS Code Workflow

Use a terminal rooted at the repo and run the standard project commands:

```bash
lefthook install
make test
make lint
```

If you are developing against the sibling examples repo, the binary can read from it directly during development:

```bash
REF_EXAMPLES_DIR=../ref-examples go run ./cmd/ref
```

That lets you edit example files in `ref-examples/` and immediately test changes without reinstalling bundled examples. `REF_EXAMPLES_DIR` can point to the repo root or directly to `examples/`.

## What to work on

Most changes fall into one of these buckets:

- Example content in `../ref-examples/`
- CLI behavior in `cmd/ref/`
- Parsing, loading, or rendering logic in `internal/`
- Documentation in `docs/` or `README.md`

## Editing examples

Example files are plain text files with optional YAML frontmatter. A typical file
looks like this:

```text
---
tags: [ compression ]
---
# To create a gzip archive:
tar -czvf archive.tar.gz /path/to/dir/
```

Keep the comment line short and make the command line immediately follow it.

## Useful Repo Commands

```bash
make build
make test
make lint
make build-all
make release
make bundle-examples
```

## Notes

- Keep Go files formatted with `gofmt`.
- Run `lefthook install` once per clone so the pre-commit hooks are active.
- If you only need a quick local run, use `go run ./cmd/ref` from the repo root.
- If you are new to Go, read [docs/go-primer.md](docs/go-primer.md) first.

## Before opening a PR

- Run `make test`
- Run `make lint`
- Check that `gofmt` has been applied to any Go files you touched

## Releasing

Releases are triggered by pushing a version tag. The GitHub Actions release workflow handles everything else.

**Prerequisites (one-time setup)**

Add a `TAP_TOKEN` secret to the `ref-cli/ref-cli` repository (Settings → Secrets → Actions). It must be a GitHub personal access token with `repo` write access to `ref-cli/homebrew-tap` — the workflow uses it to auto-update the Homebrew formula.

**Steps**

```bash
# 1. Verify everything passes locally
make test
make lint

# 2. Tag and push — this triggers the release workflow
git tag v0.2.0
git push origin v0.2.0
```

**What the workflow does**

1. Runs the test suite
2. Fetches the latest `ref-examples` tag and bundles it into the binary (`make bundle-examples`)
3. Cross-compiles for `darwin/arm64`, `darwin/amd64`, `linux/arm64`, `linux/amd64`
4. Creates a GitHub release with tarballs and `checksums.txt` attached
5. Updates `Formula/ref.rb` in `ref-cli/homebrew-tap` with the new version and SHA256s

The workflow file is at [`.github/workflows/release.yml`](.github/workflows/release.yml).
