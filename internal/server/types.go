package server

import (
	"sync/atomic"
	"time"
)

type Role int

const (
	Master Role = iota
	Slave
)

func (r Role) String() string {
	switch r {
	case Master:
		return "master"
	case Slave:
		return "slave"
	default:
		return ""
	}
}

type ReplicationInfo struct {
	Role             Role
	MasterHost       string           // for replica
	MasterPort       string           // for replica
	MasterReplID     string           // for master
	MasterReplOffset int              // for master
	Replicas         map[*Client]bool // for master
}

type ValueStore struct {
	Value  string
	Expiry *time.Time
}

type Waiter struct {
	Ch      chan struct{}
	Expired atomic.Bool
}

type StreamEntry struct {
	ID     string
	Fields map[string]string
}
