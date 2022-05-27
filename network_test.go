package flowbase

import (
	"fmt"
	"testing"
	t "testing"

	"github.com/stretchr/testify/assert"
)

func TestAddProcesses(t *t.T) {
	InitLogError()

	proc1 := NewBogusProcess()
	proc2 := NewBogusProcess()
	pipeline := NewNet()
	pipeline.AddProcesses(proc1, proc2)

	assert.EqualValues(t, len(pipeline.processes), 2)

	assert.IsType(t, &BogusProcess{}, pipeline.processes[0], "Process 1 was not of the right type!")
	assert.IsType(t, &BogusProcess{}, pipeline.processes[1], "Process 2 was not of the right type!")
}

func TestRunProcessesInNet(t *t.T) {
	proc1 := NewBogusProcess()
	proc2 := NewBogusProcess()

	pipeline := NewNet()
	pipeline.AddProcesses(proc1, proc2)
	pipeline.Run()

	// Only the last process is supposed to be run by the pipeline directly,
	// while the others are only run if an output is pulled on an out-port,
	// but since we haven't connected the tasks here, only the last one
	// should be ran in this case.
	assert.False(t, proc1.WasRun, "Process 1 was run!")
	assert.True(t, proc2.WasRun, "Process 2 was not run!")
}

func ExamplePrintProcesses() {
	proc1 := NewBogusProcess()
	proc2 := NewBogusProcess()

	pipeline := NewNet()
	pipeline.AddProcesses(proc1, proc2)
	pipeline.Run()

	pipeline.PrintProcesses()
	// Output:
	// Process 0: *flowbase.BogusProcess
	// Process 1: *flowbase.BogusProcess
}

func TestNetworkWithPortObjects(t *testing.T) {
	net := NewNet()
	rs := NewRandomSender()
	sp := NewStringPrinter()
	sp.InStrings.From(rs.OutRandomStrings)
	net.AddProcesses(rs, sp)
	net.Run()
}

// --------------------------------
// Helper stuff
// --------------------------------

type RandomSender struct {
	OutRandomStrings *OutPort[string]
}

func NewRandomSender() *RandomSender {
	return &RandomSender{NewOutPort[string]("random-strings")}
}

func (p *RandomSender) Ready() bool {
	return p.OutRandomStrings.ready
}

func (p *RandomSender) Run() {
	for _, str := range []string{"abc", "xyz", "urg"} {
		p.OutRandomStrings.Send(str)
	}
	p.OutRandomStrings.Close()
}

type StringPrinter struct {
	InStrings *InPort[string]
}

func NewStringPrinter() *StringPrinter {
	return &StringPrinter{NewInPort[string]("strings")}
}

func (p *StringPrinter) Ready() bool {
	return p.InStrings.ready
}

func (p *StringPrinter) Run() {
	for str := range p.InStrings.Chan {
		fmt.Println(str)
	}
}

// A process with does just satisfy the Process interface, without doing any
// actual work.
type BogusProcess struct {
	Process
	WasRun bool
}

func NewBogusProcess() *BogusProcess {
	return &BogusProcess{WasRun: false}
}

func (p *BogusProcess) Run() {
	p.WasRun = true
}

func (p *BogusProcess) IsConnected() bool {
	return true
}
