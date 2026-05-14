# ref

Quick command examples in your terminal.

```
ref tar              # show examples for the tar command
ref -s "compress"    # search examples by use case
ref                  # open interactive TUI
```

## Install

**Homebrew (macOS / Linux)**

```bash
brew install ref-cli/tap/ref
```

**Manual**

Download the latest binary from the [releases page](https://github.com/ref-cli/ref-cli/releases),
extract it, and move it somewhere on your `$PATH`.

**From source**

```bash
git clone https://github.com/ref-cli/ref-cli.git
cd ref-cli
make install          # installs to $(GOPATH)/bin
```

## Usage

| Command                    | Description                                                   |
| -------------------------- | ------------------------------------------------------------- |
| `ref`                      | Open interactive TUI (fuzzy search + preview)                 |
| `ref <command>`            | Show examples for a command                                   |
| `ref -s <phrase>`          | Search examples by use case                                   |
| `ref -l`                   | List all commands with examples                               |
| `ref <command> --ai`       | Force AI generation for a command                             |
| `ref init`                 | Set up `~/.config/ref/` (run automatically on first use)      |
| `ref examples`             | Show bundled examples version                                 |
| `ref examples sync`        | Download the latest examples from the registry                |
| `ref path`                 | Print the examples directory path                             |
| `ref open [command]`       | Open the examples directory (or a specific file) in `$EDITOR` |
| `ref contribute [command]` | Print instructions for contributing an example                |

### TUI key bindings

The TUI has two modes: **input mode** (default — type to search) and **command mode** (navigate and act on results). Press `esc` to toggle between modes.

**Both modes**

| Key            | Action                                      |
| -------------- | ------------------------------------------- |
| `↑` / `ctrl+p` | Navigate list up (left pane) or previous entry (right pane) |
| `↓` / `ctrl+n` | Navigate list down (left pane) or next entry (right pane)   |
| `←` / `→`      | Switch active pane                          |
| `ctrl+c`       | Quit                                        |

**Command mode only**

| Key  | Action                             |
| ---- | ---------------------------------- |
| `esc` | Toggle to input mode              |
| `y`  | Copy selected command to clipboard |
| `e`  | Edit example file in `$EDITOR`     |
| `↵`  | Toggle fullscreen preview          |

**Input mode only**

| Key   | Action                  |
| ----- | ----------------------- |
| `esc` | Toggle to command mode  |

**Fullscreen**

| Key       | Action                             |
| --------- | ---------------------------------- |
| `↑` / `↓` | Previous / next entry              |
| `y`       | Copy selected command to clipboard |
| `esc`     | Back to normal view                |
| `ctrl+c`  | Quit                               |

### AI generation

When no example exists for a command, `ref` can generate one using your
locally installed AI CLI:

```bash
ref mycli                         # auto-detect claude or codex if no example exists
ref mycli --ai                    # force AI generation even if example exists
ref mycli --save                  # generate and save permanently
ref mycli --ai-backend codex      # override backend for this invocation
```

Supported backends: [`claude`](https://docs.anthropic.com/en/docs/claude-code)
(Claude Code) and [`codex`](https://github.com/openai/codex) (OpenAI Codex
CLI). No API key is required by `ref` — the backend CLIs handle authentication.
Your preference is saved to `~/.config/ref/conf.yml` after first use.

## Examples format

Each example file is a plain `.txt` file with an optional YAML frontmatter block:

```
---
tags: [ compression, archiving ]
variants: [ bsd, gnu ]
---
# To extract an archive:
tar -xvf /path/to/foo.tar

# [GNU] To use Perl regex:
grep -P '\d+' file
```

- One `# comment` line followed by one command line makes an entry.
- `[TAG]` at the start of a comment line adds an inline tag (e.g. `[GNU]`, `[BSD]`).
- Frontmatter keys: `tags`, `variants`, `versions`, `min_version`.

User examples live at `~/.config/ref/examples/` — edit them freely.

## Configuration

`~/.config/ref/conf.yml` is created automatically on first run:

```yaml
examples_version: v0.3.0
last_update_check: "2026-05-11T10:00:00Z"
install_method: homebrew
ai_backend: claude
```

## Documentation

| Document                                     | Description                                                                 |
| -------------------------------------------- | --------------------------------------------------------------------------- |
| [docs/go-primer.md](docs/go-primer.md)       | Go language features used in this codebase, explained for non-Go developers |
| [docs/architecture.md](docs/architecture.md) | Architecture overview and implementation notes                              |
| [docs/spec.md](docs/spec.md)                 | Full design specification                                                   |
| [CONTRIBUTING.md](CONTRIBUTING.md)           | How to build, test, and contribute                                          |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup instructions. In short:

```bash
brew install go lefthook golangci-lint make
lefthook install
make test
make lint
```

New to Go? Read [docs/go-primer.md](docs/go-primer.md) first — it explains every
Go feature used in this repo with real examples from the source code.

## License

[MIT](LICENSE)
