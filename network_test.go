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
// CombinatoricsProcess helper process
// --------------------------------------------------------------------------------

type CombinatoricsProcess struct {
	BaseProcess
	name string
	A    *OutParamPort
	B    *OutParamPort
	C    *OutParamPort
}

func NewCombinatoricsProcess(name string) *CombinatoricsProcess {
	a := NewOutParamPort("a")
	b := NewOutParamPort("b")
	c := NewOutParamPort("c")
	p := &CombinatoricsProcess{
		A:    a,
		B:    b,
		C:    c,
		name: name,
	}
	a.process = p
	b.process = p
	c.process = p
	return p
}

func (p *CombinatoricsProcess) InPorts() map[string]*InPort {
	return map[string]*InPort{}
}
func (p *CombinatoricsProcess) OutPorts() map[string]*OutPort {
	return map[string]*OutPort{}
}
func (p *CombinatoricsProcess) InParamPorts() map[string]*InParamPort {
	return map[string]*InParamPort{}
}
func (p *CombinatoricsProcess) OutParamPorts() map[string]*OutParamPort {
	return map[string]*OutParamPort{
		p.A.Name(): p.A,
		p.B.Name(): p.B,
		p.C.Name(): p.C,
	}
}

func (p *CombinatoricsProcess) Run() {
	defer p.A.Close()
	defer p.B.Close()
	defer p.C.Close()

	for _, a := range []string{"a1", "a2", "a3"} {
		for _, b := range []string{"b1", "b2", "b3"} {
			for _, c := range []string{"c1", "c2", "c3"} {
				p.A.Send(a)
				p.B.Send(b)
				p.C.Send(c)
			}
		}
	}
}

func (p *CombinatoricsProcess) Name() string {
	return p.name
}

func (p *CombinatoricsProcess) Ready() bool { return true }

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
	defer p.CloseAllOutPorts()
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
	defer p.CloseAllOutPorts()
	for _, filePath := range p.filePaths {
		newIP, err := NewFileIP(filePath)
		if err != nil {
			p.Fail(err)
		}
		p.Out().Send(newIP)
	}
}

// --------------------------------------------------------------------------------
// ParamSource helper process
// --------------------------------------------------------------------------------

// ParamSource will feed parameters on an out-port
type ParamSource struct {
	BaseProcess
	params []string
}

// NewParamSource returns a new ParamSource
func NewParamSource(wf *Workflow, name string, params ...string) *ParamSource {
	p := &ParamSource{
		BaseProcess: NewBaseProcess(wf, name),
		params:      params,
	}
	p.InitOutParamPort(p, "out")
	wf.AddProc(p)
	return p
}

// Out returns the out-port, on which parameters the process was initialized
// with, will be retrieved.
func (p *ParamSource) Out() *OutParamPort { return p.OutParamPort("out") }

// Run runs the process
func (p *ParamSource) Run() {
	defer p.CloseAllOutPorts()
	for _, param := range p.params {
		p.Out().Send(param)
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
