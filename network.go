package flowbase

import (
	"fmt"
	"os"
	"reflect"
)

type Network struct {
	nodes []Node
}

func NewNetwork() *Network {
	return &Network{}
}

func (net *Network) AddNode(node Node) {
	net.nodes = append(net.nodes, node)
}

func (net *Network) AddNodes(nodes ...Node) {
	for _, node := range nodes {
		net.AddNode(node)
	}
}

func (net *Network) PrintProcesses() {
	for i, node := range net.nodes {
		fmt.Printf("Node %d: %v\n", i, reflect.TypeOf(node))
	}
}

func (net *Network) Run() {
	if !LogExists {
		InitLogAudit()
	}
	if len(net.nodes) == 0 {
		Error.Println("Network: The Network is empty. Did you forget to add the nodes to it?")
		os.Exit(1)
	}
	for i, node := range net.nodes {
		if i < len(net.nodes)-1 {
			Debug.Printf("Network: Starting node %d of type %v: in new go-routine...\n", i, reflect.TypeOf(node))
			go node.Run()
		} else {
			Debug.Printf("Network: Starting node %d of type %v: in main go-routine...\n", i, reflect.TypeOf(node))
			node.Run()
		}
	}
}
