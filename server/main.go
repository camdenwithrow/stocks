package main

import (
	"bufio"
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
	log    = common.NewLogger()
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
	MsgCh  chan []byte
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

	workers := NewWorkerPool(10)
	go DistributeConnections(listener, workers)
	go broadcastInput(workers)
	// go broadcastRandomNums(workers)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	StopWorkerPool(workers)
}

func broadcastInput(workers []*Worker) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		var wg sync.WaitGroup
		wg.Wait()
		for _, worker := range workers {
			wg.Add(1)
			go func() {
				worker.MsgCh <- []byte(text)
				defer wg.Done()
			}()
		}
	}
}

// func broadcastRandomNums(workers []*Worker) {
// 	for {
// 		msg := fmt.Sprintln(time.Now().Format("15:05:05"))
// 		for _, worker := range workers {
// 			worker.MsgCh <- []byte(msg)
// 		}
// 	}
// }

func NewWorkerPool(numWorkers int) []*Worker {
	workers := make([]*Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		worker := &Worker{
			ID:     i,
			Conns:  make(map[int]Conn),
			ConnCh: make(chan Conn, 10),
			MsgCh:  make(chan []byte, 10),
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
		case <-w.DoneCh:
			// Stop worker
			fmt.Printf("Worker %d stopping\n", w.ID)
			for id, conn := range w.Conns {
				conn.Close()
				delete(w.Conns, id)
			}
			return

		case conn := <-w.ConnCh:
			w.Conns[conn.Id] = conn
			fmt.Printf("Worker %d added connection %d\n", w.ID, conn.Id)

		case msg := <-w.MsgCh:
			var wg sync.WaitGroup
			packet := common.Packet{
				Version:   common.VERSION,
				Type:      common.MESSAGE_PACKET,
				Length:    uint16(len([]byte(msg))),
				Timestamp: uint64(time.Now().Unix()),
				Data:      []byte(msg),
			}
			for id, conn := range w.Conns {
				wg.Add(1)
				go func() {
					defer wg.Done()
					buf, err := packet.Serialize()
					if err != nil {
						log.Error("Failed to serialize package: %v", err)
						return
					}
					_, err = conn.Write(buf)
					if err != nil {
						log.Error("Error worker: %d writing to connection: %d. Error: %v", w.ID, id, err)
					}
				}()
			}
			wg.Wait()

		default:
			// Iterate over connections and process incoming data
			var wg sync.WaitGroup
			for id, conn := range w.Conns {
				wg.Add(1)
				go func() {
					defer wg.Done()
					conn.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
					buf := byteArrayPool.Get().([]byte)
					n, err := conn.Read(buf)
					if err != nil {
						if errors.Is(err, os.ErrDeadlineExceeded) {
							return
						}
						fmt.Printf("Worker %d closing connection %d\n", w.ID, id)
						delete(w.Conns, id)
						conn.Close()
						return
					}
					packet, err := common.Deserialize(buf, n)
					if err != nil {
						log.Error("Failed to deserialize packet: %v", err)
					}
					fmt.Printf("Worker %d received from connection %d: %s\n", w.ID, id, string(packet.Data))
				}()
			}
			wg.Wait()
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
