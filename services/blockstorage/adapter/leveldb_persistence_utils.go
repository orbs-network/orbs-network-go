package adapter

import (
	"strconv"
)

func formatInt(i int) string {
	return strconv.FormatInt(int64(i), 10)
}

func anyErrors(errors ...error) (bool, error) {
	for _, err := range errors {
		if err != nil {
			return true, err
		}
	}

	return false, nil
}

func anyConditions(bools []bool) bool {
	for _, val := range bools {
		if val == false {
			return false
		}
	}

	return true
}
