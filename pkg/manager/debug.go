package manager

import (
	"fmt"
)

// Debug controls whether debug messages are printed
var Debug = false

// Debugf prints a debug message if debug output is enabled
func Debugf(format string, args ...interface{}) {
	if Debug {
		fmt.Printf("Debug: "+format+"\n", args...)
	}
}
