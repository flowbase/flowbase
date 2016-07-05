package flowbase

// Base interface for all processes
type Process interface {
	IsConnected() bool
	Run()
}
