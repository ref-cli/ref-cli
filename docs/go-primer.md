# Go Primer for `ref-cli` Contributors

This document explains Go language features you will encounter when reading the
`ref-cli` source. It is not a complete Go tutorial — it focuses on the idioms
that actually appear in this codebase, with pointers to the real files.

---

## Table of Contents

1. [Packages and imports](#1-packages-and-imports)
2. [Variables, constants, and named types](#2-variables-constants-and-named-types)
3. [Structs and struct tags](#3-structs-and-struct-tags)
4. [Pointers](#4-pointers)
5. [Functions and multiple return values](#5-functions-and-multiple-return-values)
6. [Methods](#6-methods)
7. [Interfaces (implicit)](#7-interfaces-implicit)
8. [Error handling](#8-error-handling)
9. [Closures](#9-closures)
10. [The blank identifier `_`](#10-the-blank-identifier-_)
11. [`defer`](#11-defer)
12. [Slices and `append`](#12-slices-and-append)
13. [Type switches](#13-type-switches)
14. [`go:embed` — bundling files into the binary](#14-goembed--bundling-files-into-the-binary)
15. [`init()` — automatic initialization](#15-init--automatic-initialization)
16. [Build-time variables (`-ldflags`)](#16-build-time-variables--ldflags)

---

## 1. Packages and imports

Every `.go` file starts with a **package declaration**. Files in the same
directory must share the same package name.

```go
// internal/ai/ai.go
package ai
```

The special package name `main` marks the entry point of an executable binary.
Only the `cmd/ref/` directory uses `package main`.

### Internal packages

The `internal/` directory is a Go convention that enforces encapsulation: code
inside `internal/` can only be imported by code rooted at the parent module. So
`internal/ai` can be imported by `cmd/ref/` but **not** by any outside module.

### Import grouping

Imports are split into three groups by convention (enforced by `gofmt`):

```go
import (
    // 1. Standard library
    "fmt"
    "os"

    // 2. Third-party modules
    "github.com/spf13/cobra"

    // 3. Internal packages (same module)
    "github.com/ref-cli/ref-cli/internal/config"
)
```

See [`cmd/ref/cmd_view.go`](../cmd/ref/cmd_view.go) for a real example.

---

## 2. Variables, constants, and named types

### Package-level `var` blocks

Variables declared at the top of a file are accessible throughout the package:

```go
// cmd/ref/cmd_view.go
var (
    commentColor = lipgloss.AdaptiveColor{Light: "#555555", Dark: "#888888"}
    tagColor     = lipgloss.AdaptiveColor{Light: "#CC7700", Dark: "#FFAF5F"}
)
```

### `const` blocks

Constants are untyped or typed values that cannot change at runtime:

```go
// internal/update/update.go
const (
    binaryReleasesURL  = "https://api.github.com/repos/ref-cli/ref-cli/releases/latest"
    timeout            = 3 * time.Second
)
```

### Named types

Go lets you create a new type that is based on an existing one. This adds
type-safety and lets you attach methods:

```go
// internal/ai/ai.go
type Backend string  // a new type whose underlying type is string

const (
    Claude Backend = "claude"
    Codex  Backend = "codex"
)
```

`Backend` and `string` are distinct types — you cannot accidentally pass a plain
`string` where a `Backend` is expected without an explicit conversion.

---

## 3. Structs and struct tags

A **struct** groups fields together, similar to an object or record in other
languages.

```go
// internal/config/config.go
type Config struct {
    ExamplesVersion string `yaml:"examples_version"`
    LastUpdateCheck string `yaml:"last_update_check"`
    AIBackend       string `yaml:"ai_backend"`
}
```

### Struct tags

The backtick strings after each field — `` `yaml:"examples_version"` `` — are
**struct tags**. They are metadata read at runtime by libraries such as
`gopkg.in/yaml.v3` and `encoding/json`. Here they tell the YAML library which
key name maps to which Go field. They have no effect on normal Go code.

```go
// internal/update/update.go
type releaseResponse struct {
    TagName string `json:"tag_name"`
}
```

### Struct literals

Creating a struct value:

```go
cfg := &Config{ExamplesVersion: "v0.1.0", InstallMethod: "homebrew"}
```

Unspecified fields get their **zero value** (`""` for strings, `0` for numbers,
`false` for bools, `nil` for pointers/slices).

---

## 4. Pointers

A **pointer** holds the memory address of a value. The `*` prefix denotes a
pointer type; `&` takes the address of a value.

```go
func Load() (*Config, error) {  // returns a pointer to Config
    var c Config
    // ...
    return &c, nil              // & gives the address of c
}
```

**When you see `*T`** in a function signature, the function may return `nil` to
signal "nothing" (Go has no `null`/`None` at the language level for concrete
values, only for pointers, interfaces, slices, and maps).

```go
// internal/examples/examples.go — returns nil when file does not exist
func Find(dir, name string) (*example.Example, error) {
    // ...
    if os.IsNotExist(err) {
        return nil, nil   // no example, no error
    }
    // ...
}
```

---

## 5. Functions and multiple return values

Go functions can return **more than one value**. The idiomatic pattern is
`(result, error)`:

```go
func Load() (*Config, error) { ... }

// Caller:
cfg, err := config.Load()
if err != nil {
    return err
}
```

If you do not need one of the return values, discard it with `_`:

```go
home, _ := os.UserHomeDir()   // ignore the error
```

### Short variable declaration `:=`

`:=` declares and assigns in one step; it infers the type automatically:

```go
cfg, err := config.Load()  // cfg is *config.Config, err is error
```

This is only allowed inside a function body. Package-level variables use `var`.

---

## 6. Methods

A **method** is a function with a **receiver** — a special first parameter that
binds the function to a type.

```go
// internal/config/config.go
func (c *Config) Save() error {
    // c is a pointer to the Config being acted on
    data, _ := yaml.Marshal(c)
    return os.WriteFile(File(), data, 0o644)
}
```

The call site looks like `cfg.Save()` — identical to calling a method on an
object in Python or Java.

### Pointer receivers vs value receivers

- **Pointer receiver** (`*Config`): the method can modify the original struct.
  Use this when the method mutates state or when copying the struct is expensive.
- **Value receiver** (`Model`): the method receives a copy; it cannot modify the
  original.

```go
// internal/tui/model.go
func (m Model) Init() tea.Cmd { ... }   // value receiver — no mutation needed
func (m Model) Update(...) (tea.Model, tea.Cmd) { ... }
```

---

## 7. Interfaces (implicit)

Go interfaces are **satisfied implicitly** — a type just needs to have the right
methods; there is no `implements` keyword.

```go
// The bubbletea library defines:
type Model interface {
    Init() Cmd
    Update(Msg) (Model, Cmd)
    View() string
}
```

`tui.Model` satisfies this interface because it has all three methods (see
[`internal/tui/model.go`](../internal/tui/model.go)). No explicit declaration
is required.

### The `error` interface

`error` is a built-in interface with one method:

```go
type error interface {
    Error() string
}
```

Any type that has an `Error() string` method is an `error`. The standard library
and `fmt.Errorf` return values that implement this interface.

---

## 8. Error handling

Go does not have exceptions. Errors are ordinary values returned alongside the
result. The canonical pattern:

```go
data, err := os.ReadFile(path)
if err != nil {
    return nil, err   // propagate the error up
}
```

### Error wrapping with `%w`

`fmt.Errorf` with the `%w` verb wraps an error so callers can inspect the
original cause with `errors.Is` / `errors.As`:

```go
// internal/ai/ai.go
return "", fmt.Errorf("AI generation failed: %w", err)
```

This preserves the full error chain and adds context to the message.

---

## 9. Closures

A **closure** is a function defined inside another function that can reference
variables from the outer scope — even after the outer function returns.

```go
// internal/example/example.go
var comment, cmd string

flush := func() {
    if comment != "" && cmd != "" {
        ex.Entries = append(ex.Entries, makeEntry(comment, cmd))
    }
    comment, cmd = "", ""   // resets the outer variables
}

for _, line := range strings.Split(body, "\n") {
    // ...
    flush()   // called multiple times
}
flush()       // called once more at the end
```

`flush` "closes over" `comment`, `cmd`, and `ex` — it can read and write them
even though they live in the enclosing `Parse` function.

---

## 10. The blank identifier `_`

`_` discards a value without triggering Go's "declared but not used" compile
error:

```go
cfg.AIBackend = string(detected[0])
_ = cfg.Save()   // save best-effort; ignore the returned error explicitly
```

Using `_` for an error is a deliberate choice that says "I know this can fail
and I am choosing not to handle it here." It makes the intent visible to
reviewers.

---

## 11. `defer`

`defer` schedules a function call to run when the surrounding function returns,
regardless of whether it returns normally or via an error. It is the idiomatic
way to release resources:

```go
// internal/update/update.go
resp, err := client.Do(req)
if err != nil {
    return "", err
}
defer resp.Body.Close()   // guaranteed to run when latestTag() exits
```

Multiple `defer` calls run in **LIFO** order (last deferred, first executed).

---

## 12. Slices and `append`

A **slice** is a dynamically sized view into an array. The zero value of a
slice is `nil`, which is equivalent to an empty slice for most purposes.

```go
var found []Backend         // nil slice — len(found) == 0

found = append(found, Claude)  // append returns a (potentially new) slice
```

### Range loops

`range` iterates over a slice (or map, channel, string):

```go
for i, ex := range exs {
    names[i] = ex.Name   // i is the index, ex is a copy of the element
}

// If you don't need the index:
for _, ex := range exs { ... }
```

### `sort.Slice`

Sorting with an anonymous comparison function:

```go
// internal/examples/examples.go
sort.Slice(exs, func(i, j int) bool {
    return exs[i].Name < exs[j].Name
})
```

The anonymous function is a closure that captures `exs`.

---

## 13. Type switches

A **type switch** inspects the dynamic type of an interface value at runtime:

```go
// internal/tui/model.go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width    // msg is tea.WindowSizeMsg inside this case
        m.height = msg.Height
    case tea.KeyMsg:
        // msg is tea.KeyMsg here
    }
    // ...
}
```

`msg.(type)` is the type-assertion syntax used in a `switch`. Each `case`
matches a concrete type and re-binds `msg` to that type, giving you access to
its specific fields.

---

## 14. `go:embed` — bundling files into the binary

The `//go:embed` directive copies files from disk into the binary at **compile
time**. The compiler reads the directive (it must be directly above the
variable) and embeds the matching files:

```go
// internal/bundled/bundled.go
import "embed"

//go:embed all:examples
var FS embed.FS
```

`all:examples` embeds the entire `examples/` directory (including hidden files).
`FS` then acts like a read-only filesystem you can query at runtime — no disk
access needed after the binary is built. This is how `ref init` extracts
examples without a network call.

---

## 15. `init()` — automatic initialization

A file can have an `init()` function. Go calls it automatically after all
package-level variables are initialized, before `main()` runs. You cannot call
`init()` yourself.

```go
// internal/bundled/bundled.go
func init() {
    data, err := FS.ReadFile("examples/VERSION")
    if err != nil {
        panic("ref: bundled examples are missing VERSION — binary was not built correctly")
    }
    ExamplesVersion = strings.TrimSpace(string(data))
}
```

`init()` is commonly used for one-time setup that does not fit neatly into a
constructor. A package can have multiple `init()` functions (even in the same
file); they run in declaration order.

---

## 16. Build-time variables (`-ldflags`)

Go allows injecting values into package-level variables at link time:

```go
// cmd/ref/main.go
var version = "dev"   // default when built without ldflags
```

When the release pipeline builds the binary, it passes something like:

```
go build -ldflags "-X main.version=v1.2.3" ./cmd/ref
```

The linker replaces the string `"dev"` with `"v1.2.3"` in the final binary.
This is the standard Go way to embed version numbers without code changes.

---

## Further reading

- [A Tour of Go](https://go.dev/tour/) — interactive browser-based intro
- [Effective Go](https://go.dev/doc/effective_go) — official style guide
- [Go by Example](https://gobyexample.com/) — short, runnable examples
- [`go.mod`](../go.mod) — lists every external dependency used by this module
