package flowbase

import (
	"testing"
)

func TestInPortSendRecv(t *testing.T) {
	inp := NewInPort("test_inport")
	inp.process = NewBogusProcess("bogus_process")

	ip, err := NewFileIP(".tmp/test.txt")
	Check(err)
	go func() {
		inp.Send(ip)
	}()
	oip := inp.Recv()
	if ip != oip {
		t.Errorf("Received ip (with path %s) was not the same as the one sent (with path %s)", oip.Path(), ip.Path())
	}
}
