// Package executor will execute an arbitrary command on linux in its own namespace and with restricted resources using cgroups
// To use the executor package some all of the following prerequisites must be true
// * The app using the executor must be run as root
// * Cgroups v2 must be enabled on the host machine
// * Cgroups v1 must be unmounted or empty, so it can be unmounted by the executor
// * The host must cupport the io, memory, and cpu controllers in cgroups v2
package executor

import (
	"errors"
	"fmt"
	"github.com/maczikasz/cu-thor/library/internal"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type (
	Executor struct {
		cgroupsInfo map[string]string
	}

	//Config holds the configuration needed to setup the required cgroups in order to provide resource restrictions
	//CpuPercent represent the allowed percentage of CPU time, it should be 0 < CpuPercent < 100
	//MemoryBytes represent the accessible maximum memory in bytes
	//WriteBytePerSec and ReadBytePerSec represent the allowed maximum bytes to be read from or written to, this will be applied to all disks
	Config struct {
		CpuPercent      int
		MemoryBytes     int64
		WriteBytePerSec int64
		ReadBytePerSec  int64
	}
)

var expectedControllers = []string{"io", "memory", "cpu"}

const (
	cgroupFolder     = "/sys/fs/cgroup"
	defaultCPUPeriod = 100_000
)

// InitializeExecutorFromConfig will initialize a new executor by making sure that it has the neccesary cgroups setup to operate
func InitializeExecutorFromConfig(config Config) (*Executor, error) {

	if config.CpuPercent == 0 {
		return nil, errors.New("invalid cpu percentage")
	}

	err := ensureCgroupsV2()

	cgroupInfo := createCgroupsInfoFromConfig(config)
	if err != nil {
		return nil, err
	}

	return &Executor{
		cgroupsInfo: cgroupInfo,
	}, nil
}

func createCgroupsInfoFromConfig(config Config) map[string]string {
	result := make(map[string]string)

	result["io"] = fmt.Sprintf("wbps=%d rbps=%d", config.WriteBytePerSec, config.ReadBytePerSec)
	result["cpu"] = fmt.Sprintf("%d %d", defaultCPUPeriod*config.CpuPercent, defaultCPUPeriod)
	result["memory"] = strconv.FormatInt(config.MemoryBytes, 10)
	result["memory_high"] = strconv.FormatInt(config.MemoryBytes*75/100, 10)

	return result
}

func ensureCgroupsV2() error {
	err := ensureCgroupsV2Mounted()

	if err != nil {
		return err
	}

	err = ensureCgroupV2Controllers()

	if err != nil {
		return err
	}

	return nil
}

func ensureCgroupV2Controllers() error {
	content, err := ioutil.ReadFile(filepath.Join(cgroupFolder, "cgroup.controllers"))

	if err != nil {
		return fmt.Errorf("failed to read cgroup controllers %w", err)
	}

	contentStr := string(content)

	for _, controller := range expectedControllers {
		if !strings.Contains(contentStr, controller) {
			return errors.New(fmt.Sprintf("%s controller is not enabled for cgroups v2", controller))
		}
	}

	subtreeControlPath := filepath.Join(cgroupFolder, "cgroup.subtree_control")
	missingSubtreeControllers, err := findMissingSubtreeControllers(subtreeControlPath)

	if err != nil {
		return fmt.Errorf("failed check cgroup subtree controls %w", err)
	}

	if len(missingSubtreeControllers) != 0 {
		addNewControllers := strings.Join(missingSubtreeControllers, " ")
		err = os.WriteFile(subtreeControlPath, []byte(addNewControllers), os.ModeAppend)

		if err != nil {
			return fmt.Errorf("failed enable cgroup subtree controls %w", err)
		}
	}

	missingSubtreeControllers, err = findMissingSubtreeControllers(subtreeControlPath)

	if err != nil {
		return fmt.Errorf("failed check cgroup subtree controls %w", err)
	}

	if len(missingSubtreeControllers) != 0 {
		return fmt.Errorf("failed enable cgroup subtree controls, still missing %v", missingSubtreeControllers)
	}

	return nil
}

func findMissingSubtreeControllers(subtreeControlPath string) ([]string, error) {
	subtreeControlContent, err := ioutil.ReadFile(subtreeControlPath)
	if err != nil {
		return nil, err
	}

	subtreeControlContentStr := string(subtreeControlContent)

	var missingSubtreeControllers []string

	for _, controller := range expectedControllers {
		if !strings.Contains(subtreeControlContentStr, controller) {
			missingSubtreeControllers = append(missingSubtreeControllers, "+"+controller)
		}
	}
	return missingSubtreeControllers, nil
}

func ensureCgroupsV2Mounted() error {
	err := ensureCgroupsV1IsNotMounted()
	if err != nil {
		return fmt.Errorf("failed to unmount cgroupv1 %w", err)
	}

	if cgroupsV2Mounted() {
		return nil
	}

	err = syscall.Mount("cgroup2", cgroupFolder, "cgroup2", 0, "")

	if err != nil {
		return fmt.Errorf("failed to mount cgroupv2 %w", err)
	}

	if cgroupsV2Mounted() {
		return nil
	}

	return errors.New("failed to mount cgroups v2")
}

func cgroupsV2Mounted() bool {
	mountCommand := exec.Command("mount", "-t", "cgroup2")
	output, err := mountCommand.Output()

	if err != nil {
		return false
	}
	return len(output) != 0
}

func ensureCgroupsV1IsNotMounted() error {
	mountCommand := exec.Command("mount", "-t", "cgroup")
	output, err := mountCommand.Output()

	if err != nil {
		return fmt.Errorf("failed to check v1 cgroups, %w", err)
	}

	if len(output) == 0 {
		return nil
	}

	for _, line := range strings.Split(string(output), "\n") {
		path := strings.Split(line, " ")[2]
		err = syscall.Unmount(path, 0)
		if err != nil {
			return fmt.Errorf("failed to unmount v1 cgroup at %s, %w", path, err)
		}
	}

	return nil
}

// StartJob will execute the supplied command with the supplied args
// processId is used to create the underlying cgroup for the process to be executed in
//
// Before the job is started the following restrictions will be set in cgroups:
// * cpu.max set based on the CpuPercent of the Config of the Executor
// * memory.max set based on the MemoryBytes of the Config of the Executor
// * memory.high set based on the MemoryBytes * 0.75 of the Config of the Executor
// * io.max set based on the WriteBytePerSec and ReadBytePerSec of the Config of the Executor for all disks
//
// in case there is any problem with the setup of the process, these cgroup controls cannot be created an error will be returned
// In case the process setup was successful a Command will be returned
// The Command will either have "running" or "failed" state based on whether the process was succesfully started
// Further interaction with the process should happen through the returned Command instance
func (e Executor) StartJob(processId string, command string, args []string) (Command, error) {
	err := e.createCgroup(processId)

	if err != nil {
		cleanCgroups(processId)
		return nil, err
	}

	cmd := exec.Command(command, args...)

	outputBuffer := &internal.MultiplexingBuffer{}

	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = outputBuffer
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWUSER | syscall.CLONE_NEWNET | syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}

	err = cmd.Start()

	if err != nil {
		return &internal.FailedCommand{}, nil
	}

	pid := cmd.Process.Pid

	err = e.addPidToCgroups(processId, pid)
	if err != nil {
		killErr := cmd.Process.Kill()

		if killErr != nil {
			return nil, fmt.Errorf("the process could not be added to cgroup and killed in cleanup %w", killErr)
		}

		return nil, fmt.Errorf("failed to add process to cgroup %w", killErr)
	}

	c := &internal.StartedCommand{
		Cmd:          cmd,
		CmdStatus:    internal.RUNNING,
		OutputBuffer: outputBuffer,
		AfterDone: func() {
			cleanCgroups(processId)
		},
	}
	return c, nil
}

func cleanCgroups(processId string) {
	processCgroupFolder := filepath.Join(cgroupFolder, processId)

	_ = syscall.Rmdir(processCgroupFolder)
}

func (e Executor) createCgroup(processId string) error {
	processCgroupFolder := filepath.Join(cgroupFolder, processId)
	err := os.Mkdir(processCgroupFolder, 0600)

	if err != nil {
		return fmt.Errorf("failed to create cgroup %w", err)
	}

	err = os.WriteFile(filepath.Join(processCgroupFolder, "cpu.max"), []byte(e.cgroupsInfo["cpu"]), os.ModeAppend)
	if err != nil {
		return fmt.Errorf("failed to add resouce control %w", err)
	}

	err = os.WriteFile(filepath.Join(processCgroupFolder, "memory.max"), []byte(e.cgroupsInfo["memory"]), os.ModeAppend)
	if err != nil {
		return fmt.Errorf("failed to add resouce control %w", err)
	}

	err = os.WriteFile(filepath.Join(processCgroupFolder, "memory.high"), []byte(e.cgroupsInfo["memory_high"]), os.ModeAppend)
	if err != nil {
		return fmt.Errorf("failed to add resouce control %w", err)
	}

	diskLines, err := readDisks()
	if err != nil {
		return err
	}

	for _, diskLine := range diskLines {
		parsedDiskLine := strings.Split(diskLine, "\t")
		if len(parsedDiskLine) >= 2 {
			err = os.WriteFile(
				filepath.Join(processCgroupFolder, "io.max"),
				[]byte(fmt.Sprintf("%s %s", parsedDiskLine[1], e.cgroupsInfo["io"])),
				os.ModeAppend,
			)
			if err != nil {
				return fmt.Errorf("failed to add resouce control %w", err)
			}
		}
	}

	return nil
}

func readDisks() ([]string, error) {
	lookupDisks := exec.Command("lsblk", "-n", "-d")
	output, err := lookupDisks.Output()

	if err != nil {
		return nil, err
	}

	outputStr := string(output)

	blkDeviceLines := strings.Split(outputStr, "\n")
	return blkDeviceLines, nil
}

func (e Executor) addPidToCgroups(processId string, pid int) error {
	processCgroupFolder := filepath.Join(cgroupFolder, processId)

	err := os.WriteFile(filepath.Join(processCgroupFolder, "cgroup.procs"), []byte(strconv.Itoa(pid)), os.ModeAppend)
	if err != nil {
		return err
	}
	return nil
}
