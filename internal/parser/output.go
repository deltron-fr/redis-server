package parser

import (
	"fmt"
	"strings"
)

func RESPOutputParser(data any) string {
	strData, ok := data.(string)
	if ok {
		length := len(strData)
		return fmt.Sprintf("$%d\r\n%s\r\n", length, data)
	}

	arrData, ok := data.([]string)
	if ok {
		var result strings.Builder
		for _, str := range arrData {
			output := BulkStringOutputParser(str)
			result.WriteString(output)
		}

		return fmt.Sprintf("*%d\r\n%s", len(arrData), result.String())
	}

	switch d := data.(type) {
	case []any:
		var result strings.Builder
		for _, item := range d {
			output := RESPOutputParser(item)
			result.WriteString(output)
		}

		return fmt.Sprintf("*%d\r\n%s", len(d), result.String())
	}

	return ""
}

func BulkStringOutputParser(data string) string {
	length := len(data)
	return fmt.Sprintf("$%d\r\n%s\r\n", length, data)
}

func ErrorOutputParser(data string) string {
	return fmt.Sprintf("-ERR %s\r\n", data)
}

func SimpleStringOutputParser(data string) string {
	return fmt.Sprintf("+%s\r\n", data)
}

func IntegerOutputParser(data int) string {
	return fmt.Sprintf(":%d\r\n", data)
}
