/** MyTCPClient.go
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
	serverPort := "30768"

	conn, err := net.Dial("tcp", serverName+":"+serverPort)
	if err != nil {
		fmt.Println("Error connecting to server.")
		return
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.TCPAddr)
	fmt.Printf("Client is running on port %d\n", localAddr.Port)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() { // Exits when Ctrl-C is entered
		<-sig
		exit(conn)
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
				fmt.Println("\nError encoding packet.\n")
				continue
			}

			sendTime := time.Now()
			if err := sendReq(conn, packet); err != nil {
				fmt.Println("\nError sending request.\n")
				continue
			}

			response, rtt, err := receiveRes(conn, sendTime)
			if err != nil {
				fmt.Println("\nError receiving response.\n")
				continue
			}

			fmt.Printf("\nResponse from server: %s\n", response)
			fmt.Printf("RTT = %.3f ms\n\n", float64(rtt.Microseconds())/1000)

		case "5":
			exit(conn)

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
func sendReq(conn net.Conn, packet []byte) error {
	_, err := conn.Write(packet)
	return err
}

/** Return response, rtt and error */
func receiveRes(conn net.Conn, sendTime time.Time) (string, time.Duration, error) {
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(time.Second)) // Timeout 1s

	n, err := conn.Read(buffer)
	rtt := time.Since(sendTime)

	if err != nil {
		fmt.Println("Server is not running.")
		os.Exit(0)
	}
	return string(buffer[:n]), rtt, nil
}

/** Disconnect and exit program */
func exit(conn net.Conn) {
	fmt.Println("\nBye bye~")
	os.Exit(0)
}
