package parser

import (
	"fmt"
	"strconv"
	"strings"
)

func ArrayInputParser(data string) ([]string, error) {
	trimmedStr := strings.TrimRight(data, "\r\n")
	lines := strings.Split(trimmedStr, "\r\n")

	numElements := lines[0][1:]
	numElems, err := strconv.Atoi(numElements)
	if err != nil {
		return nil, fmt.Errorf("invalid array length: %v", err)
	}

	gotNumElems := (len(lines) - 1) / 2

	if numElems != gotNumElems {
		return nil, fmt.Errorf("array length mismatch: expected %d, got %d", numElems, len(lines)-1)
	}

	var result []string
	var bulkStr string
	for _, str := range lines[1:] {
		if strings.HasPrefix(str, "$") {
			bulkStr = str
			continue
		}

		bulkStr = bulkStr + "\r\n" + str
		parsedStr, err := BulkStringInputParser(bulkStr)
		if err != nil {
			return nil, fmt.Errorf("error parsing array element: %v", err)
		}
		result = append(result, parsedStr[0])
	}

	return result, nil
}

func ArrayOutputParser(data []string) string {
	var result strings.Builder
	for _, str := range data {
		output := BulkStringOutputParser(str)
		result.WriteString(output)
	}

	return fmt.Sprintf("*%d\r\n%s", len(data), result.String())
}
