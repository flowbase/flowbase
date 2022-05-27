package flowbase

import (
	"os"
	"os/exec"
	"sync"
	"testing"
)

func TestSetWfName(t *testing.T) {
	initTestLogs()
	net := NewNetwork("TestNetwork")

	expectedWfName := "TestNetwork"
	if net.name != expectedWfName {
		t.Errorf("Network name is wrong, should be %s but is %s\n", net.name, expectedWfName)
	}
}

// --------------------------------------------------------------------------------
// MapToTag helper process
// --------------------------------------------------------------------------------

type MapToTags struct {
	BaseProcess
	mapFunc func(ip *Packet) map[string]string
}

func NewMapToTags(net *Network, name string, mapFunc func(ip *Packet) map[string]string) *MapToTags {
	p := &MapToTags{
		BaseProcess: NewBaseProcess(net, name),
		mapFunc:     mapFunc,
	}
	p.InitInPort(p, "in")
	p.InitOutPort(p, "out")
	net.AddProc(p)
	return p
}

func (p *MapToTags) In() *InPort   { return p.InPort("in") }
func (p *MapToTags) Out() *OutPort { return p.OutPort("out") }

func (p *MapToTags) Run() {
	defer p.CloseOutPorts()
	for ip := range p.In().Chan {
		newTags := p.mapFunc(ip)
		ip.AddTags(newTags)
		p.Out().Send(ip)
	}
}

// --------------------------------------------------------------------------------
// FileSource helper process
// --------------------------------------------------------------------------------

// FileSource is initiated with a set of file paths, which it will send as a
// stream of File IPs on its outport Out()
type FileSource struct {
	BaseProcess
	filePaths []string
}

// NewFileSource returns a new initialized FileSource process
func NewFileSource(net *Network, name string, filePaths ...string) *FileSource {
	p := &FileSource{
		BaseProcess: NewBaseProcess(net, name),
		filePaths:   filePaths,
	}
	p.InitOutPort(p, "out")
	net.AddProc(p)
	return p
}

// Out returns the out-port, on which file IPs based on the file paths the
// process was initialized with, will be retrieved.
func (p *FileSource) Out() *OutPort { return p.OutPort("out") }

// Run runs the FileSource process
func (p *FileSource) Run() {
	defer p.CloseOutPorts()
	for _, filePath := range p.filePaths {
		newIP := NewPacket(filePath)
		p.Out().Send(newIP)
	}
}

// --------------------------------
// BogusProcess helper process
// --------------------------------

// A process with does just satisfy the Process interface, without doing any
// actual work.
type BogusProcess struct {
	BaseProcess
	name       string
	WasRun     bool
	WasRunLock sync.Mutex
}

func NewBogusProcess(name string) *BogusProcess {
	return &BogusProcess{WasRun: false, name: name}
}

func (p *BogusProcess) Run() {
	p.WasRunLock.Lock()
	p.WasRun = true
	p.WasRunLock.Unlock()
}

func (p *BogusProcess) Name() string {
	return p.name
}

func (p *BogusProcess) Ready() bool {
	return true
}

func ensureFailsProgram(testName string, crasher func(), t *testing.T) {
	// After https://talks.golang.org/2014/testing.slide#23
	if os.Getenv("BE_CRASHER") == "1" {
		crasher()
	}
	cmd := exec.Command(os.Args[0], "-test.run="+testName)
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
