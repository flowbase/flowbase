package flowbase

import (
	"fmt"
	"sync"
)

// --------------------------------------------------------------------------------
// InPort
// --------------------------------------------------------------------------------

// InPort represents a pluggable connection to multiple out-ports from other
// processes, from its own process, and with which it is communicating via
// channels under the hood
type InPort[T any] struct {
	Chan        chan T
	name        string
	process     Process
	RemotePorts map[string]*OutPort[T]
	ready       bool
	closeLock   sync.Mutex
}

// NewInPort returns a new InPort struct
func NewInPort[T any](name string) *InPort[T] {
	inp := &InPort[T]{
		name:        name,
		RemotePorts: map[string]*OutPort[T]{},
		Chan:        make(chan T, 16), // This one will contain merged inputs from inChans
		ready:       false,
	}
	return inp
}

func (pt *InPort[T]) Name() string {
	return pt.name
}

// AddRemotePort adds a remote OutPort to the InPort
func (pt *InPort[T]) AddRemotePort(rpt *OutPort[T]) {
	if pt.RemotePorts[rpt.Name()] != nil {
		pt.Failf("A remote port with name (%s) already exists", rpt.Name())
	}
	pt.RemotePorts[rpt.Name()] = rpt
}

// From connects an OutPort to the InPort
func (pt *InPort[T]) From(rpt *OutPort[T]) {
	pt.AddRemotePort(rpt)
	rpt.AddRemotePort(pt)

	pt.SetReady(true)
	rpt.SetReady(true)
}

// Disconnect disconnects the (out-)port with name rptName, from the InPort
func (pt *InPort[T]) Disconnect(rptName string) {
	pt.removeRemotePort(rptName)
	if len(pt.RemotePorts) == 0 {
		pt.SetReady(false)
	}
}

// removeRemotePort removes the (out-)port with name rptName, from the InPort
func (pt *InPort[T]) removeRemotePort(rptName string) {
	if _, ok := pt.RemotePorts[rptName]; !ok {
		pt.Failf("No remote port with name (%s) exists", rptName)
	}
	delete(pt.RemotePorts, rptName)
}

// SetReady sets the ready status of the InPort
func (pt *InPort[T]) SetReady(ready bool) {
	pt.ready = ready
}

// Ready tells whether the port is ready or not
func (pt *InPort[T]) Ready() bool {
	return pt.ready
}

// Send sends IPs to the in-port, and is supposed to be called from the remote
// (out-) port, to send to this in-port
func (pt *InPort[T]) Send(ip T) {
	pt.Chan <- ip
}

// Recv receives IPs from the port
func (pt *InPort[T]) Recv() T {
	return <-pt.Chan
}

// CloseConnection closes the connection to the remote out-port with name
// rptName, on the InPort
func (pt *InPort[T]) CloseConnection(rptName string) {
	pt.closeLock.Lock()
	delete(pt.RemotePorts, rptName)
	if len(pt.RemotePorts) == 0 {
		close(pt.Chan)
	}
	pt.closeLock.Unlock()
}

// Failf fails with a message that includes the process name
func (pt *InPort[T]) Failf(msg string, parts ...interface{}) {
	pt.Fail(fmt.Sprintf(msg, parts...))
}

// Fail fails with a message that includes the process name
func (pt *InPort[T]) Fail(msg interface{}) {
	Failf("[In-Port:%s] %s", pt.Name(), msg)
}

// --------------------------------------------------------------------------------
// OutPort
// --------------------------------------------------------------------------------

// OutPort represents a pluggable connection to multiple in-ports from other
// processes, from its own process, and with which it is communicating via
// channels under the hood
type OutPort[T any] struct {
	name        string
	process     Process
	RemotePorts map[string]*InPort[T]
	ready       bool
}

// NewOutPort returns a new OutPort struct
func NewOutPort[T any](name string) *OutPort[T] {
	outp := &OutPort[T]{
		name:        name,
		RemotePorts: map[string]*InPort[T]{},
		ready:       false,
	}
	return outp
}

func (pt *OutPort[T]) Name() string {
	return pt.name
}

// AddRemotePort adds a remote InPort to the OutPort
func (pt *OutPort[T]) AddRemotePort(rpt *InPort[T]) {
	if _, ok := pt.RemotePorts[rpt.Name()]; ok {
		pt.Failf("A remote port with name (%s) already exists", rpt.Name())
	}
	pt.RemotePorts[rpt.Name()] = rpt
}

// removeRemotePort removes the (in-)port with name rptName, from the OutPort
func (pt *OutPort[T]) removeRemotePort(rptName string) {
	if _, ok := pt.RemotePorts[rptName]; !ok {
		pt.Failf("No remote port with name (%s) exists", rptName)
	}
	delete(pt.RemotePorts, rptName)
}

// To connects an InPort to the OutPort
func (pt *OutPort[T]) To(rpt *InPort[T]) {
	pt.AddRemotePort(rpt)
	rpt.AddRemotePort(pt)

	pt.SetReady(true)
	rpt.SetReady(true)
}

// Disconnect disconnects the (in-)port with name rptName, from the OutPort
func (pt *OutPort[T]) Disconnect(rptName string) {
	pt.removeRemotePort(rptName)
	if len(pt.RemotePorts) == 0 {
		pt.SetReady(false)
	}
}

// SetReady sets the ready status of the OutPort
func (pt *OutPort[T]) SetReady(ready bool) {
	pt.ready = ready
}

// Ready tells whether the port is ready or not
func (pt *OutPort[T]) Ready() bool {
	return pt.ready
}

// Send sends an IP to all the in-ports connected to the OutPort
func (pt *OutPort[T]) Send(ip T) {
	for _, rpt := range pt.RemotePorts {
		Debug.Printf("Sending on out-port (%s) connected to in-port (%s)", pt.Name(), rpt.Name())
		rpt.Send(ip)
	}
}

// Close closes the connection between this port and all the ports it is
// connected to. If this port is the last connected port to an in-port, that
// in-ports channel will also be closed.
func (pt *OutPort[T]) Close() {
	for _, rpt := range pt.RemotePorts {
		Debug.Printf("Closing out-port (%s) connected to in-port (%s)", pt.Name(), rpt.Name())
		rpt.CloseConnection(pt.Name())
		pt.removeRemotePort(rpt.Name())
	}
}

// Failf fails with a message that includes the process name
func (pt *OutPort[T]) Failf(msg string, parts ...interface{}) {
	pt.Fail(fmt.Sprintf(msg, parts...))
}

// Fail fails with a message that includes the process name
func (pt *OutPort[T]) Fail(msg interface{}) {
	Failf("[Out-Port:%s] %s", pt.Name(), msg)
}
