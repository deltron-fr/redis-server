package parser

import "fmt"

func SimpleStringOutputParser(data string) string {
	return fmt.Sprintf("+%s\r\n", data)
}
