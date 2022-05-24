package flowbase

type Sink struct {
	Node
	inPorts []chan interface{}
}

// Instantiate a Sink component
func NewSink() *Sink {
	return &Sink{
		inPorts: []chan interface{}{},
	}
}

func (node *Sink) Connect(ch chan interface{}) {
	node.inPorts = append(node.inPorts, ch)
}

// Execute the Sink component
func (node *Sink) Run() {
	ok := true
	Debug.Printf("Length of inPorts: %d\n", len(node.inPorts))
	for len(node.inPorts) > 0 {
		for i, ich := range node.inPorts {
			select {
			case _, ok = <-ich:
				Debug.Printf("Received on in-port %d in sink\n", i)
				if !ok {
					Debug.Printf("Port on  %d not ok, in sink\n", i)
					node.deleteInPortAtKey(i)
					continue
				}
			default:
			}
		}
	}
}

func (node *Sink) deleteInPortAtKey(i int) {
	Debug.Println("Deleting inport at key", i, "in sink")
	node.inPorts = append(node.inPorts[:i], node.inPorts[i+1:]...)
}

type SinkString struct {
	Node
	inPorts []chan string
}

// Instantiate a SinkString component
func NewSinkString() (s *SinkString) {
	return &SinkString{
		inPorts: []chan string{},
	}
}

func (node *SinkString) Connect(ch chan string) {
	node.inPorts = append(node.inPorts, ch)
}

// Execute the SinkString component
func (node *SinkString) Run() {
	for len(node.inPorts) > 0 {
		for i, ich := range node.inPorts {
			select {
			case str, ok := <-ich:
				Debug.Printf("Received string in sink: %s\n", str)
				if !ok {
					Debug.Println("Port was not ok!")
					node.deleteInPortAtKey(i)
					continue
				}
			default:
			}
		}
	}
}

func (node *SinkString) deleteInPortAtKey(i int) {
	node.inPorts = append(node.inPorts[:i], node.inPorts[i+1:]...)
}
