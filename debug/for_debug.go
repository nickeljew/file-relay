//+build debug

package debug

import (
	"fmt"
	"math"
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
	style Style
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

	dt.setStyle(namespace)
	return dt
}

func (dt *DTrace) setStyle(namespace string) {
	chars := []byte(namespace)
	var hash byte
	for _, c := range chars {
		hash = ((hash << 5) - hash) + c
	}
	c := int( math.Abs(float64(hash)) ) % len(FgColors)
	dt.style = Style{FgColors[c], OpBold}
}

func (dt *DTrace) Log(a ...interface{}) {
	if !dt.enabled {
		return
	}
	s := dt.style.Render(dt.ns) + " " + fmt.Sprintln(a...)
	fmt.Printf(s)
}

func (dt *DTrace) Logf(s string, a ...interface{}) {
	if !dt.enabled {
		return
	}
	s = dt.style.Render(dt.ns) + " " + s
	fmt.Printf(s + "\n", a...)
}



//
func ValFrom(deb, rel interface{}) interface{} {
	return deb
}
