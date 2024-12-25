/** MyUDPServer.go
 * Student ID: 20200768
 * Name: Youngjin Son **/

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
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
	serverPort := "20768"

	pconn, err := net.ListenPacket("udp", ":"+serverPort)
	if err != nil {
		fmt.Println("Error starting UDP server:", err)
		return
	}
	defer pconn.Close()

	startTime := time.Now()
	requestCount := 0

	fmt.Printf("Server is ready to receive on port %s\n", serverPort)
	buffer := make([]byte, 1024)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() { // Exits when Ctrl-c is entered
		<-sig
		fmt.Println("\nBye bye~")
		os.Exit(0)
	}()

	for {
		requestCount++

		count, clientAddr, _ := pconn.ReadFrom(buffer)

		fmt.Printf("Connection request from %s\n", clientAddr.String())

		var packet MyPacket
		if err := json.Unmarshal(buffer[:count], &packet); err != nil { // get packet
			fmt.Println("Error decoding packet.")
			continue
		}

		fmt.Println("Command", packet.Header.Command)

		switch packet.Header.Command {
		case "1": // convert text to uppercase
			response := []byte(strings.ToUpper(packet.Body.Data))
			sendRes(pconn, clientAddr, response)

		case "2": // server running time
			uptime := time.Since(startTime)
			hours := int(uptime.Hours())
			minutes := int(uptime.Minutes()) % 60
			seconds := int(uptime.Seconds()) % 60

			response := []byte(fmt.Sprintf("runtime = %02d:%02d:%02d", hours, minutes, seconds))
			sendRes(pconn, clientAddr, response)

		case "3": // client ip and port
			clientIP := clientAddr.(*net.UDPAddr).IP
			clientPort := clientAddr.(*net.UDPAddr).Port

			response := []byte(fmt.Sprintf("client IP = %s, port = %d", clientIP, clientPort))
			sendRes(pconn, clientAddr, response)

		case "4": // number of requests
			response := []byte(fmt.Sprintf("requests served = %d", requestCount))
			sendRes(pconn, clientAddr, response)

		default:
			fmt.Println("Invalid command option.")
		}
	}
}

/** Send response to client */
func sendRes(pconn net.PacketConn, clientAddr net.Addr, response []byte) {
	_, err := pconn.WriteTo(response, clientAddr)
	if err != nil {
		fmt.Println("Error sending response.")
	}
}
