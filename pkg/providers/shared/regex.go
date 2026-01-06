package shared

import (
	"fmt"
	"regexp"
)

// ParseFirstFloat finds the first float match in the string using the provided regex.
// The regex must have at least one capture group.
func ParseFirstFloat(re *regexp.Regexp, s string) float64 {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0
	}
	var v float64
	fmt.Sscanf(m[1], "%f", &v)
	return v
}
