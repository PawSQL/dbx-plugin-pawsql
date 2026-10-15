package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// openURL opens url in the user's default browser. It only accepts http(s)
// URLs and never invokes a shell (args are passed straight to exec), so there
// is no command-injection surface. The sandboxed UI iframe cannot navigate to
// an external page on its own, so the native sidecar performs the open.
func openURL(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("url is required")
	}
	lower := strings.ToLower(url)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return fmt.Errorf("only http(s) URLs can be opened")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}
	return nil
}
