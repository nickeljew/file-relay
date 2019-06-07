//+build !debug

package debug

const Dev = false

func Debug(a ...interface{}) {}
func Debugf(a ...interface{}) {}


//
func ValFrom(deb, rel interface{}) interface{} {
	return rel
}
