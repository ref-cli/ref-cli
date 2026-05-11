# `ref` Implementation Plan

This document describes the architecture and implementation plan for the `ref` CLI tool — a quick command example lookup tool with AI fallback.

## Overview

`ref` is a standalone Go CLI tool that helps developers quickly find practical examples for CLI commands. It supports:
- Interactive TUI with fuzzy search
- Use-case search (search by what you want to do, not just command name)
- AI-powered example generation (via Claude Code or OpenAI Codex CLI)
- User-managed examples with contribution workflow back to the public repo

## Architecture

### Directory Structure

**`ref-cli/ref-cli`** — the binary (`github.com/ref-cli/ref-cli`):
```
ref/
├── cmd/ref/
│   ├── main.go           # entry point, cobra root command + auto-init
│   ├── cmd_view.go       # `ref <command>` — display example
│   ├── cmd_search.go     # `ref "phrase"` — content search
│   ├── cmd_tui.go        # TUI mode (bare `ref`)
│   ├── cmd_init.go       # `ref init` — setup ~/.config/ref/
│   ├── cmd_update.go     # `ref update` — re-download examples
│   ├── cmd_path.go       # `ref path` — print examples directory
│   ├── cmd_open.go       # `ref open [cmd]` — open dir/file
│   └── cmd_contribute.go # `ref contribute [cmd]` — contribution instructions
├── internal/
│   ├── example/
│   │   └── example.go     # parse .txt files: YAML frontmatter + plain text
│   ├── examples/
│   │   └── examples.go    # load from ~/.config/ref/examples/
│   ├── ai/
│   │   └── ai.go         # detect/invoke claude or codex CLI
│   ├── tui/
│   │   ├── model.go      # bubbletea model: list + preview
│   │   └── keys.go       # keybindings
│   └── download/
│       └── download.go   # GitHub zip download + extraction (from ref-examples)
├── go.mod                # module: github.com/ref-cli/ref-cli
├── go.sum
├── Makefile              # build, install targets
├── lefthook.yml          # pre-commit hooks (Go only)
├── .editorconfig         # formatting rules for *.go
└── README.md             # user-facing docs
```

**`ref-cli/ref-examples`** — examples only (no Go required to contribute):
```
ref-examples/
├── tar.txt
├── git.txt
├── docker.txt
├── ...                   # one .txt file per command
├── lefthook.yml          # pre-commit: example-lint only
└── .editorconfig         # formatting rules for *.txt
```

## Core Components

### 1. Example File Format (`.txt` files)

Files stored at:
- Repo defaults: `tar.txt` (in `ref-cli/ref-examples`)
- User's copy: `~/.config/ref/examples/tar.txt` (on user's machine)
- **Dev override**: `../ref-examples/*.txt` — when the sibling repo exists at `~/ref-cli/ref-examples/`, `ref` loads examples from there directly, skipping `~/.config/ref/examples/`. Also overridable via `REF_EXAMPLES_DIR`.

Format: Optional YAML frontmatter + plain text entries:

```
---
tags: [ compression, archiving ]
---
# To extract an uncompressed archive:
tar -xvf /path/to/foo.tar

# To create a gzipped archive:
tar -czvf /path/to/foo.tgz /path/to/foo/
```

**Rules:**
- One blank line between entries
- No trailing whitespace
- LF line endings, UTF-8, final newline
- Comments must start with `# ` (space after hash)

### 2. CLI Commands

| Command | Purpose |
|---|---|
| `ref` | Auto-init if needed, show brief help, open TUI |
| `ref <cmd>` | Display example for a command |
| `ref "phrase"` | Use-case search across all examples |
| `ref -l` | List all available commands |
| `ref --version` | Print binary version |
| `ref -s <phrase>` | Explicit content search |
| `ref <cmd> --ai` | Force AI generation |
| `ref <cmd> --ai=codex` | Override AI backend |
| `ref <cmd> --save` | Save AI-generated example |
| `ref init` | Manually re-run initialization |
| `ref update` | Re-download examples (skips local modifications) |
| `ref path` | Print examples directory path |
| `ref open [cmd]` | Open directory in Finder or specific file in editor |
| `ref contribute [cmd]` | Print contribution instructions |

### 3. Initialization Flow

**First run** (`ref` without `~/.config/ref/conf.yml`):
1. Create `~/.config/ref/examples/`
2. Download latest tagged release from `ref-examples`: `https://github.com/ref-cli/ref-examples/archive/refs/tags/<latest-tag>.zip`
3. Extract all `*.txt` files → `~/.config/ref/examples/`
4. Write `~/.config/ref/conf.yml` with defaults

**`ref init`** (manual re-run):
- Same as above, no-overwrite mode (preserves user edits)

**`ref update`**:
- Re-downloads zip
- Compares checksums in `~/.config/ref/.checksums`
- Overwrites only unmodified files
- Reports: "Updated N, skipped M (locally modified)"

### 4. TUI (bubbletea)

Layout:
```
> search: [input field with fuzzy results]
──────────────────────────────────────────
[command list]  |  [content preview]
──────────────────────────────────────────
[↑↓] navigate  [y] copy  [e] edit  [q] quit
```

Features:
- Fuzzy search on command names + content
- Live preview pane
- Copy to clipboard
- Open in editor
- Fullscreen view

### 5. AI Fallback

When no local example exists:

1. Detect installed AI CLI: `which claude` or `which codex`
2. If none: print install message
3. If one: use it automatically, save preference
4. If both: prompt user to choose
5. Shell out: `claude -p "Generate a ref example for '<cmd>'..."`
6. Display with `[AI: claude]` header
7. Prompt to save to `~/.config/ref/examples/<cmd>.txt`

Prompt template:
```
Generate a ref example file for the '<command>' command.
Format: plain text. Each entry is a comment line starting with '# ' 
describing the use case, followed by the exact shell command.
Include 8-12 practical examples covering common developer workflows.
Output only the file content, no markdown fences, no extra explanation.
```

### 6. Contribution Workflow

`ref contribute <cmd>` prints two sections:

**For AI Agents:**
```
Paste this prompt to your AI agent:
  Fork https://github.com/ref-cli/ref and clone locally.
  Copy the file below into examples/<cmd>.txt...
  [shows file content]
```

**For Humans:**
```
1. Fork https://github.com/ref-cli/ref
2. git clone git@github.com:<user>/ref.git
3. cp ~/.config/ref/examples/<cmd>.txt examples/<cmd>.txt
4. git add examples/<cmd>.txt && git commit -m "examples: add <cmd>"
5. git push origin main  →  open PR
```

## Development Setup

### Example Lookup During Development

When both repos are cloned side-by-side under `~/ref-cli/`:

```
~/ref-cli/
  ref-cli/        ← you're here
  ref-examples/   ← sibling repo
```

`ref` automatically detects `../ref-examples/` and uses it as the examples source, so you can edit example files and test them without running `ref init` or publishing a release. To force a specific directory:

```bash
REF_EXAMPLES_DIR=../ref-examples go run ./cmd/ref tar
```

### Dependencies

```bash
go get github.com/spf13/cobra
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/glamour
go get github.com/charmbracelet/lipgloss
go get github.com/sahilm/fuzzy
```

### Building

```bash
go build -o /usr/local/bin/ref ./cmd/ref
```

### Pre-commit Hooks

```bash
brew install lefthook
lefthook install
```

Hooks check:
- Go formatting (gofmt)
- Go vet
- Build succeeds
- Tests pass
- Example files follow format rules

## Testing Strategy

- Unit tests for `example/` parsing
- Unit tests for `ai/` CLI detection
- Integration tests for `init`, `update`, `download`
- Manual TUI testing with bubbletea

## Phase 1: Core (MVP)

1. Project scaffolding + go.mod
2. `cmd/ref/main.go` with cobra
3. `example/` parsing
4. `examples/` loading from disk
5. `cmd_view.go` — display example
6. Default 5-10 example files
7. Basic tests

## Phase 2: TUI & Search

1. `cmd_tui.go` with bubbletea
2. `cmd_search.go` — use-case search
3. Fuzzy matching
4. Copy-to-clipboard
5. E2E manual testing

## Phase 3: Initialization & Distribution

1. `cmd_init.go` — download from GitHub
2. Zip extraction logic
3. Config file management
4. `cmd_update.go` — checksum-based updates

## Phase 4: AI & Contribution

1. `ai/` — detect claude/codex
2. `cmd_contribute.go` — instructions
3. Preference storage
4. Save workflow

## Phase 5: Polish

1. `cmd_path.go`, `cmd_open.go`
2. Error handling
3. Documentation (README)
4. Homebrew formula
5. Shell completion scripts (bash/zsh/fish)

## Key Design Decisions

1. **No embedding** — examples downloaded on init, stored on disk for easy user editing
2. **Single source** — no "public vs personal" distinction; users own all their examples
3. **No API key** — AI is accessed via existing installed CLIs (claude, codex), not direct API calls
4. **Contribute back** — examples can be contributed to the public repo via GitHub PRs
5. **Cross-platform** — Go, works on macOS and Linux; uses `open` (macOS) or `xdg-open` (Linux)

## Success Criteria

- ✓ Users can run `ref` and get TUI
- ✓ Users can search by use case ("compress a folder")
- ✓ Users can generate examples with AI
- ✓ Users can contribute examples back
- ✓ Examples persist and are editable
- ✓ Startup under 100ms (after first init)
- ✓ Works on macOS and Linux
