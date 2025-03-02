package asserts

import "fmt"

func checkEqual(expected, actual any) {
	if expected != actual {
		panic(fmt.Sprintf("expected %v, got %v", expected, actual))
	}
}
