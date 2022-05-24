package flowbase

import (
	"fmt"
	"os"
	"reflect"
)

type Net struct {
	nodes []Node
}

func NewNet() *Net {
	return &Net{}
}

func (net *Net) AddNode(node Node) {
	net.nodes = append(net.nodes, node)
}

func (net *Net) AddNodes(nodes ...Node) {
	for _, node := range nodes {
		net.AddNode(node)
	}
}

func (net *Net) PrintProcesses() {
	for i, node := range net.nodes {
		fmt.Printf("Node %d: %v\n", i, reflect.TypeOf(node))
	}
}

func (net *Net) Run() {
	if !LogExists {
		InitLogAudit()
	}
	if len(net.nodes) == 0 {
		Error.Println("Net: The Net is empty. Did you forget to add the nodes to it?")
		os.Exit(1)
	}
	for i, node := range net.nodes {
		if i < len(net.nodes)-1 {
			Debug.Printf("Net: Starting node %d of type %v: in new go-routine...\n", i, reflect.TypeOf(node))
			go node.Run()
		} else {
			Debug.Printf("Net: Starting node %d of type %v: in main go-routine...\n", i, reflect.TypeOf(node))
			node.Run()
		}
	}
}
