package chrome

import (
	"fmt"
	"os/exec"
)

// Open opens the specified URL in Google Chrome.
func Open(url string) error {
	if url == "" {
		return fmt.Errorf("url must not be empty")
	}

	cmd := exec.Command("open", "-a", "Google Chrome", url)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// If you need to wait for the process to finish, you can use cmd.Wait()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

func Close() error {
	cmd := exec.Command("osascript", "-e", "quit app \"Google Chrome\"")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}
	return nil
}