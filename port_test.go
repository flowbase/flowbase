package flowbase

import (
	"reflect"
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

func TestInParamPortSendRecv(t *testing.T) {
	initTestLogs()

	pip := NewInParamPort("test_param_inport")
	param := "foo-bar"
	go func() {
		pip.Send(param)
	}()
	outParam := pip.Recv()
	if param != outParam {
		t.Errorf("Received param (%s) was not the same as the sent one (%s)", outParam, param)
	}
}

func TestInParamPortFromStr(t *testing.T) {
	initTestLogs()

	pip := NewInParamPort("test_inport")
	pip.process = NewBogusProcess("bogus_process")

	pip.FromStr("foo", "bar", "baz")
	expectedStrs := []string{"foo", "bar", "baz"}

	outStrs := []string{}
	for s := range pip.Chan {
		outStrs = append(outStrs, s)
	}
	if !reflect.DeepEqual(outStrs, expectedStrs) {
		t.Errorf("Received strings %v are not the same as expected strings %v", outStrs, expectedStrs)
	}
}

func TestOutParamPortFrom(t *testing.T) {
	initTestLogs()

	popName := "test_param_outport"
	pop := NewOutParamPort(popName)
	pop.process = NewBogusProcess("bogus_process")

	pipName := "test_param_inport"
	pip := NewInParamPort(pipName)
	pip.process = NewBogusProcess("bogus_process")

	pop.To(pip)

	if !pop.Ready() {
		t.Errorf("Param out port '%s' not having connected status = true", pop.Name())
	}
	if !pip.Ready() {
		t.Errorf("Param out port '%s' not having connected status = true", pip.Name())
	}

	if pop.RemotePorts["bogus_process."+pipName] == nil {
		t.Errorf("InParamPort not among remote ports in OutParamPort")
	}
	if pip.RemotePorts["bogus_process."+popName] == nil {
		t.Errorf("OutParamPort not among remote ports in InParamPort")
	}
}
