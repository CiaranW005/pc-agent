package executor

import "os/exec"

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Shutdown() error {
	return exec.Command("systemctl", "poweroff").Run()
}

func (e *Executor) Restart() error {
	return exec.Command("systemctl", "reboot").Run()
}
