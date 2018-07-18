package adapter

import (
	"strconv"
	"fmt"
)

func copyByteArray(data []byte) []byte {
	result := make([]byte, len(data))
	copy(result, data)

	return result
}

func copyArrayOfByteArrays(data [][]byte) [][]byte {
	result := make([][]byte, len(data))
	for i := range data {
		result[i] = make([]byte, len(data[i]))
		copy(result[i], data[i])
	}

	return result
}

func formatInt(i int) string {
	return strconv.FormatInt(int64(i), 10)
}

func anyErrors(errors []error) bool {
	for _, error := range errors {
		if error != nil {
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