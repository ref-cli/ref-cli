package clipboard

import (
	"os/exec"
	"runtime"
	"strings"
)

// Write copies text to the system clipboard.
func Write(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	default:
		// Try xclip first, fall back to xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		}
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
