package flowbase

import (
	"fmt"
	"os"
	"reflect"
)

type Net struct {
	processes []Process
}

func NewNet() *Net {
	return &Net{}
}

func (pl *Net) AddProcess(proc Process) {
	pl.processes = append(pl.processes, proc)
}

func (pl *Net) AddProcesses(procs ...Process) {
	for _, proc := range procs {
		pl.AddProcess(proc)
	}
}

func (pl *Net) PrintProcesses() {
	for i, proc := range pl.processes {
		fmt.Printf("Process %d: %v\n", i, reflect.TypeOf(proc))
	}
}

func (pl *Net) Run() {
	if !LogExists {
		InitLogAudit()
	}
	if len(pl.processes) == 0 {
		Error.Println("Net: The Net is empty. Did you forget to add the processes to it?")
		os.Exit(1)
	}
	for i, proc := range pl.processes {
		if i < len(pl.processes)-1 {
			Debug.Printf("Net: Starting process %d of type %v: in new go-routine...\n", i, reflect.TypeOf(proc))
			go proc.Run()
		} else {
			Debug.Printf("Net: Starting process %d of type %v: in main go-routine...\n", i, reflect.TypeOf(proc))
			proc.Run()
		}
	}
}
