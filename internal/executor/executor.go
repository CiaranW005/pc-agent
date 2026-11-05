package executor

import (
	"fmt"
	"os/exec"
	"strings"
)

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Shutdown() error {
	// Request shutdown through systemd. The service account must be authorized
	// to perform this action. A nil error confirms the request, not shutdown completion.
	output, err := exec.Command("systemctl", "poweroff").CombinedOutput()
	if err != nil {
		return fmt.Errorf("Shutdown failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (e *Executor) Restart() error {
	// Request restart through systemd. The service account must be authorized
	// to perform this action. A nil error confirms the request, not restart completion.
	output, err := exec.Command("systemctl", "reboot").CombinedOutput()
	if err != nil {
		return fmt.Errorf("Reboot failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
