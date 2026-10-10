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
	output, err := exec.Command("systemctl", "poweroff").CombinedOutput()
	if err != nil {
		return fmt.Errorf("Shutdown failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (e *Executor) Restart() error {
	output, err := exec.Command("systemctl", "reboot").CombinedOutput()
	if err != nil {
		return fmt.Errorf("Reboot failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
