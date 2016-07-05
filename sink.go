package flowbase

type Sink struct {
	Process
	inPorts []chan *interface{}
}

// Instantiate a Sink component
func NewSink() (s *Sink) {
	return &Sink{
		inPorts: make([]chan *interface{}, BUFSIZE),
	}
}

func (proc *Sink) Connect(ch chan *interface{}) {
	proc.inPorts = append(proc.inPorts, ch)
}

// Execute the Sink component
func (proc *Sink) Run() {
	ok := true
	for len(proc.inPorts) > 0 {
		for i, ich := range proc.inPorts {
			select {
			case _, ok = <-ich:
				if !ok {
					proc.deleteInPortAtKey(i)
					continue
				}
			default:
			}
		}
	}
}

func (proc *Sink) deleteInPortAtKey(i int) {
	proc.inPorts = append(proc.inPorts[:i], proc.inPorts[i+1:]...)
}

type SinkString struct {
	Process
	inPorts []chan string
}

// Instantiate a SinkString component
func NewSinkString() (s *SinkString) {
	return &SinkString{}
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
