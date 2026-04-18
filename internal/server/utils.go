package server

import (
	"strconv"
)

func parseIndex(indexStr string) (int, error) {
	idx, err := strconv.Atoi(indexStr)
	if err != nil {
		return 0, nil
	}

	return idx, nil
}
