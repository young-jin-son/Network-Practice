/** ChatClient.go
 * Student ID: 20200768
 * Name: Youngjin Son **/

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
)

/* Request Code *
 * 0: make new connection
 * 1: default (send to all)
 * 2: "\ls" command
 * 3: "\secret" command
 * 4: "\except" command
 * 5: "\ping" command
 * 6: "\quit" command */
type Request struct {
	Header struct {
		Code     byte   `json:"code"`
		Sender   string `json:"sender"`
		Receiver string `json:"receiver"`
	} `json:"header"`
	Body struct {
		Message string `json:"message"`
	} `json:"body"`
}

/* Response Code *
 * 0: for RTT
 * 1: response for my request
 * 2: message from other clients
 * 3: something bad
 * 4: server terminated */
type Response struct {
	Code    byte   `json:"code"`
	Message string `json:"message"`
}

func main() {
	// Check nickname.
	if len(os.Args) < 2 {
		fmt.Println("Please enter your nickname.")
		os.Exit(0)
	}

	nickname := os.Args[1]
	if isValidNickname(nickname) == false {
		fmt.Println("Please enter a valid nickname.\n(English only, 32 characters or less)")
		os.Exit(0)
	}

	// Connect to server.
	serverName := "nsl2.cau.ac.kr"
	serverPort := "30768"

	conn, err := net.Dial("tcp", serverName+":"+serverPort)
	if err != nil {
		fmt.Println("Error connecting to server.")
		return
	}
	defer conn.Close()

	initConn(conn, nickname)

	// Exits when Ctrl-C is entered.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		fmt.Printf("\n\n")
		request, _ := newRequest(byte(6), nickname, "", "")
		exit(conn, request)
		os.Exit(0)
	}()

	sendTime := time.Now()

	// Receive messages.
	go func() {
		for {
			responses := receiveRes(conn)
			for _, response := range responses {
				if response.Code == 0 { // ping
					rtt := time.Since(sendTime)
					fmt.Printf("RTT = %.3f ms\n\n", float64(rtt.Microseconds())/1000)
				} else if response.Code == 2 {
					fmt.Printf("%s\n\n", response.Message)
				} else if response.Code == 3 || response.Code == 4 {
					fmt.Printf("%s\n\n", response.Message)
					os.Exit(0)
				}
			}
		}
	}()

	// Send Requests.
	for {
		input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		input = strings.TrimSpace(input)
		fmt.Printf("\n")

		receiver := ""
		message := ""

		if strings.HasPrefix(input, "\\") { // Command entered
			split := strings.Split(input, " ")
			command := split[0]

			switch command {
			case "\\ls":
				request, _ := newRequest(2, nickname, receiver, message)
				sendReq(conn, request)

			case "\\secret":
				receiver = split[1]
				message = strings.Join(split[2:], " ")

				request, _ := newRequest(3, nickname, receiver, message)
				sendReq(conn, request)

			case "\\except":
				receiver = split[1]
				message = strings.Join(split[2:], " ")

				request, _ := newRequest(4, nickname, receiver, message)
				sendReq(conn, request)

			case "\\ping":
				sendTime = time.Now()
				request, _ := newRequest(5, nickname, receiver, message)
				sendReq(conn, request)

			case "\\quit":
				request, _ := newRequest(6, nickname, receiver, message)
				exit(conn, request)

			default:
				fmt.Printf("Invalid command.\n\n")
				continue
			}

		} else { // No command, broadcast message
			message = input
			request, _ := newRequest(1, nickname, receiver, message)
			sendReq(conn, request)
		}
	}
}

/** max length of <= 32, English nickname, no spaces or special char in nickname. */
func isValidNickname(userNickname string) bool {
	length := len(userNickname)
	if length > 32 {
		return false
	}

	for i := 0; i < length; i++ {
		matched, _ := regexp.MatchString("[a-zA-Z]", string(userNickname[i]))
		if matched == false {
			return false
		}
	}

	return true
}

/** Return a request packet. **/
func newRequest(code byte, sender string, receiver string, message string) ([]byte, error) {
	request := Request{
		Header: struct {
			Code     byte   `json:"code"`
			Sender   string `json:"sender"`
			Receiver string `json:"receiver"`
		}{Code: code, Sender: sender, Receiver: receiver},
		Body: struct {
			Message string `json:"message"`
		}{Message: message},
	}

	return json.Marshal(request)
}

/** Initialize connection. **/
func initConn(conn net.Conn, nickname string) {
	packet, _ := newRequest(0, nickname, "", "")

	if err := sendReq(conn, packet); err != nil {
		fmt.Println("\nError sending request.\n")
		return
	}

	buffer := make([]byte, 1024)
	n, _ := conn.Read(buffer)

	var response Response
	err := json.Unmarshal(buffer[:n], &response)
	if err != nil {
		fmt.Println("Error decoding response:", err)
		os.Exit(1)
	}

	fmt.Printf("\n%s\n\n", response.Message)

	if response.Code == 3 { // Something wrong
		os.Exit(0)
	}
}

/** Send request to server and return error */
func sendReq(conn net.Conn, packet []byte) error {
	_, err := conn.Write(packet)
	return err
}

/** Return response, rtt and error */
func receiveRes(conn net.Conn) []Response {
	buffer := make([]byte, 1024)

	n, _ := conn.Read(buffer)

	var responses []Response
	decoder := json.NewDecoder(bytes.NewReader(buffer[:n]))
	for decoder.More() {
		var response Response
		if err := decoder.Decode(&response); err != nil {
			fmt.Println("Error decoding:", err)
			continue
		}
		responses = append(responses, response)
	}
	return responses
}

/** Disconnect and exit program */
func exit(conn net.Conn, request []byte) {
	_ = sendReq(conn, request)

	conn.Close()
	fmt.Println("gg~")
	os.Exit(0)
}
