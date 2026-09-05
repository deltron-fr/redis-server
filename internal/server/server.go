package server

import (
	"strings"
	"sync"
)

type Server struct {
	Store             map[string]ValueStore
	ListStore         map[string][]string
	StreamStore       map[string][]StreamEntry
	Commands          map[string]Command
	WatchedKeys       map[string]bool
	ReplicaInfo       ReplicationInfo
	ResyncCh          chan *Client
	Mu                sync.RWMutex
	WaiterQueueList   chan *Waiter
	WaiterQueueStream chan *Waiter
}

func NewServer(replicaInput string) *Server {
	waiterQ := make(chan *Waiter, 100)
	waiterQStream := make(chan *Waiter, 100)

	var replInfo ReplicationInfo

	if replicaInput == "" {
		replInfo.Role = Master
		replInfo.MasterReplID = "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb"
		replInfo.MasterReplOffset = 0
		replInfo.Replicas = make(map[*Client]bool)
	} else {
		replInfo.Role = Slave
		parts := strings.Fields(replicaInput)
		replInfo.MasterHost, replInfo.MasterPort = parts[0], parts[1]
	}

	s := &Server{
		Store:             make(map[string]ValueStore),
		ListStore:         make(map[string][]string),
		StreamStore:       make(map[string][]StreamEntry),
		Commands:          make(map[string]Command),
		WatchedKeys:       make(map[string]bool),
		ResyncCh:          make(chan *Client, 1),
		ReplicaInfo:       replInfo,
		WaiterQueueList:   waiterQ,
		WaiterQueueStream: waiterQStream,
	}

	s.Commands["ECHO"] = Command{Handler: s.echoHandler, Type: ReadCommand}
	s.Commands["PING"] = Command{Handler: s.pingHandler, Type: ReadCommand}
	s.Commands["SET"] = Command{Handler: s.setHandler, Type: WriteCommand}
	s.Commands["GET"] = Command{Handler: s.getHandler, Type: ReadCommand}
	s.Commands["RPUSH"] = Command{Handler: s.rPushHandler, Type: WriteCommand}
	s.Commands["LPUSH"] = Command{Handler: s.lPushHandler, Type: WriteCommand}
	s.Commands["LRANGE"] = Command{Handler: s.lRangeHandler, Type: ReadCommand}
	s.Commands["LLEN"] = Command{Handler: s.lLenHandler, Type: ReadCommand}
	s.Commands["LPOP"] = Command{Handler: s.lPopHandler, Type: WriteCommand}
	s.Commands["BLPOP"] = Command{Handler: s.bLPopHandler, Type: WriteCommand}
	s.Commands["TYPE"] = Command{Handler: s.typeHandler, Type: ReadCommand}
	s.Commands["XADD"] = Command{Handler: s.xaddHandler, Type: ReadCommand}
	s.Commands["XRANGE"] = Command{Handler: s.xRangeHandler, Type: ReadCommand}
	s.Commands["XREAD"] = Command{Handler: s.xReadHandler, Type: ReadCommand}
	s.Commands["INCR"] = Command{Handler: s.incrHandler, Type: WriteCommand}
	s.Commands["MULTI"] = Command{Handler: s.multiHandler, Type: ReadCommand}
	s.Commands["EXEC"] = Command{Handler: s.execHandler, Type: ReadCommand}
	s.Commands["DISCARD"] = Command{Handler: s.discardHandler, Type: ReadCommand}
	s.Commands["WATCH"] = Command{Handler: s.watchHandler, Type: ReadCommand}
	s.Commands["UNWATCH"] = Command{Handler: s.unwatchHandler, Type: ReadCommand}
	s.Commands["INFO"] = Command{Handler: s.infoHandler, Type: ReadCommand}
	s.Commands["REPLCONF"] = Command{Handler: s.replConfHandler, Type: ReadCommand}
	s.Commands["PSYNC"] = Command{Handler: s.psyncHandler, Type: ReadCommand}

	go func() {
		s.emptyRDBTransfer()
	}()

	return s
}
