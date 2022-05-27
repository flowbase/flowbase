package flowbase

import (
	"fmt"
	"sync"
)

// ------------------------------------------------------------------------
// InPort
// ------------------------------------------------------------------------

// InPort represents a pluggable connection to multiple out-ports from other
// processes, from its own process, and with which it is communicating via
// channels under the hood
type InPort struct {
	Chan        chan *FileIP
	name        string
	process     NetworkProcess
	RemotePorts map[string]*OutPort
	ready       bool
	closeLock   sync.Mutex
}

// NewInPort returns a new InPort struct
func NewInPort(name string) *InPort {
	inp := &InPort{
		name:        name,
		RemotePorts: map[string]*OutPort{},
		Chan:        make(chan *FileIP, getBufsize()), // This one will contain merged inputs from inChans
		ready:       false,
	}
	return inp
}

// Name returns the name of the InPort
func (pt *InPort) Name() string {
	return pt.Process().Name() + "." + pt.name
}

// Process returns the process connected to the port
func (pt *InPort) Process() NetworkProcess {
	if pt.process == nil {
		pt.Fail("No connected process!")
	}
	return pt.process
}

// SetProcess sets the process of the port to p
func (pt *InPort) SetProcess(p NetworkProcess) {
	pt.process = p
}

// AddRemotePort adds a remote OutPort to the InPort
func (pt *InPort) AddRemotePort(rpt *OutPort) {
	if pt.RemotePorts[rpt.Name()] != nil {
		pt.Failf("A remote port with name (%s) already exists", rpt.Name())
	}
	pt.RemotePorts[rpt.Name()] = rpt
}

// From connects an OutPort to the InPort
func (pt *InPort) From(rpt *OutPort) {
	pt.AddRemotePort(rpt)
	rpt.AddRemotePort(pt)

	pt.SetReady(true)
	rpt.SetReady(true)
}

// Disconnect disconnects the (out-)port with name rptName, from the InPort
func (pt *InPort) Disconnect(rptName string) {
	pt.removeRemotePort(rptName)
	if len(pt.RemotePorts) == 0 {
		pt.SetReady(false)
	}
}

// removeRemotePort removes the (out-)port with name rptName, from the InPort
func (pt *InPort) removeRemotePort(rptName string) {
	if _, ok := pt.RemotePorts[rptName]; !ok {
		pt.Failf("No remote port with name (%s) exists", rptName)
	}
	delete(pt.RemotePorts, rptName)
}

// SetReady sets the ready status of the InPort
func (pt *InPort) SetReady(ready bool) {
	pt.ready = ready
}

// Ready tells whether the port is ready or not
func (pt *InPort) Ready() bool {
	return pt.ready
}

// Send sends IPs to the in-port, and is supposed to be called from the remote
// (out-) port, to send to this in-port
func (pt *InPort) Send(ip *FileIP) {
	pt.Chan <- ip
}

// Recv receives IPs from the port
func (pt *InPort) Recv() *FileIP {
	return <-pt.Chan
}

// CloseConnection closes the connection to the remote out-port with name
// rptName, on the InPort
func (pt *InPort) CloseConnection(rptName string) {
	pt.closeLock.Lock()
	delete(pt.RemotePorts, rptName)
	if len(pt.RemotePorts) == 0 {
		close(pt.Chan)
	}
	pt.closeLock.Unlock()
}

// Failf fails with a message that includes the process name
func (pt *InPort) Failf(msg string, parts ...interface{}) {
	pt.Fail(fmt.Sprintf(msg, parts...))
}

// Fail fails with a message that includes the process name
func (pt *InPort) Fail(msg interface{}) {
	Failf("[In-Port:%s] %s", pt.Name(), msg)
}

// ------------------------------------------------------------------------
// OutPort
// ------------------------------------------------------------------------

// OutPort represents a pluggable connection to multiple in-ports from other
// processes, from its own process, and with which it is communicating via
// channels under the hood
type OutPort struct {
	name        string
	process     NetworkProcess
	RemotePorts map[string]*InPort
	ready       bool
}

// NewOutPort returns a new OutPort struct
func NewOutPort(name string) *OutPort {
	outp := &OutPort{
		name:        name,
		RemotePorts: map[string]*InPort{},
		ready:       false,
	}
	return outp
}

// Name returns the name of the OutPort
func (pt *OutPort) Name() string {
	return pt.Process().Name() + "." + pt.name
}

// Process returns the process connected to the port
func (pt *OutPort) Process() NetworkProcess {
	if pt.process == nil {
		pt.Fail("No connected process!")
	}
	return pt.process
}

// SetProcess sets the process of the port to p
func (pt *OutPort) SetProcess(p NetworkProcess) {
	pt.process = p
}

// AddRemotePort adds a remote InPort to the OutPort
func (pt *OutPort) AddRemotePort(rpt *InPort) {
	if _, ok := pt.RemotePorts[rpt.Name()]; ok {
		pt.Failf("A remote port with name (%s) already exists", rpt.Name())
	}
	pt.RemotePorts[rpt.Name()] = rpt
}

// removeRemotePort removes the (in-)port with name rptName, from the OutPort
func (pt *OutPort) removeRemotePort(rptName string) {
	if _, ok := pt.RemotePorts[rptName]; !ok {
		pt.Failf("No remote port with name (%s) exists", rptName)
	}
	delete(pt.RemotePorts, rptName)
}

// To connects an InPort to the OutPort
func (pt *OutPort) To(rpt *InPort) {
	pt.AddRemotePort(rpt)
	rpt.AddRemotePort(pt)

	pt.SetReady(true)
	rpt.SetReady(true)
}

// Disconnect disconnects the (in-)port with name rptName, from the OutPort
func (pt *OutPort) Disconnect(rptName string) {
	pt.removeRemotePort(rptName)
	if len(pt.RemotePorts) == 0 {
		pt.SetReady(false)
	}
}

// SetReady sets the ready status of the OutPort
func (pt *OutPort) SetReady(ready bool) {
	pt.ready = ready
}

// Ready tells whether the port is ready or not
func (pt *OutPort) Ready() bool {
	return pt.ready
}

// Send sends an FileIP to all the in-ports connected to the OutPort
func (pt *OutPort) Send(ip *FileIP) {
	for _, rpt := range pt.RemotePorts {
		Debug.Printf("Sending on out-port (%s) connected to in-port (%s)", pt.Name(), rpt.Name())
		rpt.Send(ip)
	}
}

// Close closes the connection between this port and all the ports it is
// connected to. If this port is the last connected port to an in-port, that
// in-ports channel will also be closed.
func (pt *OutPort) Close() {
	for _, rpt := range pt.RemotePorts {
		Debug.Printf("Closing out-port (%s) connected to in-port (%s)", pt.Name(), rpt.Name())
		rpt.CloseConnection(pt.Name())
		pt.removeRemotePort(rpt.Name())
	}
}

// Failf fails with a message that includes the process name
func (pt *OutPort) Failf(msg string, parts ...interface{}) {
	pt.Fail(fmt.Sprintf(msg, parts...))
}

// Fail fails with a message that includes the process name
func (pt *OutPort) Fail(msg interface{}) {
	Failf("[Out-Port:%s] %s", pt.Name(), msg)
}
