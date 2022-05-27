package flowbase

import (
	"os"
	"os/exec"
	"reflect"
	"sync"
	"testing"
)

func TestSetWfName(t *testing.T) {
	initTestLogs()
	wf := NewWorkflow("TestWorkflow", 16)

	expectedWfName := "TestWorkflow"
	if wf.name != expectedWfName {
		t.Errorf("Workflow name is wrong, should be %s but is %s\n", wf.name, expectedWfName)
	}
}

func TestMaxConcurrentTasksCapacity(t *testing.T) {
	initTestLogs()
	wf := NewWorkflow("TestWorkflow", 16)

	if cap(wf.concurrentTasks) != 16 {
		t.Error("Wrong number of concurrent tasks")
	}
}

func TestAddProc(t *testing.T) {
	initTestLogs()

	wf := NewWorkflow("TestAddProcsWf", 16)

	proc1 := NewBogusProcess("bogusproc1")
	wf.AddProc(proc1)
	proc2 := NewBogusProcess("bogusproc2")
	wf.AddProc(proc2)

	if len(wf.procs) != 2 {
		t.Error("Wrong number of processes")
	}

	if !reflect.DeepEqual(reflect.TypeOf(wf.procs["bogusproc1"]), reflect.TypeOf(&BogusProcess{})) {
		t.Error("Bogusproc1 was not of the right type!")
	}
	if !reflect.DeepEqual(reflect.TypeOf(wf.procs["bogusproc2"]), reflect.TypeOf(&BogusProcess{})) {
		t.Error("Bogusproc2 was not of the right type!")
	}
}

// --------------------------------------------------------------------------------
// MapToTag helper process
// --------------------------------------------------------------------------------

type MapToTags struct {
	BaseProcess
	mapFunc func(ip *FileIP) map[string]string
}

func NewMapToTags(wf *Workflow, name string, mapFunc func(ip *FileIP) map[string]string) *MapToTags {
	p := &MapToTags{
		BaseProcess: NewBaseProcess(wf, name),
		mapFunc:     mapFunc,
	}
	p.InitInPort(p, "in")
	p.InitOutPort(p, "out")
	wf.AddProc(p)
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
func NewFileSource(wf *Workflow, name string, filePaths ...string) *FileSource {
	p := &FileSource{
		BaseProcess: NewBaseProcess(wf, name),
		filePaths:   filePaths,
	}
	p.InitOutPort(p, "out")
	wf.AddProc(p)
	return p
}

// Out returns the out-port, on which file IPs based on the file paths the
// process was initialized with, will be retrieved.
func (p *FileSource) Out() *OutPort { return p.OutPort("out") }

// Run runs the FileSource process
func (p *FileSource) Run() {
	defer p.CloseOutPorts()
	for _, filePath := range p.filePaths {
		newIP, err := NewFileIP(filePath)
		if err != nil {
			p.Fail(err)
		}
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
