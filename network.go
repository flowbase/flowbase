// Package scipipe is a library for writing scientific workflows (sometimes
// also called "pipelines") of shell commands that depend on each other, in the
// Go programming languages. It was initially designed for problems in
// cheminformatics and bioinformatics, but should apply equally well to any
// domain involving complex pipelines of interdependent shell commands.
package scipipe

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ----------------------------------------------------------------------------
// Workflow
// ----------------------------------------------------------------------------

// Workflow is the centerpiece of the functionality in SciPipe, and is a
// container for a pipeline of processes making up a workflow. It has various
// methods for coordination the execution of the pipeline as a whole, such as
// keeping track of the maxiumum number of concurrent tasks, as well as helper
// methods for creating new processes, that automatically gets plugged in to the
// workflow on creation
type Workflow struct {
	name              string
	procs             map[string]WorkflowProcess
	concurrentTasks   chan struct{}
	concurrentTasksMx sync.Mutex
	sink              *Sink
	driver            WorkflowProcess
	logFile           string
	PlotConf          WorkflowPlotConf
}

// WorkflowPlotConf contains configuraiton for plotting the workflow as a graph
// with graphviz
type WorkflowPlotConf struct {
	EdgeLabels bool
}

// WorkflowProcess is an interface for processes to be handled by Workflow
type WorkflowProcess interface {
	Name() string
	InPorts() map[string]*InPort
	OutPorts() map[string]*OutPort
	InParamPorts() map[string]*InParamPort
	OutParamPorts() map[string]*OutParamPort
	Ready() bool
	Run()
	Fail(interface{})
	Failf(string, ...interface{})
}

// ----------------------------------------------------------------------------
// Factory function(s)
// ----------------------------------------------------------------------------

// NewWorkflow returns a new Workflow
func NewWorkflow(name string, maxConcurrentTasks int) *Workflow {
	wf := newWorkflowWithoutLogging(name, maxConcurrentTasks)

	// Set up logging
	allowedCharsPtrn := regexp.MustCompile("[^a-z0-9_]")
	wfNameNormalized := allowedCharsPtrn.ReplaceAllString(strings.ToLower(name), "-")
	wf.logFile = "log/scipipe-" + time.Now().Format("20060102-150405") + "-" + wfNameNormalized + ".log"
	InitLogAuditToFile(wf.logFile)

	return wf
}

// NewWorkflowCustomLogFile returns a new Workflow, with
func NewWorkflowCustomLogFile(name string, maxConcurrentTasks int, logFile string) *Workflow {
	wf := newWorkflowWithoutLogging(name, maxConcurrentTasks)

	wf.logFile = logFile
	InitLogAuditToFile(logFile)

	return wf
}

func newWorkflowWithoutLogging(name string, maxConcurrentTasks int) *Workflow {
	wf := &Workflow{
		name:            name,
		procs:           map[string]WorkflowProcess{},
		concurrentTasks: make(chan struct{}, maxConcurrentTasks),
		PlotConf:        WorkflowPlotConf{EdgeLabels: true},
	}
	sink := NewSink(wf, name+"_default_sink")
	wf.sink = sink
	wf.driver = sink
	return wf
}

// ----------------------------------------------------------------------------
// Main API methods
// ----------------------------------------------------------------------------

// Name returns the name of the workflow
func (wf *Workflow) Name() string {
	return wf.name
}

// NewProc returns a new process based on a commandPattern (See the
// documentation for scipipe.NewProcess for more details about the pattern) and
// connects the process to the workflow
func (wf *Workflow) NewProc(procName string, commandPattern string) *Process {
	proc := NewProc(wf, procName, commandPattern)
	return proc
}

// Proc returns the process with name procName from the workflow
func (wf *Workflow) Proc(procName string) WorkflowProcess {
	if _, ok := wf.procs[procName]; !ok {
		wf.Failf("No process named (%s)", procName)
	}
	return wf.procs[procName]
}

// ProcsSorted returns the processes of the workflow, in an array, sorted by the
// process names
func (wf *Workflow) ProcsSorted() []WorkflowProcess {
	keys := []string{}
	for k := range wf.Procs() {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	procs := []WorkflowProcess{}
	for _, k := range keys {
		procs = append(procs, wf.Proc(k))
	}
	return procs
}

// Procs returns a map of all processes keyed by their names in the workflow
func (wf *Workflow) Procs() map[string]WorkflowProcess {
	return wf.procs
}

// AddProc adds a Process to the workflow, to be run when the workflow runs
func (wf *Workflow) AddProc(proc WorkflowProcess) {
	if wf.procs[proc.Name()] != nil {
		wf.Failf("A process with name (%s) already exists in the workflow! Use a more unique name!", proc.Name())
	}
	wf.procs[proc.Name()] = proc
}

// AddProcs takes one or many Processes and adds them to the workflow, to be run
// when the workflow runs.
func (wf *Workflow) AddProcs(procs ...WorkflowProcess) {
	for _, proc := range procs {
		wf.AddProc(proc)
	}
}

// Sink returns the sink process of the workflow
func (wf *Workflow) Sink() *Sink {
	return wf.sink
}

// SetSink sets the sink of the workflow to the provided sink process
func (wf *Workflow) SetSink(sink *Sink) {
	if wf.sink.Ready() {
		wf.Fail("Trying to replace a sink which is already connected. Are you combining SetSink() with ConnectFinalOutPort()? That is not allowed!")
	}
	wf.sink = sink
}

// IncConcurrentTasks increases the conter for how many concurrent tasks are
// currently running in the workflow
func (wf *Workflow) IncConcurrentTasks(slots int) {
	// We must lock so that multiple processes don't end up with partially "filled slots"
	wf.concurrentTasksMx.Lock()
	for i := 0; i < slots; i++ {
		wf.concurrentTasks <- struct{}{}
		Debug.Println("Increased concurrent tasks")
	}
	wf.concurrentTasksMx.Unlock()
}

// DecConcurrentTasks decreases the conter for how many concurrent tasks are
// currently running in the workflow
func (wf *Workflow) DecConcurrentTasks(slots int) {
	for i := 0; i < slots; i++ {
		<-wf.concurrentTasks
		Debug.Println("Decreased concurrent tasks")
	}
}

// PlotGraph writes the workflow structure to a dot file
func (wf *Workflow) PlotGraph(filePath string) {
	dot := wf.DotGraph()
	createDirs(filePath)
	dotFile, err := os.Create(filePath)
	CheckWithMsg(err, "Could not create dot file "+filePath)
	_, errDot := dotFile.WriteString(dot)
	if errDot != nil {
		wf.Failf("Could not write to DOT-file %s: %s", dotFile.Name(), errDot)
	}
}

// PlotGraphPDF writes the workflow structure to a dot file, and also runs the
// graphviz dot command to produce a PDF file (requires graphviz, with the dot
// command, installed on the system)
func (wf *Workflow) PlotGraphPDF(filePath string) {
	wf.PlotGraph(filePath)
	ExecCmd(fmt.Sprintf("dot -Tpdf %s -o %s.pdf", filePath, filePath))
}

// DotGraph generates a graph description in DOT format
// (See https://en.wikipedia.org/wiki/DOT_%28graph_description_language%29)
// If Workflow.PlotConf.EdgeLabels is set to true, a label containing the
// in-port and out-port to which edges are connected to, will be printed.
func (wf *Workflow) DotGraph() (dot string) {
	dot = fmt.Sprintf(`digraph "%s" {`+"\n", wf.Name())
	dot += `  rankdir=LR;` + "\n"
	dot += `  graph [fontname="Arial",fontsize=13,color="#384A52",fontcolor="#384A52"];` + "\n"
	dot += `  node  [fontname="Arial",fontsize=11,color="#384A52",fontcolor="#384A52",fillcolor="#EFF2F5",shape=box,style=filled];` + "\n"
	dot += `  edge  [fontname="Arial",fontsize=9, color="#384A52",fontcolor="#384A52"];` + "\n"

	con := ""
	remToDotPtn := regexp.MustCompile(`^.*\.`)
	for _, p := range wf.ProcsSorted() {
		dot += fmt.Sprintf(`  "%s" [shape=box];`+"\n", p.Name())
		// File connections
		for opname, op := range p.OutPorts() {
			for rpname, rp := range op.RemotePorts {
				if wf.PlotConf.EdgeLabels {
					con += fmt.Sprintf(`  "%s" -> "%s" [taillabel="%s", headlabel="%s"];`+"\n", op.Process().Name(), rp.Process().Name(), remToDotPtn.ReplaceAllString(opname, ""), remToDotPtn.ReplaceAllString(rpname, ""))
				} else {
					con += fmt.Sprintf(`  "%s" -> "%s";`+"\n", op.Process().Name(), rp.Process().Name())
				}
			}
		}
		// Parameter connections
		for popname, pop := range p.OutParamPorts() {
			for rpname, rp := range pop.RemotePorts {
				if wf.PlotConf.EdgeLabels {
					con += fmt.Sprintf(`  "%s" -> "%s" [style="dashed", taillabel="%s", headlabel="%s"];`+"\n", pop.Process().Name(), rp.Process().Name(), remToDotPtn.ReplaceAllString(popname, ""), remToDotPtn.ReplaceAllString(rpname, ""))
				} else {
					con += fmt.Sprintf(`  "%s" -> "%s" [style="dashed"];`+"\n", pop.Process().Name(), rp.Process().Name())
				}
			}
		}
	}
	dot += con
	dot += "}\n"
	return
}

// ----------------------------------------------------------------------------
// Run methods
// ----------------------------------------------------------------------------

// Run runs all the processes of the workflow
func (wf *Workflow) Run() {
	wf.runProcs(wf.procs)
}

// RunTo runs all processes upstream of, and including, the process with
// names provided as arguments
func (wf *Workflow) RunTo(finalProcNames ...string) {
	procs := []WorkflowProcess{}
	for _, procName := range finalProcNames {
		procs = append(procs, wf.Proc(procName))
	}
	wf.RunToProcs(procs...)
}

// RunToRegex runs all processes upstream of, and including, the process
// whose name matches any of the provided regexp patterns
func (wf *Workflow) RunToRegex(procNamePatterns ...string) {
	procsToRun := []WorkflowProcess{}
	for _, pattern := range procNamePatterns {
		regexpPtrn := regexp.MustCompile(pattern)
		for procName, proc := range wf.Procs() {
			matches := regexpPtrn.MatchString(procName)
			if matches {
				procsToRun = append(procsToRun, proc)
			}
		}
	}
	wf.RunToProcs(procsToRun...)
}

// RunToProcs runs all processes upstream of, and including, the process strucs
// provided as arguments
func (wf *Workflow) RunToProcs(finalProcs ...WorkflowProcess) {
	procsToRun := map[string]WorkflowProcess{}
	for _, finalProc := range finalProcs {
		procsToRun = mergeWFMaps(procsToRun, upstreamProcsForProc(finalProc))
		procsToRun[finalProc.Name()] = finalProc
	}
	wf.runProcs(procsToRun)
}

// ----------------------------------------------------------------------------
// Helper methods for running the workflow
// ----------------------------------------------------------------------------

// runProcs runs a specified set of processes only
func (wf *Workflow) runProcs(procs map[string]WorkflowProcess) {
	wf.reconnectDeadEndConnections(procs)

	if !wf.readyToRun(procs) {
		wf.Fail("Workflow not ready to run, due to previously reported errors, so exiting.")
	}

	for _, proc := range procs {
		Debug.Printf(wf.name+": Starting process (%s) in new go-routine", proc.Name())
		go proc.Run()
	}

	Debug.Printf("%s: Starting driver process (%s) in main go-routine", wf.name, wf.driver.Name())
	wf.Auditf("Starting workflow (Writing log to %s)", wf.logFile)
	wf.driver.Run()
	wf.Auditf("Finished workflow (Log written to %s)", wf.logFile)
}

func (wf *Workflow) readyToRun(procs map[string]WorkflowProcess) bool {
	if len(procs) == 0 {
		Error.Println(wf.name + ": The workflow is empty. Did you forget to add the processes to it?")
		return false
	}
	if wf.sink == nil {
		Error.Println(wf.name + ": sink is nil!")
		return false
	}
	for _, proc := range procs {
		if !proc.Ready() {
			Error.Println(wf.name + ": Not everything connected. Workflow shutting down.")
			return false
		}
	}
	return true
}

// reconnectDeadEndConnections disonnects connections to processes which are
// not in the set of processes to be run, and, if an out-port for a process
// supposed to be run gets disconnected, its out-port(s) will be connected to
// the sink instead, to make sure it is properly executed.
func (wf *Workflow) reconnectDeadEndConnections(procs map[string]WorkflowProcess) {
	foundNewDriverProc := false

	for _, proc := range procs {
		// OutPorts
		for _, opt := range proc.OutPorts() {
			for iptName, ipt := range opt.RemotePorts {
				// If the remotely connected process is not among the ones to run ...
				if ipt.Process() == nil {
					Debug.Printf("Disconnecting in-port (%s) from out-port (%s)", ipt.Name(), opt.Name())
					opt.Disconnect(iptName)
				} else if _, ok := procs[ipt.Process().Name()]; !ok {
					Debug.Printf("Disconnecting in-port (%s) from out-port (%s)", ipt.Name(), opt.Name())
					opt.Disconnect(iptName)
				}
			}
			if !opt.Ready() {
				Debug.Printf("Connecting disconnected out-port (%s) of process (%s) to workflow sink", opt.Name(), opt.Process().Name())
				wf.sink.From(opt)
			}
		}

		// OutParamPorts
		for _, pop := range proc.OutParamPorts() {
			for rppName, rpp := range pop.RemotePorts {
				// If the remotely connected process is not among the ones to run ...
				if rpp.Process() == nil {
					Debug.Printf("Disconnecting in-port (%s) from out-port (%s)", rpp.Name(), pop.Name())
					pop.Disconnect(rppName)
				} else if _, ok := procs[rpp.Process().Name()]; !ok {
					Debug.Printf("Disconnecting in-port (%s) from out-port (%s)", rpp.Name(), pop.Name())
					pop.Disconnect(rppName)
				}
			}
			if !pop.Ready() {
				Debug.Printf("Connecting disconnected out-port (%s) of process (%s) to workflow sink", pop.Name(), pop.Process().Name())
				wf.sink.FromParam(pop)
			}
		}

		if len(proc.OutPorts()) == 0 && len(proc.OutParamPorts()) == 0 {
			if foundNewDriverProc {
				wf.Failf("Found more than one process without out-ports nor out-param ports. Cannot use both as drivers (One of them being '%s'). Adapt your workflow accordingly.", proc.Name())
			}
			foundNewDriverProc = true
			wf.driver = proc
		}
	}

	if foundNewDriverProc && len(procs) > 1 { // Allow for a workflow with a single process
		// A process can't both be the driver and be included in the main procs
		// map, so if we have an alerative driver, it should not be in the main
		// procs map
		delete(wf.procs, wf.driver.Name())
	}
}

// upstreamProcsForProc returns all processes it is connected to, either
// directly or indirectly, via its in-ports and param-in-ports
func upstreamProcsForProc(proc WorkflowProcess) map[string]WorkflowProcess {
	procs := map[string]WorkflowProcess{}
	for _, inp := range proc.InPorts() {
		for _, rpt := range inp.RemotePorts {
			procs[rpt.Process().Name()] = rpt.Process()
			mergeWFMaps(procs, upstreamProcsForProc(rpt.Process()))
		}
	}
	for _, pip := range proc.InParamPorts() {
		for _, rpp := range pip.RemotePorts {
			procs[rpp.Process().Name()] = rpp.Process()
			mergeWFMaps(procs, upstreamProcsForProc(rpp.Process()))
		}
	}
	return procs
}

func mergeWFMaps(a map[string]WorkflowProcess, b map[string]WorkflowProcess) map[string]WorkflowProcess {
	for k, v := range b {
		a[k] = v
	}
	return a
}

func (wf *Workflow) Auditf(msg string, parts ...interface{}) {
	Audit.Printf("[Workflow:%s] %s\n", wf.Name(), fmt.Sprintf(msg, parts...))
}

func (wf *Workflow) Failf(msg string, parts ...interface{}) {
	wf.Fail(fmt.Sprintf(msg, parts...))
}

func (wf *Workflow) Fail(msg interface{}) {
	Failf("[Workflow:%s] %s", wf.Name(), msg)
}
