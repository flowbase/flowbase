package main

import (
	"fmt"

	fb "github.com/flowbase/flowbase"
)

func main() {
	// Init network
	net := fb.NewNetwork("net")

	// Init processes
	sc := NewStringCreator(net, "string-creator")
	sp := NewStringPrinter(net, "string-printer")
	net.AddProcs(sc, sp)

	// Connect network
	sp.In().From(sc.Out())

	// Run
	net.Run()
}

// ----------------------------------------------------------------------------
// StringCreator
// ----------------------------------------------------------------------------

func NewStringCreator(net *fb.Network, name string) *StringCreator {
	p := &StringCreator{
		fb.NewBaseProcess(net, name),
	}
	p.InitOutPort(p, name+"-out")
	return p
}

type StringCreator struct {
	fb.BaseProcess
}

func (n *StringCreator) Out() *fb.OutPort {
	return n.OutPort(n.Name() + "-out")
}

func (n *StringCreator) Run() {
	defer n.CloseOutPorts()
	for _, s := range []string{"abc", "cde", "xyz"} {
		n.Out().Send(s)
	}
}

// ----------------------------------------------------------------------------
// Printer
// ----------------------------------------------------------------------------

func NewStringPrinter(net *fb.Network, name string) *StringPrinter {
	p := &StringPrinter{
		fb.NewBaseProcess(net, name),
	}
	p.InitInPort(p, name+"-in")
	return p
}

type StringPrinter struct {
	fb.BaseProcess
}

func (n *StringPrinter) In() *fb.InPort {
	return n.InPort(n.Name() + "-in")
}

func (n *StringPrinter) Run() {
	for ip := range n.In().Chan {
		fmt.Println("Got string: ", ip.Data())
	}
}
