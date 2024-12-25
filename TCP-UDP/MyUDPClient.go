/** MyUDPClient.go
 * Student ID: 20200768
 * Name: Youngjin Son **/

package main

import (
	"bufio"
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
	serverName := "localhost"
	serverPort := "20768"

	pconn, err := net.ListenPacket("udp", ":")
	if err != nil {
		fmt.Println("Failed to connect UDP.")
		return
	}
	defer pconn.Close()

	serverAddr, err := net.ResolveUDPAddr("udp", serverName+":"+serverPort)
	if err != nil {
		fmt.Println("Error connecting to server.")
		return
	}

	localAddr := pconn.LocalAddr().(*net.UDPAddr)
	fmt.Printf("Client is running on port %d\n", localAddr.Port)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		exit(pconn)
	}()

	for {
		displayMenu()
		command := getCommand()

		switch command {
		case "1", "2", "3", "4":
			input := ""

			if command == "1" {
				fmt.Printf("Input lowercase sentence: ")
				input, _ = bufio.NewReader(os.Stdin).ReadString('\n')
				input = strings.Trim(input, "\n")
			}

			packet, err := newPacket(command, input)
			if err != nil {
				fmt.Println("Error encoding packet.")
				continue
			}

			sendTime := time.Now()
			if err := sendReq(pconn, serverAddr, packet); err != nil {
				fmt.Println("Error sending request.")
				continue
			}

			response, rtt, err := receiveRes(pconn, sendTime)
			if err != nil {
				fmt.Println("Error receiving response.")
				continue
			}

			fmt.Printf("\nResponse from server: %s\n", response)
			fmt.Printf("RTT = %.3f ms\n\n", float64(rtt.Microseconds())/1000)

		case "5":
			exit(pconn)

		default:
			fmt.Println("\nPlease enter a number between 1 and 5.\n")
		}
	}
}

func displayMenu() {
	fmt.Println("<Menu>")
	fmt.Println("1) convert text to UPPER-case")
	fmt.Println("2) get server running time")
	fmt.Println("3) get my IP address and port number")
	fmt.Println("4) get server request count")
	fmt.Println("5) exit")
}

/** Get command input */
func getCommand() string {
	fmt.Printf("Input option: ")
	command, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(command)
}

/** Return JSON format packet */
func newPacket(command, data string) ([]byte, error) {
	packet := MyPacket{
		Header: struct {
			Command string `json:"command"`
		}{Command: command},
		Body: struct {
			Data string `json:"data"`
		}{Data: data},
	}

	return json.Marshal(packet)
}

/** Send request to server and return error */
func sendReq(pconn net.PacketConn, serverAddr *net.UDPAddr, packet []byte) error {
	_, err := pconn.WriteTo(packet, serverAddr)
	return err
}

/** Return response, rtt and error */
func receiveRes(pconn net.PacketConn, sendTime time.Time) (string, time.Duration, error) {
	buffer := make([]byte, 1024)
	pconn.SetReadDeadline(time.Now().Add(time.Second)) // Timeout 1s

	n, _, err := pconn.ReadFrom(buffer)
	rtt := time.Since(sendTime)

	if err != nil {
		fmt.Println("Server is not running.")
		os.Exit(0)
	}
	return string(buffer[:n]), rtt, nil
}

/** Disconnect and exit program */
func exit(pconn net.PacketConn) {
	pconn.Close()
	fmt.Println("\nBye bye~")
	os.Exit(0)
}
