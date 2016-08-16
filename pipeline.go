package flowbase

import (
	"fmt"
	"os"
	"reflect"
)

type PipelineRunner struct {
	processes []Process
}

func NewPipelineRunner() *PipelineRunner {
	return &PipelineRunner{}
}

func (pl *PipelineRunner) AddProcess(proc Process) {
	pl.processes = append(pl.processes, proc)
}

func (pl *PipelineRunner) AddProcesses(procs ...Process) {
	for _, proc := range procs {
		pl.AddProcess(proc)
	}
}

func (pl *PipelineRunner) PrintProcesses() {
	for i, proc := range pl.processes {
		fmt.Printf("Process %d: %v\n", i, reflect.TypeOf(proc))
	}
}

func (pl *PipelineRunner) Run() {
	if !LogExists {
		InitLogAudit()
	}
	if len(pl.processes) == 0 {
		Error.Println("PipelineRunner: The PipelineRunner is empty. Did you forget to add the processes to it?")
		os.Exit(1)
	}
	for i, proc := range pl.processes {
		if i < len(pl.processes)-1 {
			Debug.Printf("PipelineRunner: Starting process %d of type %v: in new go-routine...\n", i, reflect.TypeOf(proc))
			go proc.Run()
		} else {
			Debug.Printf("PipelineRunner: Starting process %d of type %v: in main go-routine...\n", i, reflect.TypeOf(proc))
			proc.Run()
		}
	}
}
