package bundled

import (
	"embed"
	"strings"
)

//go:embed all:examples
var FS embed.FS

// ExamplesVersion is read from bundled-examples/VERSION at build time.
var ExamplesVersion string

func init() {
	data, err := FS.ReadFile("examples/VERSION")
	if err != nil {
		panic("ref: bundled examples are missing VERSION — binary was not built correctly")
	}
	ExamplesVersion = strings.TrimSpace(string(data))
}
