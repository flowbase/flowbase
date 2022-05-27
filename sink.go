package flowbase

// Sink is a simple component that just receives IPs on its In-port without
// doing anything with them. It is used to drive pipelines of processes
type Sink struct {
	BaseProcess
}

// NewSink returns a new Sink component
func NewSink(wf *Workflow, name string) *Sink {
	p := &Sink{
		BaseProcess: NewBaseProcess(wf, name),
	}
	p.InitInPort(p, "sink_in")
	return p
}

func (p *Sink) in() *InPort { return p.InPort("sink_in") }

// From connects an out-port to the sinks in-port
func (p *Sink) From(outPort *OutPort) {
	p.in().From(outPort)
}

// Run runs the Sink process
func (p *Sink) Run() {
	merged := make(chan int)
	if p.in().Ready() {
		go func() {
			for ip := range p.in().Chan {
				Debug.Printf("Got file in sink: %s\n", ip.Path())
			}
			merged <- 1
		}()
	}
	if p.in().Ready() {
		<-merged
	}
	close(merged)
	Debug.Printf("Caught up everything in sink")
}
