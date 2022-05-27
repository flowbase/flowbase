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

// Failf fails with a message that includes the process name
func (p *BaseProcess) Failf(msg string, parts ...interface{}) {
	p.Fail(fmt.Sprintf(msg, parts...))
}

// Fail fails with a message that includes the process name
func (p *BaseProcess) Fail(msg interface{}) {
	Failf("[Process:%s] %s", p.Name(), msg)
}
