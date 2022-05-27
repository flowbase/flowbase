package flowbase

import (
	"fmt"
	"testing"
	t "testing"

	"github.com/stretchr/testify/assert"
)

func TestAddNodes(t *t.T) {
	InitLogError()

	proc1 := NewBogusProcess()
	proc2 := NewBogusProcess()
	pipeline := NewNetwork()
	pipeline.AddNodes(proc1, proc2)

	assert.EqualValues(t, len(pipeline.nodes), 2)

	assert.IsType(t, &BogusProcess{}, pipeline.nodes[0], "Process 1 was not of the right type!")
	assert.IsType(t, &BogusProcess{}, pipeline.nodes[1], "Process 2 was not of the right type!")
}

func TestRunProcessesInNetwork(t *t.T) {
	proc1 := NewBogusProcess()
	proc2 := NewBogusProcess()

	pipeline := NewNetwork()
	pipeline.AddNodes(proc1, proc2)
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

	pipeline := NewNetwork()
	pipeline.AddNodes(proc1, proc2)
	pipeline.Run()

	pipeline.PrintProcesses()
	// Output:
	// Node 0: *flowbase.BogusProcess
	// Node 1: *flowbase.BogusProcess
}

func TestNetworkWithPortObjects(t *testing.T) {
	net := NewNetwork()
	rs := NewRandomSender("rs")
	sp := NewStringPrinter("sp")
	sp.InStrings.From(rs.OutRandomStrings)
	net.AddNodes(rs, sp)
	net.Run()
}

// --------------------------------
// Helper stuff
// --------------------------------

type RandomSender struct {
	name             string
	OutRandomStrings *OutPort[string]
}

func NewRandomSender(name string) *RandomSender {
	return &RandomSender{name, NewOutPort[string]("random-strings")}
}

func (n *RandomSender) Name() string {
	return n.name
}

func (n *RandomSender) Ready() bool {
	return n.OutRandomStrings.ready
}

func (n *RandomSender) Run() {
	for _, str := range []string{"abc", "xyz", "urg"} {
		n.OutRandomStrings.Send(str)
	}
	n.OutRandomStrings.Close()
}

type StringPrinter struct {
	name      string
	InStrings *InPort[string]
}

func NewStringPrinter(name string) *StringPrinter {
	return &StringPrinter{name, NewInPort[string]("strings")}
}

func (n *StringPrinter) Name() string {
	return n.name
}

func (n *StringPrinter) Ready() bool {
	return n.InStrings.ready
}

func (n *StringPrinter) Run() {
	for str := range n.InStrings.Chan() {
		fmt.Println(str)
	}
}

// A process with does just satisfy the Process interface, without doing any
// actual work.
type BogusProcess struct {
	Node
	WasRun bool
}

func NewBogusProcess() *BogusProcess {
	return &BogusProcess{WasRun: false}
}

func (n *BogusProcess) Name() string { return "bogus-process" }

func (n *BogusProcess) Run() {
	n.WasRun = true
}

func (p *BogusProcess) IsConnected() bool {
	return true
}
