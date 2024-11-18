package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/camdenwithrow/stocks/common"
)

const PORT = common.SERVER_PORT

var log = common.NewLogger()
var conn net.Conn

func main() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", PORT))
	if err != nil {
		log.Error("Failed to start server: %v", err)
		os.Exit(1)
	}
	defer listener.Close()
	log.Info("Server is listening on port: %d", PORT)

	go handleInput()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error("Failed to accept connection: %v", err)
		}
		log.Info("Client connected")
		go handleClient(conn)
	}
}

func handleInput() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		_, err := conn.Write([]byte(text))
		if err != nil {
			log.Error("Could not write to connection")
			return
		}
	}
}

func handleClient(newConn net.Conn) {
	conn = newConn
	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err == io.EOF {
			log.Error("Client disconnected")
			return
		} else if err != nil {
			log.Error("Error reading connection: %v", err)
			return
		}
		fmt.Printf("Received: %s\n", string(buf[:n]))
	}
}
