package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/camdenwithrow/stocks/common"
)

const (
	PORT       = common.SERVER_PORT
	bufferSize = 4096 // 4KB
)

var (
	log = common.NewLogger()

	connId = 0
)

type Conn struct {
	Id int
	net.Conn
}

type Worker struct {
	ID     int
	Conns  map[int]Conn
	ConnCh chan Conn
	DoneCh chan struct{}
}

var byteArrayPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, bufferSize)
	},
}

func main() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", PORT))
	if err != nil {
		log.Error("Failed to start server: %v", err)
		os.Exit(1)
	}
	defer listener.Close()

	workers := NewWorkerPool(4)
	go DistributeConnections(listener, workers)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	StopWorkerPool(workers)
}

func NewWorkerPool(numWorkers int) []*Worker {
	workers := make([]*Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		worker := &Worker{
			ID:     i,
			Conns:  make(map[int]Conn),
			ConnCh: make(chan Conn, 10),
			DoneCh: make(chan struct{}),
		}
		go worker.Start()
		workers[i] = worker
	}
	return workers
}

func (w *Worker) Start() {
	fmt.Printf("Worker %d starting\n", w.ID)
	for {
		select {
		case conn := <-w.ConnCh:
			w.Conns[conn.Id] = conn
			fmt.Printf("Worker %d added connection %d\n", w.ID, conn.Id)

		default:
			// Iterate over connections and process incoming data
			for id, conn := range w.Conns {
				conn.SetReadDeadline(time.Now().Add(1 * time.Second))
				buf := byteArrayPool.Get().([]byte)
				n, err := conn.Read(buf)
				if err != nil {
					if errors.Is(err, os.ErrDeadlineExceeded) {
						continue
					}
					fmt.Printf("Worker %d closing connection %d\n", w.ID, id)
					delete(w.Conns, id)
					conn.Close()
					continue
				}
				fmt.Printf("Worker %d received from connection %d: %s\n", w.ID, id, string(buf[:n]))
			}
		case <-w.DoneCh:
			// Stop worker
			fmt.Printf("Worker %d stopping\n", w.ID)
			for id, conn := range w.Conns {
				conn.Close()
				delete(w.Conns, id)
			}
			return
		}
	}
}

func DistributeConnections(listener net.Listener, workers []*Worker) {
	workerCount := len(workers)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error("Failed to accept connection: %v", err)
			continue
		}

		// Assign connection to a worker (round-robin)
		worker := workers[connId%workerCount]
		connId++
		worker.ConnCh <- Conn{Id: connId, Conn: conn}
	}
}

func StopWorkerPool(workers []*Worker) {
	for _, worker := range workers {
		close(worker.DoneCh)
	}
}
