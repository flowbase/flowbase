package flowbase

import (
	"fmt"
	"os"
)

// Fail logs the error message, so that it will be possible to improve error
// messages in one place
func Fail(vs ...interface{}) {
	Error.Println(vs...)
	//Error.Println("Printing stack trace (read from bottom to find the workflow code that hit this error):")
	//debug.PrintStack()
	os.Exit(1) // Indicates a "general error" (See http://www.tldp.org/LDP/abs/html/exitcodes.html)
}

// Failf is like Fail but with msg being a formatter string for the message and
// vs being items to format into the message
func Failf(msg string, vs ...interface{}) {
	Fail(fmt.Sprintf(msg+"\n", vs...))
}
