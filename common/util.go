package common

import (
	"strconv"
)

// PanicIf panic if error
func PanicIf(err error) {
	if err != nil {
		panic(err)
	}
}

// ParseInt string -> int
func ParseInt(value string) int {
	if value == "" {
		return 0
	}
	val, _ := strconv.Atoi(value)
	return val
}

// IntString int -> string
func IntString(value int) string {
	return strconv.Itoa(value)
}
