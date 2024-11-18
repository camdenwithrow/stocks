package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/camdenwithrow/stocks/common"
)

const PORT = common.SERVER_PORT

var log = common.NewLogger()
var conn net.Conn

func main() {
	connection, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", PORT))
	if err != nil {
		log.Error("Error connecting to server: %v", err)
		return
	}
	conn = connection
	defer conn.Close()
	log.Info("Connected to the server")

	go handleInput()
	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err == io.EOF {
			err := retryConnection()
			if err != nil {
				log.Error("Failed to find a connection. Shutting down")
				return
			}
			continue
		} else if err != nil {
			log.Error("Error reading connection: %v", err)
			return
		}

		fmt.Printf("Received: %s\n", string(buf[:n]))
	}
}

func handleInput() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		conn.Write([]byte(text))
	}
}

func retryConnection() error {
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
