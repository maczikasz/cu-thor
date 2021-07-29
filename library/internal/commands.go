package internal

import (
	"io"
	"os/exec"
	"syscall"
)

// StartedCommand is an executor.Command that has successfully started
type StartedCommand struct {
	Cmd          *exec.Cmd
	CmdStatus    string
	OutputBuffer *MultiplexingBuffer
	AfterDone    func()
}

func (c *StartedCommand) Output() io.Reader {

	return c.OutputBuffer.Reader()
}

// Stop will send a SIGTERM signal to the underlying process
func (c *StartedCommand) Stop() error {
	err := c.Cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		return err
	}
	return nil
}

// ForceStop will send a SIGKILL signal to the underlying process
func (c *StartedCommand) ForceStop() error {
	err := c.Cmd.Process.Kill()
	if err != nil {
		return err
	}
	return nil
}

func (c *StartedCommand) Status() string {
	return c.CmdStatus
}

//WaitForStatus will wait until the underlying process complentes, then executes cleanup and sets the status
func (c *StartedCommand) WaitForStatus() {

	err := c.Cmd.Wait()

	_ = c.OutputBuffer.Close()
	c.AfterDone()

	if err != nil {
		c.CmdStatus = ABORTED
		return
	}

	if c.Cmd.ProcessState.Success() {

		c.CmdStatus = FINISHED
	} else {
		c.CmdStatus = ABORTED
	}
}

// FailedCommand represents a command that could not start
type FailedCommand struct {
}

// Output will always return nil
func (f FailedCommand) Output() io.Reader {
	return nil
}

// Stop is NOOP
func (f FailedCommand) Stop() error {
	//NOOP
	return nil
}

// ForceStop is NOOP
func (f FailedCommand) ForceStop() error {
	//NOOP
	return nil
}

// WaitForStatus is NOOP
func (f FailedCommand) WaitForStatus() {
	//NOOP
}

// Status will always return "failed"
func (f FailedCommand) Status() string {
	return FAILED
}
