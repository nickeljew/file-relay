//+build debug

package debug

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

const Dev = true

var (
	excludes = make([]*regexp.Regexp, 0)
	includes = make([]*regexp.Regexp, 0)
)

func init() {
	namespaces := os.Getenv("DTRACE")
	arr := strings.Split(namespaces, ",")
	for _, ns := range arr {
		ns = strings.Trim(ns, " ")
		if ns != "" {
			ns = strings.ReplaceAll(ns, "*", ".*?")
			chars := []byte(ns)
			if chars[0] == '-' {
				reg := regexp.MustCompile("^" + string(chars[1:]) + "$")
				excludes = append(excludes, reg)
			} else {
				reg := regexp.MustCompile("^" + ns + "$")
				includes = append(includes, reg)
			}
		}
	}
	//fmt.Printf("Excludes: %v\n", excludes)
	//fmt.Printf("Includes: %v\n", includes)
}



//
type DTrace struct {
	ns string
	enabled bool
}

func NewDTrace(namespace string) *DTrace {
	dt := &DTrace{
		ns: namespace,
	}
	for _, re := range excludes {
		if re.MatchString(namespace) {
			dt.enabled = false
			break
		}
	}
	for _, re := range includes {
		if re.MatchString(namespace) {
			dt.enabled = true
			break
		}
	}
	return dt
}

func (dt *DTrace) Log(a ...interface{}) {
	fmt.Println(a...)
}

func (dt *DTrace) Logf(s string, a ...interface{}) {
	if !dt.enabled {
		return
	}
	s = Style{FgGreen, OpBold}.Render(dt.ns) + " " + s
	fmt.Printf(s + "\n", a...)
}



//
func ValFrom(deb, rel interface{}) interface{} {
	return deb
}
