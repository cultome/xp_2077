package ui

import (
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type openURLMsg struct {
	url string
	err error
}

// browserCommand returns the command and args used to open a URL on the given
// GOOS. Split out from openURL so it can be unit-tested without launching anything.
func browserCommand(goos, url string) (string, []string) {
	switch goos {
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		return "open", []string{url}
	default:
		return "xdg-open", []string{url}
	}
}

// openURL returns a tea.Cmd that launches the system browser for url without
// blocking the UI; the outcome is reported via openURLMsg.
func openURL(url string) tea.Cmd {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil
	}
	name, args := browserCommand(runtime.GOOS, url)
	return func() tea.Msg {
		err := exec.Command(name, args...).Start()
		return openURLMsg{url: url, err: err}
	}
}
