/** MultiTCPServer.go
 * Student ID: 20200768
 * Name: Youngjin Son **/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type MyPacket struct {
	Header struct {
		Command string `json:"command"`
	} `json:"header"`
	Body struct {
		Data string `json:"data"`
	} `json:"body"`
}

func main() {
	serverPort := "30768"

	listener, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer listener.Close()

	startTime := time.Now()
	requestCount := 0
	newClientID := 0
	clientNum := 0

	fmt.Println("Server is ready to receive on port", serverPort)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() { // Exits when Ctrl+C is entered
		<-sig
		fmt.Println("\nBye bye~")
		listener.Close()
		os.Exit(0)
	}()

	ticker := time.NewTicker(10 * time.Second) // ticker generates ticks every 10 sec.
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C: // When tick is received
				DisplayClientNum(&clientNum)
			}
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				break
			}
			fmt.Println("Error accepting connection.")
			continue
		}

		newClientID++
		clientNum++
		go handleConn(conn, newClientID, &clientNum, &requestCount, startTime)
	}
}

/** Handle each connection */
func handleConn(conn net.Conn, clientID int, clientNum *int, requestCount *int, startTime time.Time) {
	defer conn.Close()

	curTime := time.Now()
	fmt.Printf("[Time: %02d:%02d:%02d] ", curTime.Hour(), curTime.Minute(), curTime.Second())
	fmt.Printf("Client %d connected. Number of clients connected = %d\n", clientID, *clientNum)

	clientAddr := conn.RemoteAddr().(*net.TCPAddr)

	for {
		buffer := make([]byte, 1024)

		n, err := conn.Read(buffer)
		if err != nil {
			conn.Close()
			*clientNum--
			curTime = time.Now()

			fmt.Printf("[Time: %02d:%02d:%02d] ", curTime.Hour(), curTime.Minute(), curTime.Second())
			fmt.Printf("Client %d disconnected. Number of clients connected = %d\n", clientID, *clientNum)
			break
		}

		*requestCount++

		var packet MyPacket
		if err := json.Unmarshal(buffer[:n], &packet); err != nil {
			fmt.Println("Error decoding packet.")
			continue
		}

		fmt.Printf("Command %s from client %d\n", packet.Header.Command, clientID)

		switch packet.Header.Command {
		case "1": // convert text to uppercase
			response1 := bytes.ToUpper([]byte(packet.Body.Data))
			sendRes(conn, response1, clientAddr)

		case "2": // server running time
			uptime := time.Since(startTime)
			hours := int(uptime.Hours())
			minutes := int(uptime.Minutes()) % 60
			seconds := int(uptime.Seconds()) % 60

			response2 := []byte(fmt.Sprintf("runtime = %02d:%02d:%02d", hours, minutes, seconds))
			sendRes(conn, response2, clientAddr)

		case "3": // client IP and port
			response3 := []byte(fmt.Sprintf("client IP = %s, port = %d", clientAddr.IP, clientAddr.Port))
			sendRes(conn, response3, clientAddr)

		case "4": // number of requests
			response4 := []byte(fmt.Sprintf("requests served = %d", *requestCount))
			sendRes(conn, response4, clientAddr)

		default:
			fmt.Println("Invalid command option.")
		}
	}
}

/** Print the current time and number of clients every 10 sec */
func DisplayClientNum(clientNum *int) {
	curTime := time.Now()
	fmt.Printf("[Time: %02d:%02d:%02d] ", curTime.Hour(), curTime.Minute(), curTime.Second())
	fmt.Printf("Number of clients connected = %d\n", *clientNum)
}

/** Send request */
func sendRes(conn net.Conn, response []byte, clientAddr net.Addr) {
	_, err := conn.Write(response)
	if err != nil {
		fmt.Println("Error sending response to", clientAddr)
	}
}
