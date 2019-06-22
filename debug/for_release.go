//+build !debug

package debug

const Dev = false


//
type DTrace struct {}

func NewDTrace(namespace string) *DTrace {
	return &DTrace{}
}

func (dt *DTrace) Log(a ...interface{}) {}

func (dt *DTrace) Logf(s string, a ...interface{}) {}

//
func ValFrom(deb, rel interface{}) interface{} {
	return rel
}
