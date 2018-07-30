package adapter

import (
	"fmt"
	"strconv"
)

func formatInt(i int) string {
	return strconv.FormatInt(int64(i), 10)
}

func anyErrors(errors []error) bool {
	for _, error := range errors {
		if error != nil {
			// FIXME report all errors to the log
			fmt.Println("Found error", errors)
			return true
		}
	}

	return false
}

func anyConditions(bools []bool) bool {
	for _, val := range bools {
		if val == false {
			return false
		}
	}

	return true
}
