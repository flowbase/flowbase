package flowbase

import "fmt"

// Interface for all constituents of flow networks, including processes,
// networks and sub-networks
type Node interface {
	Name() string
	Ready() bool
	Run()
}

type BaseProcess struct {
	name     string
	network  *Network
	inPorts  map[string]IInPort
	outPorts map[string]IOutPort
}

func NewBaseProcess(net *Network, name string) BaseProcess {
	return BaseProcess{
		name:     name,
		network:  net,
		inPorts:  make(map[string]IInPort),
		outPorts: make(map[string]IOutPort),
	}
}

func (p *BaseProcess) Name() string {
	return p.name
}

func (p *BaseProcess) Network() *Network {
	return p.network
}

func (p *BaseProcess) InPort(portName string) any {
	if _, ok := p.inPorts[portName]; !ok {
		p.Failf("No such in-port ('%s'). Please check your workflow code!", portName)
	}
	return p.inPorts[portName]
}

// AddInPort adds the in-port port to the process
func (p *BaseProcess) AddInPort(node Node, inPort IInPort) {
	if _, ok := p.inPorts[inPort.Name()]; ok {
		p.Failf("Such an in-port ('%s') already exists. Please check your workflow code!", inPort.Name())
	}
	inPort.SetNode(node)
	p.inPorts[inPort.Name()] = inPort
}

func (p *BaseProcess) InPorts() map[string]IInPort {
	return p.inPorts
}

// DeleteInPort deletes an InPort object from the process
func (p *BaseProcess) DeleteInPort(portName string) {
	if _, ok := p.inPorts[portName]; !ok {
		p.Failf("No such in-port ('%s'). Please check your workflow code!", portName)
	}
	delete(p.inPorts, portName)
}

func (p *BaseProcess) OutPorts() map[string]IOutPort {
	return p.outPorts
}

// AddOutPort adds the in-port port to the process
func (p *BaseProcess) AddOutPort(node Node, outPort IOutPort) {
	if _, ok := p.outPorts[outPort.Name()]; ok {
		p.Failf("Such an in-port ('%s') already exists. Please check your workflow code!", outPort.Name())
	}
	outPort.SetNode(node)
	p.outPorts[outPort.Name()] = outPort
}

// DeleteOutPort deletes a OutPort object from the process
func (p *BaseProcess) DeleteOutPort(portName string) {
	if _, ok := p.outPorts[portName]; !ok {
		p.Failf("No such out-port ('%s'). Please check your workflow code!", portName)
	}
	delete(p.outPorts, portName)
}

func (p *BaseProcess) receiveOnInPorts() (ips map[string]any, inPortsOpen bool) {
	inPortsOpen = true
	ips = make(map[string]any)
	// Read input IPs on in-ports and set up path mappings
	for inpName, inPort := range p.InPorts() {
		ip, open := <-inPort.Chan()
		if !open {
			inPortsOpen = false
			continue
		}
		ips[inpName] = ip
	}
	return
}

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
