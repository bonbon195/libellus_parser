package helper

import (
	"strings"
)

func First[T, U any](val T, _ U) T {
	return val
}

// func HandleErr(err error, str ...string) {
// 	if err != nil {
// 		log.Fatalf(str[0]+" %v", err)
// 	}
// }

func IsNotEmpty(s string) bool {
	return len(strings.ReplaceAll(s, " ", "")) != 0
}
