package flowbase

// Base interface for all processes
type Process interface {
	Ready() bool
	Run()
}
