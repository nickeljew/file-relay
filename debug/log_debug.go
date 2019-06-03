//+build debug

package debug

import (
	"fmt"
)

const Dev = true

func Debug(a ...interface{}) {
	fmt.Println(a...)
}

func Debugf(s string, a ...interface{}) {
	fmt.Printf(s, a...)
	fmt.Println()
}
