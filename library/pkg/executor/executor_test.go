package executor

import (
	"github.com/maczikasz/cu-thor/library/internal"
	"testing"
)

func TestExecutor_StartJob(t *testing.T) {
	executor, err := InitializeExecutorFromConfig(
		Config{
			CpuPercent:      10,
			MemoryBytes:     100000,
			WriteBytePerSec: 1000,
			ReadBytePerSec:  1000,
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	job, err := executor.StartJob("test", "echo", []string{"test"})

	if err != nil {
		t.Fatal(err)
	}

	if internal.RUNNING != job.Status() {
		t.Fatal("Could not start echo command")
	}

	job.WaitForStatus()

	if internal.FINISHED != job.Status() {
		t.Fatalf("Job status was not finished, but %s", job.Status())
	}
}

func TestExecutor_FailsToStartJob(t *testing.T) {
	executor, err := InitializeExecutorFromConfig(
		Config{
			CpuPercent:      10,
			MemoryBytes:     100000,
			WriteBytePerSec: 1000,
			ReadBytePerSec:  1000,
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	job, err := executor.StartJob("test", "fake", []string{"command"})

	if err != nil {
		t.Fatal(err)
	}

	if internal.FAILED != job.Status() {
		t.Fatal("Fake command should have failed")
	}
}
