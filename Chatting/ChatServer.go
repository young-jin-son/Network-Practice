/** ChatServer.go
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

type Client struct {
	ID       int
	Nickname string
	Conn     net.Conn
}

var clients []*Client

func main() {
	serverPort := "30768"

	listner, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer listner.Close()

	activeClients := 0
	newClientID := 0

	fmt.Println("Server is ready to receive on port", serverPort)

	// Exits when Ctrl-C is entered.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		response, _ := json.Marshal(Response{Code: 4, Message: "[Chat server is closed.]"})
		broadcast(response, -1)
		fmt.Println("\nBye bye~")
		listner.Close()
		os.Exit(0)
	}()

	// Accept connection
	for {
		conn, err := listner.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				break
			}
			fmt.Println("Error accepting connection.")
			continue
		}
		defer conn.Close()

		if activeClients >= 8 {
			denyConn(conn)
		} else {
			newClient := initConn(conn, newClientID, &activeClients)

			if newClient != nil {
				fmt.Printf("[%s joined from %s. There are %d users in the room.]\n", newClient.Nickname, newClient.Conn.RemoteAddr(), activeClients)
				clients = append(clients, newClient)
				go handleConn(*newClient, &activeClients)
				newClientID++
			}
		}
	}
}

/** Initialize connection. **/
func initConn(conn net.Conn, newClientID int, activeClients *int) *Client {
	buffer := make([]byte, 1024)

	n, err := conn.Read(buffer)
	if err != nil {
		conn.Close()
		fmt.Println("client disconnected")
		return nil
	}

	var request Request
	if err := json.Unmarshal(buffer[:n], &request); err != nil {
		fmt.Println("Error decoding packet:", err)
		return nil
	}

	if isValidNickname(request.Header.Sender) {
		*activeClients++

		msg := fmt.Sprintf("[Welcome %s to CAU net-class chat room at %s.]\n[There are %d users in the room.]", request.Header.Sender, conn.LocalAddr(), *activeClients)
		response, _ := json.Marshal(Response{Code: 1, Message: msg})

		_, err := conn.Write(response)
		if err != nil {
			fmt.Println("Error sending response.")
		}

		return &Client{ID: newClientID, Nickname: request.Header.Sender, Conn: conn}

	} else {
		msg := "[nickname already used by another user. cannot connect.]"
		response, _ := json.Marshal(Response{Code: 3, Message: msg})

		_, err := conn.Write(response)
		if err != nil {
			fmt.Println("Error sending response.")
		}
		conn.Close()

		return nil
	}
}

/** Check for nickname duplication. **/
func isValidNickname(nickname string) bool {
	for _, c := range clients {
		if c.Nickname == nickname {
			return false
		}
	}
	return true
}

/** Deny new connection. **/
func denyConn(conn net.Conn) {
	message := "chatting room full. cannot connect"
	response, _ := json.Marshal(Response{Code: 3, Message: message})

	_, err := conn.Write(response)
	if err != nil {
		fmt.Println("Error sending response.")
	}
	conn.Close()
}

/** Check if the message contains "i hate professor". **/
func containsIHateProf(msg string) bool {
	lowercaseMsg := strings.ToLower(msg)
	return strings.Contains(lowercaseMsg, "i hate professor")
}

/** Handle each connection. **/
func handleConn(client Client, activeClients *int) {
	defer client.Conn.Close()

	for {
		buffer := make([]byte, 1024)

		n, _ := client.Conn.Read(buffer)

		var request Request
		_ = json.Unmarshal(buffer[:n], &request)

		requestCode := request.Header.Code

		if requestCode == 1 { // default (send to all)
			msg := fmt.Sprintf("%s> %s", client.Nickname, request.Body.Message)
			response, _ := json.Marshal(Response{Code: 2, Message: msg})
			go broadcast(response, client.ID)

		} else if requestCode == 2 { // \ls
			var info strings.Builder
			for _, c := range clients {
				addr := c.Conn.RemoteAddr().(*net.TCPAddr)
				info.WriteString(fmt.Sprintf("<%s, %s, %d>\n", c.Nickname, addr.IP, addr.Port))
			}
			response, _ := json.Marshal(Response{Code: 2, Message: info.String()})
			client.Conn.Write(response)

		} else if requestCode == 3 { // \secret
			msg := fmt.Sprintf("from: %s> %s", request.Header.Sender, request.Body.Message)
			response, _ := json.Marshal(Response{Code: 2, Message: msg})
			secret(response, request.Header.Receiver)

		} else if requestCode == 4 { // \except
			msg := fmt.Sprintf("%s> %s", request.Header.Sender, request.Body.Message)
			response, _ := json.Marshal(Response{Code: 2, Message: msg})
			except(response, client.Nickname, request.Header.Receiver)

		} else if requestCode == 5 { // \ping
			response, _ := json.Marshal(Response{Code: 0, Message: ""})
			_, err := client.Conn.Write(response)

			if err != nil {
				fmt.Println("Error sending response.")
			}

		} else if requestCode == 6 { // \quit
			removeClient(&client, activeClients)
			break

		} else {
			msg := fmt.Sprintf("invalid command: %s", request.Body.Message)
			response, _ := json.Marshal(Response{Code: 3, Message: msg})
			client.Conn.Write(response)
		}

		if containsIHateProf(request.Body.Message) {
			// Response to sender that it has been kicked out.
			response, _ := json.Marshal(Response{Code: 3, Message: "[You are kicked out of the chat room.]"})
			client.Conn.Write(response)
			client.Conn.Close()

			// Remove the sender
			go removeClient(&client, activeClients)
			break
		}
	}
}

/** Broadcast message **/
func broadcast(msg []byte, senderID int) {
	for _, client := range clients {
		if client.ID != senderID {
			_, err := client.Conn.Write(msg)
			if err != nil {
				fmt.Println("Error sending message to client:", err)
			}
		}
	}
}

/** Send secret message. **/
func secret(msg []byte, receiver string) {
	for _, client := range clients {
		if client.Nickname == receiver {
			_, err := client.Conn.Write(msg)
			if err != nil {
				fmt.Println("Error sending message to client:", err)
			}
		}
	}
}

/** Send except message. **/
func except(msg []byte, sender string, receiver string) {
	for _, client := range clients {
		if client.Nickname != sender && client.Nickname != receiver {
			_, err := client.Conn.Write(msg)
			if err != nil {
				fmt.Println("Error sending message to client:", err)
			}
		}
	}
}

/** Disconnect client and remove it from clients array **/
func removeClient(client *Client, activeClients *int) {
	msg := fmt.Sprintf("[%s left the room. There are %d users now.]", client.Nickname, *activeClients-1)
	fmt.Println(msg)
	response, _ := json.Marshal(Response{Code: 2, Message: msg})
	go broadcast(response, client.ID)

	for i, c := range clients {
		if c.ID == client.ID {
			clients = append(clients[:i], clients[i+1:]...)
			*activeClients--
			break
		}
	}
}
