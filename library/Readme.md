# Cu-thor library

Cu-thor library provides the functionality to execute commands in isolation, and with limited resources

## Usage

To use the library the following prerequisites must be met:
* The app using the executor must be run as root
* Cgroups v2 must be enabled on the host machine
* Cgroups v1 must be unmounted or empty, so it can be unmounted by the executor
* The host must cupport the io, memory, and cpu controllers in cgroups v2

Then a new `Config` instance must be created to define the resource limits  
Using this config a new `Executor` instance could be initialized  
That can be used to `StartJob`   

```
config := Config{
	CpuPercent:      10,
	MemoryBytes:     100000,
	WriteBytePerSec: 1000,
	ReadBytePerSec:  1000,
}

executor, _ := InitializeExecutorFromConfig(config)

job, _ := executor.StartJob("test", "echo", []string{"test"})

job.Status() //RUNNING

job.WaitForStatus() // Will block until the app is finished

job.Status() //FINISHED

```


Most cases `WaitForStatus` should be started on its own go routine, to not block further execution


```
config := Config{
	CpuPercent:      10,
	MemoryBytes:     100000,
	WriteBytePerSec: 1000,
	ReadBytePerSec:  1000,
}

executor, _ := InitializeExecutorFromConfig(config)

job, _ := executor.StartJob("test", "sleep", []string{"5"})

job.Status() //RUNNING

go job.WaitForStatus() // Will block until the app is finished

for {
    status := job.Status()
    if status == "finished" {
        break;
    }
	time.Sleep(1 * time.Second)
}

```

## Testing

Since the library heavily depends on linux internals it cannot be easily tested on non linux host machines, to make this easier the 

`make test_vagrant`

make target was prepared to run the tests inside a vagrant machine. Currently, only virtualbox provider is supported for vagrant

In order to test the library from non-linux host machines the following vagrant box can be used: https://app.vagrantup.com/maczikasz/boxes/centos-go-cgroups2

To run the test on a compatible host machine the `make test` command could be used. 

__*Running the tests will unmount cgroups v1 and mount cgroups v2 on the host machine using vagrant is generally a good idea*__
