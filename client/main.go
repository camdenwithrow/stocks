package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/camdenwithrow/stocks/common"
)

const PORT = common.SERVER_PORT

var log = common.NewLogger()

type BroadcastMsg struct {
	msg       string
	readCount *atomic.Uint64
}

var msgQ map[int]*BroadcastMsg = make(map[int]*BroadcastMsg)

func main() {
	connNum := 100

	var wg sync.WaitGroup
	go handleInput(connNum)

	for i := 0; i < connNum; i++ {
		wg.Add(1)
		go startClient(&wg)
		time.Sleep(2 * time.Millisecond)
	}
	wg.Wait()
}

func startClient(wg *sync.WaitGroup) {
	defer wg.Done()

	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", PORT))
	if err != nil {
		log.Error("Error connecting to server: %v", err)
		return
	}
	defer conn.Close()
	log.Info("Connected to the server")

	readMessages := make(map[int]bool)
	buf := make([]byte, 1024)
	for {
		newReadMessages := make(map[int]bool)

		for k := range msgQ {
			if !readMessages[k] {
				packet := common.Packet{
					Version:   common.VERSION,
					Type:      common.MESSAGE_PACKET,
					Length:    uint16(len([]byte(msgQ[k].msg))),
					Timestamp: uint64(time.Now().Unix()),
					Data:      []byte(msgQ[k].msg),
				}
				buf, err := packet.Serialize()
				if err != nil {
					log.Error("Failed to serialize packet: %v", err)
					continue
				}
				conn.Write(buf)
				msgQ[k].readCount.Add(1)
			}
			newReadMessages[k] = true
		}
		readMessages = newReadMessages

		err := conn.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
		n, err := conn.Read(buf)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			if errors.Is(err, io.EOF) {
				err := retryConnection(conn)
				if err != nil {
					log.Error("Failed to find a connection. Shutting down")
					return
				}
				continue
			} else {
				log.Error("Error reading connection: %v", err)
				return
			}
		}

		packet, err := common.Deserialize(buf, n)
		if err != nil {
			log.Error("Failed to deserialize packet: %v", err)
		}
		fmt.Printf("Received: %s\n", string(packet.Data))
	}
}

func handleInput(clientCount int) {
	msgId := 0
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		for i := range msgQ {
			if msgQ[i].readCount.Load() == uint64(clientCount) {
				delete(msgQ, i)
			}
		}
		text := scanner.Text()
		msgQ[msgId] = &BroadcastMsg{msg: text, readCount: &atomic.Uint64{}}
		msgId++
	}
}

func retryConnection(conn net.Conn) error {
	retryCount := 10
	waitTime := 3 * time.Second
	log.Warn("Disconnected from server, retrying in %v", waitTime)
	ticker := time.NewTicker(waitTime)
	var err error
	for i := 1; i <= retryCount; i++ {
		select {
		case t := <-ticker.C:
			t.UTC()
			log.Info("(%d/%d) Retrying connection...", i, retryCount)
			conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", PORT))
			if err != nil {
				log.Warn("Couldn't find connection, retrying in %v", waitTime)
				continue
			}
			log.Info("Connected to server")
			return nil
		}
	}
	return errors.New("Failed to connect")
}
