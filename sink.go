package flowbase

type Sink struct {
	Process
	inPorts []chan interface{}
}

// Instantiate a Sink component
func NewSink() *Sink {
	return &Sink{
		inPorts: []chan interface{}{},
	}
}

func (proc *Sink) Connect(ch chan interface{}) {
	proc.inPorts = append(proc.inPorts, ch)
}

// Execute the Sink component
func (proc *Sink) Run() {
	ok := true
	Debug.Printf("Length of inPorts: %d\n", len(proc.inPorts))
	for len(proc.inPorts) > 0 {
		for i, ich := range proc.inPorts {
			select {
			case _, ok = <-ich:
				Debug.Printf("Received on in-port %d in sink\n", i)
				if !ok {
					Debug.Printf("Port on  %d not ok, in sink\n", i)
					proc.deleteInPortAtKey(i)
					continue
				}
			default:
			}
		}
	}
}

func (proc *Sink) deleteInPortAtKey(i int) {
	Debug.Println("Deleting inport at key", i, "in sink")
	proc.inPorts = append(proc.inPorts[:i], proc.inPorts[i+1:]...)
}

type SinkString struct {
	Process
	inPorts []chan string
}

// Instantiate a SinkString component
func NewSinkString() (s *SinkString) {
	return &SinkString{
		inPorts: []chan string{},
	}
}

func (proc *SinkString) Connect(ch chan string) {
	proc.inPorts = append(proc.inPorts, ch)
}

// Execute the SinkString component
func (proc *SinkString) Run() {
	for len(proc.inPorts) > 0 {
		for i, ich := range proc.inPorts {
			select {
			case str, ok := <-ich:
				Debug.Printf("Received string in sink: %s\n", str)
				if !ok {
					Debug.Println("Port was not ok!")
					proc.deleteInPortAtKey(i)
					continue
				}
			default:
			}
		}
	}
}

func (proc *SinkString) deleteInPortAtKey(i int) {
	proc.inPorts = append(proc.inPorts[:i], proc.inPorts[i+1:]...)
}
