package scipipe

import (
	"strings"
	"testing"
)

func TestExecCmd_EchoFooBar(t *testing.T) {
	output := ExecCmd("echo foo bar")
	output = strings.TrimSpace(strings.TrimSuffix(output, "\n"))
	if output != "foo bar" {
		t.Errorf("output = %swant: foo bar\n", output)
	}
}

func TestRegexPatternMatchesCases(t *testing.T) {
	r := getShellCommandPlaceHolderRegex()
	placeHolders := []string{
		"{i:hej}",
		"{is:hej}",
		"{o:hej}",
		"{o:hej|.txt}",
		"{os:hej}",
		"{o:hej|%.txt}",
		"{i:hej|%.txt}",
		"{i:hej|join}",
		"{i:hej|join: }",
		"{i:hej|join:,}",
	}
	for _, ph := range placeHolders {
		if !r.Match([]byte(ph)) {
			t.Errorf("Error does not match placeholder: %s\n", ph)
		}
	}
}
