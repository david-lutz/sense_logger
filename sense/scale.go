package sense

import "fmt"

// Scale "enum"
type Scale int

// Scale "enum" values
const (
	Hour Scale = iota
	Day
	Week
	Month
	Year
)

var scales = [...]string{
	"HOUR",
	"DAY",
	"WEEK",
	"MONTH",
	"YEAR",
}

func (s Scale) String() string {
	return scales[s]
}

// ParseScale parses a string to a Scale type
func ParseScale(str string) (Scale, error) {
	for i, s := range scales {
		if str == s {
			return Scale(i), nil
		}
	}
	return 0, fmt.Errorf("Invalid Scale: %s", str)
}
