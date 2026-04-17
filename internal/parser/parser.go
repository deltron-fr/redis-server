package parser

import "fmt"

// RespType defines the structure for RESP type information, including the name of the type and a parser function to handle input of that type.
type RespType struct {
	Name   string
	Parser func(string) ([]string, error)
}

// respType maps RESP type identifiers to their corresponding names and parsing functions.
var respType = map[byte]RespType{
	'$': {Name: "Bulk String", Parser: BulkStringInputParser},
	'*': {Name: "Array", Parser: ArrayInputParser},
}

// Parse dispatches RESP data to the parser registered for its type identifier.
func Parse(data []byte) ([]string, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty RESP input")
	}

	firstChar := data[0]
	resp, ok := respType[firstChar]
	if !ok {
		return nil, fmt.Errorf("unsupported RESP type: %c", firstChar)
	}

	res, err := resp.Parser(string(data))
	if err != nil {
		return nil, fmt.Errorf("error parsing RESP data: %v", err)
	}
	return res, nil
}
