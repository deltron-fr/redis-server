package parser

import (
	"fmt"
	"strconv"
	"strings"
)

func BulkStringInputParser(data string) ([]string, error) {
	trimmedStr := strings.TrimRight(data, "\r\n")
	lines := strings.Split(trimmedStr, "\r\n")

	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid Bulk String format")
	}

	stringLength := lines[0][1:]
	stringLen, err := strconv.Atoi(stringLength)
	if err != nil {
		return nil, fmt.Errorf("invalid string length: %v", err)
	}

	if stringLen != len(lines[1]) {
		return nil, fmt.Errorf("string length mismatch: expected %d, got %d", stringLen, len(lines[1]))
	}

	return []string{lines[1]}, nil
}

func BulkStringOutputParser(data string) string {
	length := len(data)
	return fmt.Sprintf("$%d\r\n%s\r\n", length, data)
}
