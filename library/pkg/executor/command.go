package executor

import (
	"io"
)

// The Command inteface represents an execution of a command by an Executor
//
// It provides the methods to interact with the underlying process
// !! The WaitForStatus method waits for the process to complete, sets the status and closes the output buffer!!
// !! Most cases WaitForStatus should be called in a goroutine, failing to call WaitForStatus will result in the status not being updated and the buffer not being closed !!
//
// The Stop and ForceStop methods allow to stop or kill the underlying process
// The Output method will return the output of the process and will keep the returned io.Reader open, until the process is running
// The Status call will return the current status of the process, it will be always one of ("running", "finished", "aborted", "failed"), should be called after WaitForStatus
//
type Command interface {
	Stop() error
	ForceStop() error
	WaitForStatus()
	Status() string
	Output() io.Reader
}
