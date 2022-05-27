package flowbase

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
// Network
// ----------------------------------------------------------------------------

// Network is the centerpiece of the functionality in FlowBase, and is a
// container for a pipeline of processes making up a workflow. It has various
// methods for coordination the execution of the pipeline as a whole, such as
// keeping track of the maxiumum number of concurrent tasks, as well as helper
// methods for creating new processes, that automatically gets plugged in to the
// workflow on creation
type Network struct {
	name              string
	procs             map[string]Node
	concurrentTasks   chan struct{}
	concurrentTasksMx sync.Mutex
	sink              *Sink
	driver            Node
	logFile           string
	PlotConf          NetworkPlotConf
}

// NetworkPlotConf contains configuraiton for plotting the workflow as a graph
// with graphviz
type NetworkPlotConf struct {
	EdgeLabels bool
}

// Node is an interface for processes to be handled by Network
type Node interface {
	Name() string
	InPorts() map[string]*InPort
	OutPorts() map[string]*OutPort
	Ready() bool
	Run()
	Fail(interface{})
	Failf(string, ...interface{})
}

// ----------------------------------------------------------------------------
// Factory function(s)
// ----------------------------------------------------------------------------

// NewNetwork returns a new Network
func NewNetwork(name string, maxConcurrentTasks int) *Network {
	net := newNetworkWithoutLogging(name, maxConcurrentTasks)

	// Set up logging
	allowedCharsPtrn := regexp.MustCompile("[^a-z0-9_]")
	wfNameNormalized := allowedCharsPtrn.ReplaceAllString(strings.ToLower(name), "-")
	net.logFile = "log/flowbase-" + time.Now().Format("20060102-150405") + "-" + wfNameNormalized + ".log"
	InitLogAuditToFile(net.logFile)

	return net
}

// NewNetworkCustomLogFile returns a new Network, with
func NewNetworkCustomLogFile(name string, maxConcurrentTasks int, logFile string) *Network {
	net := newNetworkWithoutLogging(name, maxConcurrentTasks)

	net.logFile = logFile
	InitLogAuditToFile(logFile)

	return net
}

func newNetworkWithoutLogging(name string, maxConcurrentTasks int) *Network {
	net := &Network{
		name:            name,
		procs:           map[string]Node{},
		concurrentTasks: make(chan struct{}, maxConcurrentTasks),
		PlotConf:        NetworkPlotConf{EdgeLabels: true},
	}
	sink := NewSink(net, name+"_default_sink")
	net.sink = sink
	net.driver = sink
	return net
}

// ----------------------------------------------------------------------------
// Main API methods
// ----------------------------------------------------------------------------

// Name returns the name of the workflow
func (net *Network) Name() string {
	return net.name
}

// Proc returns the process with name procName from the workflow
func (net *Network) Proc(procName string) Node {
	if _, ok := net.procs[procName]; !ok {
		net.Failf("No process named (%s)", procName)
	}
	return net.procs[procName]
}

// ProcsSorted returns the processes of the workflow, in an array, sorted by the
// process names
func (net *Network) ProcsSorted() []Node {
	keys := []string{}
	for k := range net.Procs() {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	procs := []Node{}
	for _, k := range keys {
		procs = append(procs, net.Proc(k))
	}
	return procs
}

// Procs returns a map of all processes keyed by their names in the workflow
func (net *Network) Procs() map[string]Node {
	return net.procs
}

// AddProc adds a Process to the workflow, to be run when the workflow runs
func (net *Network) AddProc(node Node) {
	if net.procs[node.Name()] != nil {
		net.Failf("A process with name (%s) already exists in the workflow! Use a more unique name!", node.Name())
	}
	net.procs[node.Name()] = node
}

// AddProcs takes one or many Processes and adds them to the workflow, to be run
// when the workflow runs.
func (net *Network) AddProcs(procs ...Node) {
	for _, node := range procs {
		net.AddProc(node)
	}
}

// Sink returns the sink process of the workflow
func (net *Network) Sink() *Sink {
	return net.sink
}

// SetSink sets the sink of the workflow to the provided sink process
func (net *Network) SetSink(sink *Sink) {
	if net.sink.Ready() {
		net.Fail("Trying to replace a sink which is already connected. Are you combining SetSink() with ConnectFinalOutPort()? That is not allowed!")
	}
	net.sink = sink
}

// IncConcurrentTasks increases the conter for how many concurrent tasks are
// currently running in the workflow
func (net *Network) IncConcurrentTasks(slots int) {
	// We must lock so that multiple processes don't end up with partially "filled slots"
	net.concurrentTasksMx.Lock()
	for i := 0; i < slots; i++ {
		net.concurrentTasks <- struct{}{}
		Debug.Println("Increased concurrent tasks")
	}
	net.concurrentTasksMx.Unlock()
}

// DecConcurrentTasks decreases the conter for how many concurrent tasks are
// currently running in the workflow
func (net *Network) DecConcurrentTasks(slots int) {
	for i := 0; i < slots; i++ {
		<-net.concurrentTasks
		Debug.Println("Decreased concurrent tasks")
	}
}

// PlotGraph writes the workflow structure to a dot file
func (net *Network) PlotGraph(filePath string) {
	dot := net.DotGraph()
	createDirs(filePath)
	dotFile, err := os.Create(filePath)
	CheckWithMsg(err, "Could not create dot file "+filePath)
	_, errDot := dotFile.WriteString(dot)
	if errDot != nil {
		net.Failf("Could not write to DOT-file %s: %s", dotFile.Name(), errDot)
	}
}

// PlotGraphPDF writes the workflow structure to a dot file, and also runs the
// graphviz dot command to produce a PDF file (requires graphviz, with the dot
// command, installed on the system)
func (net *Network) PlotGraphPDF(filePath string) {
	net.PlotGraph(filePath)
	ExecCmd(fmt.Sprintf("dot -Tpdf %s -o %s.pdf", filePath, filePath))
}

// DotGraph generates a graph description in DOT format
// (See https://en.wikipedia.org/wiki/DOT_%28graph_description_language%29)
// If Network.PlotConf.EdgeLabels is set to true, a label containing the
// in-port and out-port to which edges are connected to, will be printed.
func (net *Network) DotGraph() (dot string) {
	dot = fmt.Sprintf(`digraph "%s" {`+"\n", net.Name())
	dot += `  rankdir=LR;` + "\n"
	dot += `  graph [fontname="Arial",fontsize=13,color="#384A52",fontcolor="#384A52"];` + "\n"
	dot += `  node  [fontname="Arial",fontsize=11,color="#384A52",fontcolor="#384A52",fillcolor="#EFF2F5",shape=box,style=filled];` + "\n"
	dot += `  edge  [fontname="Arial",fontsize=9, color="#384A52",fontcolor="#384A52"];` + "\n"

	con := ""
	remToDotPtn := regexp.MustCompile(`^.*\.`)
	for _, p := range net.ProcsSorted() {
		dot += fmt.Sprintf(`  "%s" [shape=box];`+"\n", p.Name())
		// File connections
		for opname, op := range p.OutPorts() {
			for rpname, rp := range op.RemotePorts {
				if net.PlotConf.EdgeLabels {
					con += fmt.Sprintf(`  "%s" -> "%s" [taillabel="%s", headlabel="%s"];`+"\n", op.Process().Name(), rp.Process().Name(), remToDotPtn.ReplaceAllString(opname, ""), remToDotPtn.ReplaceAllString(rpname, ""))
				} else {
					con += fmt.Sprintf(`  "%s" -> "%s";`+"\n", op.Process().Name(), rp.Process().Name())
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
func (net *Network) Run() {
	net.runProcs(net.procs)
}

// RunTo runs all processes upstream of, and including, the process with
// names provided as arguments
func (net *Network) RunTo(finalProcNames ...string) {
	procs := []Node{}
	for _, procName := range finalProcNames {
		procs = append(procs, net.Proc(procName))
	}
	net.RunToProcs(procs...)
}

// RunToRegex runs all processes upstream of, and including, the process
// whose name matches any of the provided regexp patterns
func (net *Network) RunToRegex(procNamePatterns ...string) {
	procsToRun := []Node{}
	for _, pattern := range procNamePatterns {
		regexpPtrn := regexp.MustCompile(pattern)
		for procName, node := range net.Procs() {
			matches := regexpPtrn.MatchString(procName)
			if matches {
				procsToRun = append(procsToRun, node)
			}
		}
	}
	net.RunToProcs(procsToRun...)
}

// RunToProcs runs all processes upstream of, and including, the process strucs
// provided as arguments
func (net *Network) RunToProcs(finalProcs ...Node) {
	procsToRun := map[string]Node{}
	for _, finalProc := range finalProcs {
		procsToRun = mergeWFMaps(procsToRun, upstreamProcsForProc(finalProc))
		procsToRun[finalProc.Name()] = finalProc
	}
	net.runProcs(procsToRun)
}

// ----------------------------------------------------------------------------
// Helper methods for running the workflow
// ----------------------------------------------------------------------------

// runProcs runs a specified set of processes only
func (net *Network) runProcs(procs map[string]Node) {
	net.reconnectDeadEndConnections(procs)

	if !net.readyToRun(procs) {
		net.Fail("Network not ready to run, due to previously reported errors, so exiting.")
	}

	for _, node := range procs {
		Debug.Printf(net.name+": Starting process (%s) in new go-routine", node.Name())
		go node.Run()
	}

	Debug.Printf("%s: Starting driver process (%s) in main go-routine", net.name, net.driver.Name())
	net.Auditf("Starting workflow (Writing log to %s)", net.logFile)
	net.driver.Run()
	net.Auditf("Finished workflow (Log written to %s)", net.logFile)
}

func (net *Network) readyToRun(procs map[string]Node) bool {
	if len(procs) == 0 {
		Error.Println(net.name + ": The workflow is empty. Did you forget to add the processes to it?")
		return false
	}
	if net.sink == nil {
		Error.Println(net.name + ": sink is nil!")
		return false
	}
	for _, node := range procs {
		if !node.Ready() {
			Error.Println(net.name + ": Not everything connected. Network shutting down.")
			return false
		}
	}
	return true
}

// reconnectDeadEndConnections disonnects connections to processes which are
// not in the set of processes to be run, and, if an out-port for a process
// supposed to be run gets disconnected, its out-port(s) will be connected to
// the sink instead, to make sure it is properly executed.
func (net *Network) reconnectDeadEndConnections(procs map[string]Node) {
	foundNewDriverProc := false

	for _, node := range procs {
		// OutPorts
		for _, opt := range node.OutPorts() {
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
				net.sink.From(opt)
			}
		}

		if len(node.OutPorts()) == 0 {
			if foundNewDriverProc {
				net.Failf("Found more than one process without out-ports. Cannot use both as drivers (One of them being '%s'). Adapt your workflow accordingly.", node.Name())
			}
			foundNewDriverProc = true
			net.driver = node
		}
	}

	if foundNewDriverProc && len(procs) > 1 { // Allow for a workflow with a single process
		// A process can't both be the driver and be included in the main procs
		// map, so if we have an alerative driver, it should not be in the main
		// procs map
		delete(net.procs, net.driver.Name())
	}
}

// upstreamProcsForProc returns all processes it is connected to, either
// directly or indirectly, via its in-ports and param-in-ports
func upstreamProcsForProc(node Node) map[string]Node {
	procs := map[string]Node{}
	for _, inp := range node.InPorts() {
		for _, rpt := range inp.RemotePorts {
			procs[rpt.Process().Name()] = rpt.Process()
			mergeWFMaps(procs, upstreamProcsForProc(rpt.Process()))
		}
	}
	return procs
}

func mergeWFMaps(a map[string]Node, b map[string]Node) map[string]Node {
	for k, v := range b {
		a[k] = v
	}
	return a
}

func (net *Network) Auditf(msg string, parts ...interface{}) {
	Audit.Printf("[Network:%s] %s\n", net.Name(), fmt.Sprintf(msg, parts...))
}

func (net *Network) Failf(msg string, parts ...interface{}) {
	net.Fail(fmt.Sprintf(msg, parts...))
}

func (net *Network) Fail(msg interface{}) {
	Failf("[Network:%s] %s", net.Name(), msg)
}
