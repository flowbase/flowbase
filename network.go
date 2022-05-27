package flowbase

import (
	"fmt"
	"reflect"
)

type Network struct {
	name   string
	nodes  map[string]Node
	sink   *Sink
	driver Node
}

func NewNetwork() *Network {
	return &Network{}
}

func (n *Network) Name() string {
	return n.name
}

func (net *Network) AddNode(node Node) {
	net.nodes[node.Name()] = node
}

func (net *Network) AddNodes(nodes ...Node) {
	for _, node := range nodes {
		net.AddNode(node)
	}
}

func (net *Network) PrintNodes() {
	for i, node := range net.nodes {
		fmt.Printf("Node %d: %v\n", i, reflect.TypeOf(node))
	}
}

// Run runs all the nodes of the Network
func (net *Network) Run() {
	net.runNodes(net.nodes)
}

// runNodes runs a specified set of nodes only
func (net *Network) runNodes(nodes map[string]Node) {
	net.reconnectDeadEndConnections(nodes)

	if !net.readyToRun(nodes) {
		net.Fail("Network not ready to run, due to previously reported errors, so exiting.")
	}

	for _, node := range nodes {
		Debug.Printf(net.Name()+": Starting node (%s) in new go-routine", node.Name())
		go node.Run()
	}

	Debug.Printf("%s: Starting driver node (%s) in main go-routine", net.name, net.driver.Name())
	net.driver.Run()
}

func (net *Network) reconnectDeadEndConnections(nodes map[string]Node) {
	foundNewDriverProc := false

	for _, node := range nodes {
		// OutPorts
		for _, opt := range node.OutPorts() {
			for iptName, ipt := range opt.RemotePorts() {
				// If the remotely connected node is not among the ones to run ...
				if ipt.Node() == nil {
					Debug.Printf("Disconnecting in-port (%s) from out-port (%s)", ipt.Name(), opt.Name())
					opt.Disconnect(iptName)
				} else if _, ok := nodes[ipt.Node().Name()]; !ok {
					Debug.Printf("Disconnecting in-port (%s) from out-port (%s)", ipt.Name(), opt.Name())
					opt.Disconnect(iptName)
				}
			}
			if !opt.Ready() {
				Debug.Printf("Connecting disconnected out-port (%s) of node (%s) to network sink", opt.Name(), opt.Node().Name())
				net.sink.From(opt)
			}
		}

		if len(node.OutPorts()) == 0 {
			if foundNewDriverProc {
				net.Failf("Found more than one node without out-ports nor out-param ports. Cannot use both as drivers (One of them being '%s'). Adapt your network accordingly.", node.Name())
			}
			foundNewDriverProc = true
			net.driver = node
		}
	}

	if foundNewDriverProc && len(nodes) > 1 { // Allow for a network with a single node
		// A node can't both be the driver and be included in the main nodes
		// map, so if we have an alerative driver, it should not be in the main
		// nodes map
		delete(net.nodes, net.driver.Name())
	}
}

func (wf *Network) readyToRun(nodes map[string]Node) bool {
	if len(nodes) == 0 {
		Error.Println(wf.name + ": The network is empty. Did you forget to add the processes to it?")
		return false
	}
	if wf.sink == nil {
		Error.Println(wf.name + ": sink is nil!")
		return false
	}
	for _, proc := range nodes {
		if !proc.Ready() {
			Error.Println(wf.name + ": Not everything connected. Network shutting down.")
			return false
		}
	}
	return true
}

func (net *Network) Failf(msg string, parts ...interface{}) {
	net.Fail(fmt.Sprintf(msg, parts...))
}

func (net *Network) Fail(msg interface{}) {
	Failf("[Network:%s] %s", net.Name(), msg)
}
