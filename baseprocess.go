package flowbase

import "fmt"

// BaseProcess provides a skeleton for processes, such as the main Process
// component, and the custom components in the scipipe/components library
type BaseProcess struct {
	name     string
	workflow *Workflow
	inPorts  map[string]*InPort
	outPorts map[string]*OutPort
}

// NewBaseProcess returns a new BaseProcess, connected to the provided workflow,
// and with the name name
func NewBaseProcess(wf *Workflow, name string) BaseProcess {
	return BaseProcess{
		workflow: wf,
		name:     name,
		inPorts:  make(map[string]*InPort),
		outPorts: make(map[string]*OutPort),
	}
}

// Name returns the name of the process
func (p *BaseProcess) Name() string {
	return p.name
}

// Workflow returns the workflow the process is connected to
func (p *BaseProcess) Workflow() *Workflow {
	return p.workflow
}

// ------------------------------------------------
// In-port stuff
// ------------------------------------------------

// InPort returns the in-port with name portName
func (p *BaseProcess) InPort(portName string) *InPort {
	if _, ok := p.inPorts[portName]; !ok {
		p.Failf("No such in-port ('%s'). Please check your workflow code!", portName)
	}
	return p.inPorts[portName]
}

// InitInPort adds the in-port port to the process, with name portName
func (p *BaseProcess) InitInPort(proc WorkflowProcess, portName string) {
	if _, ok := p.inPorts[portName]; ok {
		p.Failf("Such an in-port ('%s') already exists. Please check your workflow code!", portName)
	}
	ipt := NewInPort(portName)
	ipt.process = proc
	p.inPorts[portName] = ipt
}

// InPorts returns a map of all the in-ports of the process, keyed by their
// names
func (p *BaseProcess) InPorts() map[string]*InPort {
	return p.inPorts
}

// DeleteInPort deletes an InPort object from the process
func (p *BaseProcess) DeleteInPort(portName string) {
	if _, ok := p.inPorts[portName]; !ok {
		p.Failf("No such in-port ('%s'). Please check your workflow code!", portName)
	}
	delete(p.inPorts, portName)
}

// ------------------------------------------------
// Out-port stuff
// ------------------------------------------------

// InitOutPort adds the out-port port to the process, with name portName
func (p *BaseProcess) InitOutPort(proc WorkflowProcess, portName string) {
	if _, ok := p.outPorts[portName]; ok {
		p.Failf("Such an out-port ('%s') already exists. Please check your workflow code!", portName)
	}
	opt := NewOutPort(portName)
	opt.process = proc
	p.outPorts[portName] = opt
}

// OutPort returns the out-port with name portName
func (p *BaseProcess) OutPort(portName string) *OutPort {
	if _, ok := p.outPorts[portName]; !ok {
		p.Failf("No such out-port ('%s'). Please check your workflow code!", portName)
	}
	return p.outPorts[portName]
}

// OutPorts returns a map of all the out-ports of the process, keyed by their
// names
func (p *BaseProcess) OutPorts() map[string]*OutPort {
	return p.outPorts
}

// DeleteOutPort deletes a OutPort object from the process
func (p *BaseProcess) DeleteOutPort(portName string) {
	if _, ok := p.outPorts[portName]; !ok {
		p.Failf("No such out-port ('%s'). Please check your workflow code!", portName)
	}
	delete(p.outPorts, portName)
}

// ------------------------------------------------
// Other stuff
// ------------------------------------------------

// Ready checks whether all the process' ports are connected
func (p *BaseProcess) Ready() (isReady bool) {
	isReady = true
	for portName, port := range p.inPorts {
		if !port.Ready() {
			p.Failf("InPort (%s) is not connected - check your workflow code!", portName)
			isReady = false
		}
	}
	for portName, port := range p.outPorts {
		if !port.Ready() {
			p.Failf("OutPort (%s) is not connected - check your workflow code!", portName)
			isReady = false
		}
	}
	return isReady
}

// CloseOutPorts closes all (normal) out-ports
func (p *BaseProcess) CloseOutPorts() {
	for _, p := range p.OutPorts() {
		p.Close()
	}
}

// Failf fails with a message that includes the process name
func (p *BaseProcess) Failf(msg string, parts ...interface{}) {
	p.Fail(fmt.Sprintf(msg, parts...))
}

// Fail fails with a message that includes the process name
func (p *BaseProcess) Fail(msg interface{}) {
	Failf("[Process:%s] %s", p.Name(), msg)
}

func (p *BaseProcess) Auditf(msg string, parts ...interface{}) {
	p.Audit(fmt.Sprintf(msg, parts...))
}

func (p *BaseProcess) Audit(msg interface{}) {
	Audit.Printf("[Process:%s] %s"+"\n", p.Name(), msg)
}

func (p *BaseProcess) receiveOnInPorts() (ips map[string]*FileIP, inPortsOpen bool) {
	inPortsOpen = true
	ips = make(map[string]*FileIP)
	// Read input IPs on in-ports and set up path mappings
	for inpName, inPort := range p.InPorts() {
		Debug.Printf("[Process %s]: Receieving on inPort (%s) ...", p.name, inpName)
		ip, open := <-inPort.Chan
		if !open {
			inPortsOpen = false
			continue
		}
		Debug.Printf("[Process %s]: Got ip (%s) ...", p.name, ip.Path())
		ips[inpName] = ip
	}
	return
}
