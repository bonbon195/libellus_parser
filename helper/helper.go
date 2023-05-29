package helper

import (
	"strings"
)

func First[T, U any](val T, _ U) T {
	return val
}

func IsNotEmpty(s string) bool {
	return len(strings.ReplaceAll(s, " ", "")) != 0
}
