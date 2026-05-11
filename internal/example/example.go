package example

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Frontmatter struct {
	Tags       []string `yaml:"tags"`
	Variants   []string `yaml:"variants"`
	Versions   []string `yaml:"versions"`
	MinVersion string   `yaml:"min_version"`
}

type Entry struct {
	Comment string   // description text (without leading "# ")
	Command string   // shell command
	Tags    []string // inline tags extracted from comment, e.g. ["GNU"]
}

type Example struct {
	Name        string
	Frontmatter Frontmatter
	Entries     []Entry
	Raw         string // full original file content
}

// SearchText returns a single string for fuzzy matching: name + all entry comments.
func (e *Example) SearchText() string {
	parts := []string{e.Name}
	for _, en := range e.Entries {
		parts = append(parts, en.Comment)
	}
	return strings.Join(parts, " ")
}

var inlineTagRe = regexp.MustCompile(`^\[([A-Z][A-Z0-9a-z]*)\]\s*`)

// Parse parses a .txt example file. name is the command name (e.g. "tar").
func Parse(name, content string) *Example {
	ex := &Example{Name: name, Raw: content}

	body := content
	if strings.HasPrefix(content, "---\n") {
		if end := strings.Index(content[4:], "\n---\n"); end >= 0 {
			_ = yaml.Unmarshal([]byte(content[4:end+4]), &ex.Frontmatter)
			body = content[end+9:]
		}
	}

	var comment, cmd string
	flush := func() {
		if comment != "" && cmd != "" {
			ex.Entries = append(ex.Entries, makeEntry(comment, cmd))
		}
		comment, cmd = "", ""
	}

	for _, line := range strings.Split(body, "\n") {
		switch {
		case strings.HasPrefix(line, "# "):
			flush()
			comment = strings.TrimPrefix(line, "# ")
		case line == "":
			flush()
		case comment != "" && cmd == "":
			cmd = line
		}
	}
	flush()

	return ex
}

func makeEntry(comment, cmd string) Entry {
	e := Entry{Command: cmd}
	rest := comment
	for {
		m := inlineTagRe.FindStringSubmatchIndex(rest)
		if m == nil {
			break
		}
		e.Tags = append(e.Tags, rest[m[2]:m[3]])
		rest = rest[m[1]:]
	}
	e.Comment = rest
	return e
}
