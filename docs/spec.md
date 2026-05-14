# Plan: `ref` — Interactive CLI Cheat Sheet Tool

## Context

A new standalone Go CLI tool called `ref`. The CLI and examples live in **separate repos**: `ref-cli/ref-cli` (the binary) and `ref-cli/ref-examples` (the example `.txt` files). Each `ref` release **bundles the latest `ref-examples`** directly in the release tarball — no network call needed on first run. Users can sync to a newer examples version at any time via `ref examples sync`. Users own all their examples. Three differentiators over existing tools:

1. **Interactive TUI** — fuzzy-searchable list + preview pane, copy-to-clipboard, edit in `$EDITOR`
2. **Content search** — search by what you want to DO (`ref -s "compress a folder"`), not just command name
3. **AI-powered fallback** — shells out to installed `claude` or `codex` CLI when no example exists

`ref` is safe to use as the binary name (researched 2026-05-10): not a builtin in bash/zsh/fish, no Homebrew formula, no toolchain (git, nvm, rbenv, asdf, pyenv) installs a `ref` binary, and the only npm package named `ref` is an unmaintained C buffer library with no CLI. Short single-word names (`tldr`, `cheat`, `navi`, `fzf`) are standard practice for developer tools.

---

## Project Location

Three repos:
- `~/ref-cli/ref-cli/` — CLI binary (`github.com/ref-cli/ref-cli`)
- `~/ref-cli/ref-examples/` — example `.txt` files (`github.com/ref-cli/ref-examples`)
- `~/ref-cli/homebrew-tap/` — Homebrew formula (`github.com/ref-cli/homebrew-tap`)

---

## Language & Key Libraries

**Go** — single static binary, trivial cross-compile for macOS + Linux.

| Library | Version | Purpose | Notes |
|---|---|---|---|
| `github.com/spf13/cobra` | v1.10.2 | CLI argument parsing | Stable v1; no breaking changes. Used by kubectl, gh, Hugo. |
| `github.com/charmbracelet/bubbletea` | v1.3.10 | TUI framework | Pin to v1. v2 beta exists but API unstable — do not mix v1/v2. |
| `github.com/charmbracelet/bubbles` | v1.0.0 | list, textinput, viewport components | Stay on v1 to match bubbletea v1. v2 beta tracks bubbletea v2. |
| `github.com/charmbracelet/lipgloss` | v1.1.0 | Terminal styling | Pin to v1. v2 beta has breaking API changes (renderer/color model). |
| `github.com/sahilm/fuzzy` | v0.1.2 | Fuzzy matching | Pre-v1 but API is practically frozen; no v1 roadmap. |
| `os/exec` (stdlib) | — | Shell out to `claude` or `codex` CLI | |
| `net/http` + `archive/zip` (stdlib) | — | Download and extract examples zip from GitHub | |

> **Note (researched 2026-05-10):** bubbletea, bubbles, and lipgloss each have parallel v1 stable / v2 beta tracks. Do not mix v1 and v2 of these packages in one module — the import paths differ (e.g. `.../bubbletea/v2`).

---

## Directory Structure

**`ref-cli/ref-cli`** — the binary:
```
~/ref-cli/ref-cli/
  cmd/ref/
    main.go              # entry point, cobra root command + auto-init on first run
    cmd_view.go          # `ref <command>` — show example
    cmd_search.go        # `ref -s <phrase>` — content search (non-interactive)
    cmd_tui.go           # TUI launched by bare `ref`
    cmd_init.go          # `ref init` — extract bundled examples to ~/.config/ref/
    cmd_contribute.go    # `ref contribute [command]` — print contribution instructions
    cmd_examples.go      # `ref examples` — show bundled version; `ref examples sync`
    cmd_path.go          # `ref path` — print examples dir path (scriptable)
    cmd_open.go          # `ref open [command]` — open dir in Finder/xdg-open or file in $EDITOR
  internal/
    example/
      example.go         # parse .txt files: YAML frontmatter (tags, variants, versions, min_version) + plain text; extract inline [TAG] annotations from comment lines
    examples/
      examples.go        # load all .txt files from ~/.config/ref/examples/
    ai/
      ai.go              # detect/invoke claude or codex CLI; handle preference
    tui/
      model.go           # bubbletea model: list + preview pane
      keys.go            # keybindings
    bundled/
      bundled.go         # embed bundled examples via go:embed; expose version string
      examples/          # snapshot of ref-examples fetched at release time by bundle-examples.sh
        tar.txt
        git.txt
        ...
  conf.yml.template      # default config written to ~/.config/ref/conf.yml on init
  go.mod                 # module: github.com/ref-cli/ref-cli
  go.sum
  Makefile               # build, cross-compile, release tarball targets
  lefthook.yml           # pre-commit hooks for Go linting only
  .editorconfig          # formatting rules for *.go
  README.md              # user-facing install and usage docs
  LICENSE                # MIT
  .github/
    workflows/
      ci.yml             # lint + test on push/PR
      release.yml        # build multi-platform binaries + tarballs on tag; update homebrew-tap
```

**`ref-cli/ref-examples`** — the examples (separate repo, no Go required to contribute):
```
~/ref-cli/ref-examples/
  tar.txt
  git.txt
  docker.txt
  curl.txt
  ...               # one .txt file per command
  VERSION           # plain-text version string, e.g. "v0.1.0" — bumped with each release tag
  .editorconfig     # formatting rules for *.txt
  lefthook.yml      # pre-commit: example-lint only
  LICENSE           # MIT
  .github/
    workflows/
      lint.yml      # example-lint on push/PR
      release.yml   # create GitHub release + zip on tag
```

**`ref-cli/homebrew-tap`** — Homebrew formula:
```
~/ref-cli/homebrew-tap/
  Formula/
    ref.rb          # Homebrew formula (auto-updated by ref-cli release workflow)
  README.md
  LICENSE
```

---

## Example File Format

Files use the `.txt` extension, named after the command: `tar.txt`, `git.txt`.

- **Repo defaults**: `~/ref-cli/ref-examples/tar.txt` (source of truth, committed to `ref-cli/ref-examples`)
- **Bundled copy**: embedded in the `ref` binary at build time via `go:embed bundled-examples/`
- **User's copy**: `~/.config/ref/examples/tar.txt` (extracted from bundle on init, user-editable)

File contents — optional YAML frontmatter, then plain text entries:

```
---
tags: [ compression, archiving ]
variants: [ bsd, gnu ]
---
# To extract an uncompressed archive:
tar -xvf /path/to/foo.tar

# [GNU] To extract and show progress (GNU tar only):
tar -xvf /path/to/foo.tar --checkpoint=100

# [BSD] To list archive contents (macOS default tar):
tar -tvf /path/to/foo.tar
```

**Rules (enforced by EditorConfig + pre-commit):**
- One blank line between entries (comment + command pairs)
- No trailing whitespace
- LF line endings, UTF-8, final newline
- Comment lines must start with `# ` (space after hash)
- Variant/version tags are optional and placed immediately after `# `: `# [GNU]`, `# [BSD]`, `# [v2]`, `# [macOS]`, `# [Linux]`
- Command line follows immediately on the next line after its comment

---

## Variants and Versions

### BSD vs GNU commands

Many standard Unix commands (`grep`, `sed`, `tar`, `find`, `date`, `xargs`) differ between BSD (macOS default) and GNU (Linux default, also installable on macOS via Homebrew). These are tracked in a **single file** using inline `[BSD]` / `[GNU]` tags in the comment line.

Example — `grep.txt`:
```
---
tags: [ search, text ]
variants: [ bsd, gnu ]
---
# To search recursively for a pattern:
grep -r "pattern" .

# [GNU] To search with Perl-compatible regex:
grep -P '\d{3}-\d{4}' file.txt

# [BSD] To search with extended regex (macOS grep):
grep -E '\d{3}-\d{4}' file.txt

# [GNU] To search and show N lines of context:
grep -C 3 "pattern" file.txt
```

Entries with no tag apply to both variants. The TUI displays tags inline; future filtering by `[GNU]` / `[BSD]` is a planned enhancement.

**Frontmatter field**: `variants: [ bsd, gnu ]` — declares which variants the file covers. Used by tooling and future search filtering.

### CLI version differences

For tools with breaking changes across major versions (`aws`, `kubectl`, `terraform`, `pip`, etc.), entries are annotated with `[v1]`, `[v2]`, etc.

Example — `aws.txt`:
```
---
tags: [ cloud, aws ]
versions: [ "1", "2" ]
min_version: "2"
---
# To list S3 buckets:
aws s3 ls

# [v2] To configure SSO login (v2 only):
aws sso login --profile my-profile

# [v1] To configure credentials (v1 style, deprecated in v2):
aws configure
```

**Frontmatter fields**:
- `versions: ["1", "2"]` — which major versions are covered in this file
- `min_version: "2"` — if the entire file only applies from a certain version onwards

Entries with no version tag apply across all documented versions. Version tags are informational — `ref` does not auto-detect the installed CLI version.

---

## Example Source & Lookup

**Production**: `~/.config/ref/examples/` on disk. All `.txt` files in this directory are loaded.

**Development**: when the sibling directory `../ref-examples/` exists relative to the repo root (i.e. `~/ref-cli/ref-examples/`), `ref` uses it directly as the examples source. Can also be forced with `REF_EXAMPLES_DIR=../ref-examples ref <cmd>`.

### Bundled Examples

Each `ref` release embeds a snapshot of `ref-examples` at build time using `go:embed all:examples` (from `internal/bundled/bundled.go`). The CI release workflow runs `scripts/bundle-examples.sh`, which downloads the latest tagged `ref-examples` zip, extracts it into `internal/bundled/examples/`, and stages the result. The bundled version string is read from `internal/bundled/examples/VERSION` and exposed as `internal/bundled.ExamplesVersion`.

`ref init` (and auto-init on first run):
1. Creates `~/.config/ref/examples/`
2. Extracts bundled examples (from `go:embed`) — **no network call**
3. Writes files with no-overwrite (`os.O_CREATE|os.O_EXCL`) — preserves user edits
4. Writes default `~/.config/ref/conf.yml` including `examples_version: <bundled version>`

### Syncing Examples

`ref examples sync` (formerly `ref update`) downloads newer examples from GitHub:
- Fetches latest tag from GitHub API: `https://api.github.com/repos/ref-cli/ref-examples/releases/latest`
- Downloads zip: `https://github.com/ref-cli/ref-examples/archive/refs/tags/<latest-tag>.zip`
- Compares checksums in `~/.config/ref/.checksums` to skip locally modified files
- `.checksums` format: `<sha256hex>  <filename>` (one per line, same as `shasum -a 256`)
- Updates `examples_version` in `~/.config/ref/conf.yml`
- Prints: "Updated 3 examples (v1.1.0 → v1.2.0), skipped 2 (locally modified)"

Specific version: `ref examples sync --version=v1.2.0`

**Timeouts**: all network calls use a 10-second timeout via `http.Client{Timeout: 10 * time.Second}`. On timeout or any network error during sync, print a warning and exit non-zero:
```
ref: examples sync failed: request timed out (10s)
Your examples are unchanged. Try again later or check your connection.
```

No local clone of either repo required — just the binary.

---

## CLI Interface

```
ref                              # auto-init if needed, print brief help, open TUI
ref --version                    # print binary version (e.g. v1.2.0)
ref tar                          # view example for 'tar' directly
ref -l                           # list all commands with examples
ref -s <phrase>                  # content search across all examples (non-interactive)
ref tar --ai                     # force AI generation even if example exists
ref tar --ai=codex               # override AI backend for this invocation
ref tar --save                   # save AI-generated example to ~/.config/ref/examples/
ref init                         # manually re-run init (safe: no-overwrite, uses bundled examples)
ref examples                     # show bundled examples version and installed version
ref examples sync                # download latest ref-examples from GitHub, skip local edits
ref examples sync --version=v1.2.0  # download a specific ref-examples version
ref path                         # print path to ~/.config/ref/examples/
ref open                         # open examples directory in Finder (macOS) or xdg-open (Linux)
ref open tar                     # open specific example file in $EDITOR
ref contribute <command>         # print contribution instructions for a specific example
ref contribute                   # print general contribution guide (fork URL, format rules, PR checklist)
```

---

## Example CLI Usage

```
# First run — auto-inits from bundled examples (no network call) and opens TUI
$ ref
Initializing ref with bundled examples (v0.1.0)... done (19 examples)
ref — quick command examples  [↑↓] scroll  [←→] switch  [esc] command mode  [ctrl+c] quit
> _

# Look up a specific command
$ ref tar
# To extract an uncompressed archive:
tar -xvf /path/to/foo.tar

# To extract into a specific directory:
tar -xvf /path/to/foo.tar -C /path/to/dest/

# To create a gzipped archive:
tar -czvf /path/to/foo.tgz /path/to/foo/

# Search by use case (not command name)
$ ref -s "compress a folder"
[tar] # To create a gzipped archive:
tar -czvf /path/to/foo.tgz /path/to/foo/

[zip] # To compress a folder:
zip -r archive.zip /path/to/folder/

# List all available commands
$ ref -l
awk       base64    brew      chmod     curl
docker    find      git       grep      jq
kill      npm       openssl   ps        python
rsync     sed       ssh       tar

# AI generation when no example exists
$ ref kubectl
No example found for 'kubectl'.
Using claude... done

[AI: claude]
# To get all pods in a namespace:
kubectl get pods -n <namespace>

# To apply a manifest:
kubectl apply -f <file.yaml>
...
Save to examples? [y/N]: y
Saved to ~/.config/ref/examples/kubectl.txt

# Get the examples directory path
$ ref path
/Users/alo/.config/ref/examples

# Use in a shell pipeline
$ ls $(ref path)
awk.txt  base64.txt  brew.txt  chmod.txt  curl.txt  docker.txt ...

# Open examples directory in Finder
$ ref open

# Open a specific example in $EDITOR
$ ref open tar

# Show examples version info (no network call)
$ ref examples
Bundled:   v0.1.0
Installed: v0.1.0

# Sync to latest examples from GitHub (preserves local edits)
$ ref examples sync
Updated 3 examples (v0.1.0 → v0.2.0): tar, git, docker
Skipped 2 (locally modified): curl, ssh

# Sync to a specific version
$ ref examples sync --version=v0.1.0
Downgraded to v0.1.0 (3 files updated, 2 skipped)

# Contribute your kubectl example back
$ ref contribute kubectl
━━━ Contributing 'kubectl' to ref examples ━━━

Your file: ~/.config/ref/examples/kubectl.txt
──────────────────────────────────────────────
# To get all pods in a namespace:
kubectl get pods -n <namespace>
...
──────────────────────────────────────────────

── For AI Agents (Claude Code / Codex) ────────
Paste the following as a prompt to your AI agent:
  Fork https://github.com/ref-cli/ref-examples and clone it locally.
  Copy the file below into kubectl.txt ...

── For Humans ──────────────────────────────────
1. Fork https://github.com/ref-cli/ref-examples on GitHub
2. git clone git@github.com:<your-username>/ref-examples.git
3. cp ~/.config/ref/examples/kubectl.txt kubectl.txt
4. git add kubectl.txt && git commit -m "examples: add kubectl"
5. git push origin main  →  open PR at github.com/ref-cli/ref-examples
```

---

## Startup Sequence (bare `ref`)

1. Check if `~/.config/ref/conf.yml` exists — if not, auto-run init (extract bundled examples)
2. Display any pending update notifications stored from the previous background check; clear them from conf.yml
3. If an examples update notification is pending, prompt `[y/n]` and wait for response before opening TUI
4. If `last_update_check` is more than 7 days ago, spawn a background goroutine to check GitHub — no blocking on the hot path
5. Enter interactive TUI

---

## Update Check

On every invocation of `ref` (any subcommand):
1. Display any pending update notifications (`pending_binary_update`, `pending_examples_update`) from the previous background check; clear them from conf.yml.
2. Read `last_update_check` from `~/.config/ref/conf.yml` — **no network call** if fewer than 7 days have passed; skip the rest.
3. If 7 days have passed, spawn a background goroutine (non-blocking) that calls both GitHub release APIs with a **3-second timeout** each.
4. If the goroutine fails (no network, DNS error, rate-limit, etc.), do not update `last_update_check` — the check retries on the next invocation after the failure.
5. If the goroutine succeeds, save results as `pending_binary_update` / `pending_examples_update` in conf.yml and update `last_update_check`. Notifications are displayed on the **next** invocation.

### ref-cli binary out of date

If a newer `ref-cli` release exists on GitHub:

```
A new version of ref is available: v1.2.0 (you have v1.1.0)
  Homebrew: brew upgrade ref
  Manual:   https://github.com/ref-cli/ref-cli/releases/latest
```

Printed once per session to stderr; does not block. No auto-upgrade.

### ref-examples out of date

If a newer `ref-examples` release exists on GitHub and the user's installed examples version (from `conf.yml`) is behind:

```
New examples available: v0.3.0 (you have v0.1.0)
Run 'ref examples sync' to update. [y to sync now, n to skip]
```

- If the user presses `y`: runs `ref examples sync` inline before continuing
- If the user presses `n` or hits Enter: continues; re-prompts next time the check fires
- Never prompts more than once per session
- For Homebrew users the message reads: "Run 'brew upgrade ref' to get the latest binary and examples."

### `conf.yml` tracking fields

```yaml
last_update_check: "2026-05-03T10:00:00Z"   # RFC3339, updated on successful background check
examples_version: "v0.1.0"                   # version of currently installed examples
install_method: "homebrew"                    # "homebrew" | "manual" | "source" — set on init
pending_binary_update: "v1.2.0"              # set by background goroutine, cleared on display
pending_examples_update: "v0.3.0"            # set by background goroutine, cleared on display
```

`install_method` is detected on first init (check if `ref` binary path is under Homebrew's prefix) and used to tailor upgrade instructions.

---

## TUI Layout

```
┌─────────────────────────────────────────────────────────────┐
│ > search: compress_____                                     │
├─ command ──────────────┬─ tar ───────────────────────────── │
│ tar          │ # To extract an uncompressed archive:        │
│ gzip         │ tar -xvf /path/to/foo.tar                    │
│ zip          │                                              │
│ rsync        │ # To create a .gz archive:                   │
│ scp          │ tar -czvf /path/to/foo.tgz /path/to/foo/     │
│ ...          │                                              │
├─────────────────────────────────────────────────────────────┤
│ [↑↓] scroll  [←→] switch  [esc] command mode  [ctrl+c] quit│
└─────────────────────────────────────────────────────────────┘
```

The TUI has two modes:

- **Input mode** (default): typing filters the command list. `↑`/`↓` and `←`/`→` work for navigation; `esc` switches to command mode.
- **Command mode**: single-letter shortcuts are active. `i` returns to input mode.

The top divider labels each pane — the left always shows **command**, the right shows the currently selected command name (e.g. **tar**). The active pane label is highlighted.

**Key bindings (both modes):**

| Key | Action |
|---|---|
| `↑` / `ctrl+p` | Navigate list up (left pane) or previous entry highlight (right pane) |
| `↓` / `ctrl+n` | Navigate list down (left pane) or next entry highlight (right pane) |
| `←` / `→` | Switch active pane |
| `ctrl+c` | Quit |

**Command mode only:**

| Key | Action |
|---|---|
| `i` | Enter input mode |
| `y` | Copy selected command to clipboard |
| `e` | Open example file in `$EDITOR` |
| `↵` | Fullscreen preview |

**Fullscreen view:** title shows only the command name (e.g. `tar`). `↑`/`↓` moves between entries; `y` copies; `esc` goes back; `ctrl+c` quits.

Search ranking: exact name match → name prefix → name contains → command text prefix → command text contains → frontmatter tag match → inline tag match → comment match → fuzzy name → fuzzy full text.

---

## AI Fallback Behavior

No API key — shells out to the user's installed AI CLI.

| CLI | Detection | Invocation |
|---|---|---|
| Claude Code | `which claude` | `claude -p "<prompt>"` |
| OpenAI Codex | `which codex` | `codex "<prompt>"` — exact non-interactive/pipe mode syntax to be researched and confirmed during implementation phase |

**Preference**: stored in `~/.config/ref/conf.yml` as `ai_backend: claude` or `ai_backend: codex`.

**Flow when no example found:**
1. Check which AI CLIs are installed
2. If none: print "No example found. Install `claude` (Claude Code) or `codex` (OpenAI Codex CLI) to enable AI generation."
3. If one installed: use it, save preference to conf on first use
4. If both installed and no preference set: prompt user to choose, save preference
5. Shell out (capture stdout): `claude -p "Generate a ref example file for '<command>'..."`
6. Display with `[AI: claude]` header
7. Prompt "Save to examples? [y/N]" — if yes, write to `~/.config/ref/examples/<command>.txt`

**Prompt template:**
```
Generate a ref example file for the '<command>' command.
Format: plain text. Each entry is a comment line starting with '# ' describing the use case,
followed by the exact shell command on the next line.
Include 8-12 practical examples covering common developer workflows.
Output only the file content, no markdown fences, no extra explanation.
```

---

## `ref contribute` — Contribution Flow

`ref contribute` (no args) prints a general contribution guide to stdout: the `ref-examples` repo URL, the example file format rules, and the PR checklist. No file reads; no network calls.

`ref contribute <command>` prints contribution instructions to stdout in two sections: one for AI agents and one for humans. Reads `~/.config/ref/examples/<command>.txt` and includes its content in the output.

**Output format:**

```
━━━ Contributing 'tar' to ref examples ━━━

Your file: ~/.config/ref/examples/tar.txt
─────────────────────────────────────────────────
[file contents shown here]
─────────────────────────────────────────────────

── For AI Agents (Claude Code / Codex) ─────────

Paste the following as a prompt to your AI agent:

  Fork https://github.com/ref-cli/ref-examples and clone it locally.
  Copy the file below into tar.txt (create if it doesn't exist;
  append if it does, avoiding duplicates). Commit with message
  "examples: add/update tar". Push to your fork and open a pull request
  against ref-cli/ref-examples main with title "examples: contribute tar"
  and a brief description of what was added.

  File content to add:
  <contents of ~/.config/ref/examples/tar.txt>

── For Humans ───────────────────────────────────

1. Fork https://github.com/ref-cli/ref-examples on GitHub
2. Clone your fork:
     git clone git@github.com:<your-username>/ref-examples.git
3. Copy your example:
     cp ~/.config/ref/examples/tar.txt tar.txt
   (Or append to an existing file, removing duplicates.)
4. Commit:
     git add tar.txt
     git commit -m "examples: add/update tar"
5. Push and open a PR against ref-cli/ref-examples main:
     git push origin main
   Then visit https://github.com/ref-cli/ref-examples to open the pull request.
```

**Implementation**: `cmd_contribute.go` — reads the example file, formats both sections, prints to stdout. No network calls.

---

## `ref path` and `ref open`

**`ref path`** — prints `~/.config/ref/examples/` to stdout. Designed to be scriptable:
```bash
ls $(ref path)
cd $(ref path)
```

**`ref open`** — opens the examples directory or a specific file:
- `ref open` → opens the directory: `open ~/.config/ref/examples/` (macOS) or `xdg-open` (Linux)
- `ref open tar` → opens `~/.config/ref/examples/tar.txt` in `$EDITOR` (falls back to `open`/`xdg-open` if `$EDITOR` unset)

Cross-platform detection in `cmd_open.go`:
```go
switch runtime.GOOS {
case "darwin":
    exec.Command("open", path)
default:
    exec.Command("xdg-open", path)
}
```

---

## EditorConfig (`.editorconfig`)

```ini
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true

[*.go]
indent_style = tab
indent_size = 4

[*.txt]
indent_style = space
indent_size = 2
max_line_length = 120
```

---

## Lefthook (`lefthook.yml`)

```yaml
pre-commit:
  parallel: false
  commands:
    gofmt:
      glob: "*.go"
      run: gofmt -l {staged_files} | grep . && exit 1 || exit 0
      fail_text: "Run: gofmt -w ./..."

    govet:
      glob: "*.go"
      run: go vet ./...

    golangci-lint:
      glob: "*.go"
      run: golangci-lint run
      skip:
        - merge
        - rebase

    go-build:
      run: go build ./...
      fail_text: "Build failed — fix compilation errors before committing"

    go-test:
      run: go test ./...
      fail_text: "Tests failed"

    example-lint:
      glob: "*.txt"
      run: |
        failed=0
        for f in {staged_files}; do
          # comment lines must start with '# '
          grep -En "^#[^ ]" "$f" && echo "ERR $f: comment lines must start with '# '" && failed=1
          # inline tags must be uppercase and bracket-enclosed: [GNU], [BSD], [v2], [macOS], [Linux]
          grep -En "^# \[[a-z]" "$f" && echo "ERR $f: inline tags must be uppercase e.g. [GNU] not [gnu]" && failed=1
        done
        exit $failed
      fail_text: "Fix example file formatting issues above"
```

**Install lefthook** (contributors only — do not include in end-user install script):
```bash
brew install lefthook
cd ~/ref-cli/ref-cli && lefthook install
```

---

## Default Examples to Create

Hand-craft these under `examples/` as part of implementation:

| File | Contents focus |
|---|---|
| `tar.txt` | compress, extract, list archive contents |
| `git.txt` | clone, branch, commit, stash, rebase, log, diff |
| `docker.txt` | build, run, exec, ps, images, logs, rm/rmi |
| `curl.txt` | GET/POST, headers, auth, file download, JSON |
| `grep.txt` | basic search, recursive, regex, invert, count |
| `find.txt` | by name, by type, by date, exec, delete |
| `ssh.txt` | connect, tunnel, copy keys, config file |
| `rsync.txt` | local copy, remote sync, dry-run, exclude |
| `jq.txt` | select, filter, map, keys, pretty-print |
| `awk.txt` | print columns, filter rows, field separator |
| `sed.txt` | substitute, delete lines, in-place edit |
| `chmod.txt` | common permission patterns (755, 644, +x) |
| `ps.txt` | list processes, find by name, kill |
| `kill.txt` | by PID, by name, signals (pkill too) |
| `brew.txt` | install, update, upgrade, search, uninstall |
| `npm.txt` | init, install, run scripts, publish |
| `python.txt` | run script, venv (uv), install packages |
| `openssl.txt` | generate cert/key, inspect cert, hash |
| `base64.txt` | encode/decode string and files |

---

## Installation

### Homebrew (recommended)

```bash
brew tap ref-cli/tap
brew install ref
```

The Homebrew formula runs `ref init` automatically via `post_install`, so examples are ready immediately after install. To upgrade both the binary and bundled examples:

```bash
brew upgrade ref
```

### Manual (pre-built binary)

Download the tarball for your platform from the [latest release](https://github.com/ref-cli/ref-cli/releases/latest), extract, and move `ref` onto your `$PATH`:

```bash
# macOS arm64 example
curl -L https://github.com/ref-cli/ref-cli/releases/latest/download/ref_darwin_arm64.tar.gz | tar xz
sudo mv ref /usr/local/bin/ref
ref init
```

### Build from source

```bash
git clone git@github.com:ref-cli/ref-cli.git ~/ref-cli/ref-cli
cd ~/ref-cli/ref-cli
make install   # builds and copies ref to $(go env GOPATH)/bin
ref init
```

`ref init` extracts the bundled examples to `~/.config/ref/examples/` — no network call required. For Homebrew installs this runs automatically; manual and source installs require running it once.

---

## Makefile Targets

```makefile
VERSION    := $(shell git describe --tags --always)
LDFLAGS    := -ldflags "-X main.version=$(VERSION)"
PLATFORMS  := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

build:                             ## build for current platform
    go build $(LDFLAGS) -o bin/ref ./cmd/ref

install:                           ## install to $(GOPATH)/bin
    go install $(LDFLAGS) ./cmd/ref

build-all:                         ## cross-compile for all platforms
    $(foreach p,$(PLATFORMS), \
      GOOS=$(word 1,$(subst /, ,$(p))) \
      GOARCH=$(word 2,$(subst /, ,$(p))) \
      go build $(LDFLAGS) -o dist/ref_$(subst /,_,$(p))/ref ./cmd/ref;)

release:                           ## build all platform tarballs into dist/
    $(MAKE) build-all
    $(foreach p,$(PLATFORMS), \
      tar -czf dist/ref_$(subst /,_,$(p)).tar.gz \
        -C dist/ref_$(subst /,_,$(p)) ref LICENSE README.md;)

test:
    go test ./...

lint:
    golangci-lint run

bundle-examples:                   ## fetch latest ref-examples tag into bundled-examples/
    scripts/bundle-examples.sh
```

`scripts/bundle-examples.sh` fetches the latest `ref-examples` tag from the GitHub API, downloads the zip, extracts `*.txt` files and `VERSION` into `internal/bundled/examples/`, and stages the result. Run by the CI release workflow before building.

---

## GitHub Actions

### `ref-cli/ref-cli` — `.github/workflows/ci.yml`

Runs on every push and PR:
- `go vet ./...`
- `golangci-lint run`
- `go test ./...`
- `go build ./...` (verify compilation)
- Matrix: `ubuntu-latest`, `macos-latest`

### `ref-cli/ref-cli` — `.github/workflows/release.yml`

Triggered on tag push `v*`:
1. Run `make bundle-examples` — fetches latest `ref-examples` tag, populates `bundled-examples/`
2. Run `make release` — builds platform tarballs for `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`
3. Create GitHub release with all tarballs attached
4. Compute SHA256 for each tarball
5. Open a PR (or push directly) to `ref-cli/homebrew-tap` updating `Formula/ref.rb` with new version + SHA256 values

### `ref-cli/ref-examples` — `.github/workflows/lint.yml`

Runs on every push and PR:
- Example-lint: verifies comment lines start with `# `, no trailing whitespace, LF endings

### `ref-cli/ref-examples` — `.github/workflows/release.yml`

Triggered on tag push `v*`:
1. Update `VERSION` file with the tag
2. Create GitHub release with the repo zip attached (used by `ref examples sync`)

---

## Homebrew Tap Setup (`ref-cli/homebrew-tap`)

### Initial repo setup

```bash
cd ~/ref-cli/homebrew-tap
git init
mkdir -p Formula
```

Create `Formula/ref.rb` (template — SHA256 and version filled in by release CI):

```ruby
class Ref < Formula
  desc "Interactive CLI cheat sheet tool with fuzzy search and AI fallback"
  homepage "https://github.com/ref-cli/ref-cli"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ref-cli/ref-cli/releases/download/v#{version}/ref_darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER"
    end
    on_intel do
      url "https://github.com/ref-cli/ref-cli/releases/download/v#{version}/ref_darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ref-cli/ref-cli/releases/download/v#{version}/ref_linux_arm64.tar.gz"
      sha256 "PLACEHOLDER"
    end
    on_intel do
      url "https://github.com/ref-cli/ref-cli/releases/download/v#{version}/ref_linux_amd64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  def install
    bin.install "ref"
  end

  def post_install
    system "#{bin}/ref", "init"
  end

  test do
    system "#{bin}/ref", "--version"
  end
end
```

Push to `github.com/ref-cli/homebrew-tap`. Users install via:
```bash
brew tap ref-cli/tap
brew install ref
```

The `ref-cli` release workflow updates this formula automatically on each new tag.

---

## Verification

1. `ref` (first run, no `~/.config/ref/`) — extracts bundled examples (no network), shows help, opens TUI
2. `ref --version` — prints version string matching the release tag
3. `ref tar` — displays `~/.config/ref/examples/tar.txt`
4. `ref -s "compress a folder"` — surfaces tar, gzip, zip ranked by content match
5. `ref -l` — lists all commands with examples
6. No AI CLI installed: `ref unknowncmd` — prints install suggestion
7. `claude` installed: `ref unknowncmd` — shells out, displays `[AI: claude]` result, prompts to save
8. `ref unknowncmd --save` — writes to `~/.config/ref/examples/unknowncmd.txt`
9. `ref contribute tar` — prints AI agent prompt + human steps to stdout
10. `ref examples` — shows bundled version and installed version
11. `ref examples sync` — downloads from latest tag, reports "Updated N (vX → vY), skipped M (locally modified)"
12. `ref examples sync --version=v0.1.0` — downloads specific version
13. `ref path` — prints `~/.config/ref/examples/` (usable in `cd $(ref path)`)
14. `ref open` — opens examples dir in Finder (macOS)
15. `ref open tar` — opens `tar.txt` in `$EDITOR`
16. `lefthook run pre-commit` — all hooks pass on a clean repo
17. `make release` — produces tarballs for all 4 platform targets
18. `brew install ref-cli/tap/ref` — installs and runs `ref init` automatically via `post_install`; no manual `ref init` needed
19. `brew upgrade ref` — updates binary and bundled examples atomically
20. Update check fires after 7 days: out-of-date ref-cli prints upgrade hint; out-of-date examples prompts `[y/n]` to sync

---

## README (`ref-cli/ref-cli`)

The `README.md` is the user-facing landing page and must cover:

1. **Tagline** — one sentence: "Quick command examples in your terminal."
2. **Demo** — terminal screenshot or GIF of the TUI
3. **Install**
   - Homebrew: `brew tap ref-cli/tap && brew install ref`
   - Manual: link to releases page + per-platform `curl | tar` snippet (macOS arm64, macOS amd64, Linux amd64, Linux arm64)
   - Build from source: `git clone` → `make install`
4. **Quick start** — `ref tar`, `ref "compress a folder"`, `ref` (TUI); note that Homebrew auto-inits, manual install needs `ref init` once
5. **Key commands table** — same as CLI Interface section above
6. **AI fallback** — brief explanation, how to install claude/codex
7. **Contributing examples** — link to `ref-examples` repo, `ref contribute <cmd>` workflow
8. **License** — MIT badge + link

---

## License

Both `ref-cli/ref-cli` and `ref-cli/ref-examples` are released under the **MIT License**. Each repo must include a `LICENSE` file at its root. MIT satisfies Homebrew's OSI-approved license requirement and matches the license used by all key dependencies (cobra, bubbletea, bubbles, glamour, lipgloss).

---

## Release Checklist

### First release (one-time setup)

1. Push `ref-cli/homebrew-tap` to GitHub with the `Formula/ref.rb` template
2. Run `brew tap ref-cli/tap` locally to verify the tap is accessible

### `ref-cli/ref-examples` release

1. Tag `v0.1.0` — triggers `.github/workflows/release.yml`
2. CI updates `VERSION`, creates GitHub release with zip attached
3. Verify zip is accessible at `https://github.com/ref-cli/ref-examples/archive/refs/tags/v0.1.0.zip`

### `ref-cli/ref-cli` release (after examples are tagged)

1. Run `make bundle-examples` locally to verify it pulls the correct `ref-examples` tag
2. Tag `v0.1.0` — triggers `.github/workflows/release.yml`:
   - Bundles latest `ref-examples` into `bundled-examples/`
   - Builds tarballs for all 4 platforms
   - Creates GitHub release with tarballs attached
   - Opens PR on `homebrew-tap` updating `Formula/ref.rb` with new version + SHA256
3. Merge the homebrew-tap PR
4. Verify: `brew install ref-cli/tap/ref && ref init` on a clean machine
