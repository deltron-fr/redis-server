package server

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/deltron-fr/redis-server/internal/parser"
)

var (
	ErrXADDIDTooSmall = errors.New("the ID specified in XADD is equal or smaller than the target stream top item")
	ErrXADDIDZero     = errors.New("the ID specified in XADD must be greater than 0-0")
)

func (s *Server) xaddHandler(cmd Command) (string, error) {
	if len(cmd.Args) < 4 {
		return "", fmt.Errorf("XADD requires atleast four arguments")
	}

	if len(cmd.Args)%2 != 0 {
		return "", fmt.Errorf("invalid number of arguments")
	}

	key := cmd.Args[0]
	id := cmd.Args[1]

	idParts := strings.Split(id, "-")

	if len(idParts) != 2 && id != "*" {
		return "", fmt.Errorf("invalid id, should have format: <millisecondstime>-<sequencenumber>")
	}

	// TODO: check if the two millisecondstime(input and prevID) are of equal length before
	// string(lexicographic) comparison

	var millisecondsTime, seqNo string
	if len(idParts) == 2 {
		millisecondsTime, seqNo = idParts[0], idParts[1]
		if millisecondsTime == "0" && seqNo == "0" {
			return "", ErrXADDIDZero
		}
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	newID, err := s.handleEntryID(id, key, millisecondsTime, seqNo)
	if err != nil {
		return "", err
	}

	entry := StreamEntry{
		ID:     newID,
		Fields: make(map[string]string),
	}

	for i := 2; i < len(cmd.Args); i += 2 {
		entry.Fields[cmd.Args[i]] = cmd.Args[i+1]
	}

	s.StreamStore[key] = append(s.StreamStore[key], entry)

	return parser.BulkStringOutputParser(newID), nil
}

func (s *Server) handleEntryID(id, key, millisecondsTime, seqNo string) (string, error) {
	if id == "*" {
		id, err := s.handleFullEntryID(id, key, millisecondsTime, seqNo)
		if err != nil {
			return "", err
		}
		return id, nil
	}

	if millisecondsTime != "*" && seqNo != "*" {
		id, err := s.handleManualEntryID(id, key, millisecondsTime, seqNo)
		if err != nil {
			return "", err
		}

		return id, nil
	}

	if millisecondsTime != "*" && seqNo == "*" {
		id, err := s.handlePartialEntryID(id, key, millisecondsTime, seqNo)
		if err != nil {
			return "", err
		}
		return id, nil
	}

	return "", errors.New("invalid input")
}

func (s *Server) handleManualEntryID(id, key, millisecondsTime, seqNo string) (string, error) {
	v, ok := s.StreamStore[key]
	if ok {
		length := len(v)
		lastEntry := v[length-1]

		lastEntryIDparts := strings.Split(lastEntry.ID, "-")
		millisecondsTimeLastEntry, seqNoLastEntry := lastEntryIDparts[0], lastEntryIDparts[1]

		if millisecondsTimeLastEntry > millisecondsTime {
			return "", ErrXADDIDTooSmall
		}

		if millisecondsTimeLastEntry == millisecondsTime {
			if seqNoLastEntry >= seqNo {
				return "", ErrXADDIDTooSmall
			}
		}
	}
	return id, nil
}

func (s *Server) handlePartialEntryID(id, key, millisecondsTime, seqNo string) (string, error) {
	if millisecondsTime == "0" {
		return fmt.Sprintf("%s-%d", millisecondsTime, 1), nil
	}

	v, ok := s.StreamStore[key]
	if !ok {
		return fmt.Sprintf("%s-%d", millisecondsTime, 0), nil
	}

	topEntry := v[len(v)-1]
	lastEntryIDparts := strings.Split(topEntry.ID, "-")
	millisecondsTimeLastEntry, _ := strconv.ParseInt(lastEntryIDparts[0], 10, 64)
	seqNoLastEntry, _ := strconv.ParseInt(lastEntryIDparts[1], 10, 64)

	requestedMs, err := strconv.ParseInt(millisecondsTime, 10, 64)
	if err != nil {
		return "", err
	}

	if requestedMs < millisecondsTimeLastEntry {
		return "", ErrXADDIDTooSmall
	}

	if requestedMs == millisecondsTimeLastEntry {
		return fmt.Sprintf("%d-%d", requestedMs, seqNoLastEntry+1), nil
	}

	return fmt.Sprintf("%d-0", requestedMs), nil
}

func (s *Server) handleFullEntryID(id, key, millisecondsTime, seqNo string) (string, error) {
	currentTime := time.Now().UnixMilli()
	currentTimeStr := strconv.Itoa(int(currentTime))

	v, ok := s.StreamStore[key]
	if !ok {
		return fmt.Sprintf("%s-%d", currentTimeStr, 0), nil
	}

	topEntry := v[len(v)-1]
	lastEntryIDparts := strings.Split(topEntry.ID, "-")
	millisecondsTimeLastEntry, _ := strconv.ParseInt(lastEntryIDparts[0], 10, 64)
	seqNoLastEntry, _ := strconv.ParseInt(lastEntryIDparts[1], 10, 64)

	if currentTime < millisecondsTimeLastEntry {
		return "", ErrXADDIDTooSmall
	}

	if currentTime == millisecondsTimeLastEntry {
		return fmt.Sprintf("%d-%d", currentTime, seqNoLastEntry+1), nil
	}

	return fmt.Sprintf("%d-0", currentTime), nil
}
