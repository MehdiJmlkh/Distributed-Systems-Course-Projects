// Implementation of a KeyValueServer. Students should write their code in this file.

package p0partA

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"project0/p0partA/kvstore"
)

const (
	bufferSize = 500
)

type keyValue struct {
	key   string
	value []byte
}

type valueChange struct {
	key      string
	oldValue []byte
	newValue []byte
}

type getQuery struct {
	conn net.Conn
	key  string
}

type keyValueServer struct {
	store          kvstore.KVStore
	address        string
	listener       net.Listener
	activeCount    int
	droppedCount   int
	putChan        chan keyValue
	getChan        chan getQuery
	deleteChan     chan string
	updateChan     chan valueChange
	connChan       chan int
	activeRequest  chan bool
	activeResult   chan int
	droppedRequest chan bool
	droppedResult  chan int
	cliBuf         map[string]chan []byte
	bufLock        chan bool
	close          chan bool
	closed         chan bool
}

// New creates and returns (but does not start) a new KeyValueServer.
func New(store kvstore.KVStore) KeyValueServer {
	return &keyValueServer{
		store:          store,
		address:        "",
		listener:       nil,
		activeCount:    0,
		droppedCount:   0,
		putChan:        make(chan keyValue),
		getChan:        make(chan getQuery),
		deleteChan:     make(chan string),
		updateChan:     make(chan valueChange),
		connChan:       make(chan int),
		activeRequest:  make(chan bool),
		activeResult:   make(chan int),
		droppedRequest: make(chan bool),
		droppedResult:  make(chan int),
		cliBuf:         make(map[string]chan []byte),
		bufLock:        make(chan bool, 1),
		close:          make(chan bool),
		closed:         make(chan bool),
	}
}

func (kvs *keyValueServer) Start(port int) error {
	var err error
	kvs.address = fmt.Sprintf(":%d", port)

	if kvs.listener, err = net.Listen("tcp", kvs.address); err != nil {
		return err
	}

	go kvs.connRoutine()
	go kvs.kvstoreRoutine()
	go kvs.acceptRoutine()

	return nil
}

func (kvs *keyValueServer) Close() {
	kvs.close <- true
	for !<-kvs.closed {
		kvs.close <- true
	}

	if kvs.listener != nil {
		kvs.listener.Close()
	}
}

func (kvs *keyValueServer) CountActive() int {
	kvs.activeRequest <- true
	count := <-kvs.activeResult
	return count
}

func (kvs *keyValueServer) CountDropped() int {
	kvs.droppedRequest <- true
	count := <-kvs.droppedResult
	return count
}

func (kvs *keyValueServer) acceptRoutine() {
	for {
		conn, err := kvs.listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			return
		}
		kvs.connChan <- 1
		go kvs.readRoutine(conn)
		go kvs.writeRoutine(conn)
	}
}

func (kvs *keyValueServer) readRoutine(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		queryMsg, err := reader.ReadBytes('\n')
		if err != nil {
			if err.Error() == "EOF" {
				kvs.connChan <- -1
			} else {
				fmt.Println("Error reading from connection:", err)
			}
			break
		}
		kvs.handleQuery(queryMsg, conn)
	}
}

func (kvs *keyValueServer) handleQuery(queryMsg []byte, conn net.Conn) {
	query := bytes.Split(bytes.TrimSpace(queryMsg), []byte(":"))

	switch string(query[0]) {
	case "Put":
		kvs.put(query[1:])
	case "Get":
		kvs.get(query[1:], conn)
	case "Delete":
		kvs.delete(query[1:])
	case "Update":
		kvs.update(query[1:])
	default:
		fmt.Println("Unknown query")
	}
}

func (kvs *keyValueServer) put(queryMsg [][]byte) {
	kvs.putChan <- keyValue{string(queryMsg[0]), queryMsg[1]}
}

func (kvs *keyValueServer) get(queryMsg [][]byte, conn net.Conn) {
	kvs.getChan <- getQuery{conn, string(queryMsg[0])}
}

func (kvs *keyValueServer) delete(queryMsg [][]byte) {
	kvs.deleteChan <- string(queryMsg[0])
}

func (kvs *keyValueServer) update(queryMsg [][]byte) {
	kvs.updateChan <- valueChange{
		key:      string(queryMsg[0]),
		oldValue: queryMsg[1],
		newValue: queryMsg[2],
	}
}

func (kvs *keyValueServer) kvstoreRoutine() {
	for {
		select {
		case kv := <-kvs.putChan:
			kvs.store.Put(kv.key, kv.value)
		case key := <-kvs.deleteChan:
			kvs.store.Delete(key)
		case change := <-kvs.updateChan:
			kvs.store.Update(change.key, change.oldValue, change.newValue)
		case query := <-kvs.getChan:
			values := kvs.store.Get(query.key)
			kvs.bufValues(values, query)
		}
	}
}

func (kvs *keyValueServer) bufValues(values [][]byte, query getQuery) {
	buf := kvs.getBuf(query.conn)

	for _, value := range values {
		responseMsg := fmt.Sprintf("%s:%s\n", query.key, value)
		select {
		case buf <- []byte(responseMsg):
		default:
			fmt.Println("Buffer full. Dropping value")
		}
	}
}

func (kvs *keyValueServer) getBuf(conn net.Conn) chan []byte {
	kvs.bufLock <- true

	bufKey := conn.RemoteAddr().String()
	if _, ok := kvs.cliBuf[bufKey]; !ok {
		kvs.cliBuf[bufKey] = make(chan []byte, bufferSize)
	}
	buf := kvs.cliBuf[bufKey]

	<-kvs.bufLock
	return buf
}

func (kvs *keyValueServer) writeRoutine(conn net.Conn) {
	for {
		buf := kvs.getBuf(conn)
		msg := <-buf
		if _, err := conn.Write(msg); err != nil {
			fmt.Println("Error in sending msg to client")
		}
	}
}

func (kvs *keyValueServer) connRoutine() {
	for {
		select {
		case change := <-kvs.connChan:
			kvs.activeCount += change

			if change < 0 {
				kvs.droppedCount -= change
			}
		case <-kvs.activeRequest:
			kvs.activeResult <- kvs.activeCount
		case <-kvs.droppedRequest:
			kvs.droppedResult <- kvs.droppedCount
		case <-kvs.close:
			if kvs.activeCount == 0 {
				kvs.closed <- true
			} else {
				kvs.closed <- false
			}
		}
	}
}
