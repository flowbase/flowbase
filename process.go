package flowbase

// Interface for all constituents of flow networks, including processes,
// networks and sub-networks
type Node interface {
	Name() string
	Ready() bool
	Run()
}
