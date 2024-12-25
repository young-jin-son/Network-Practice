/** SplitFileServer.go
 * Student ID: 20200768
 * Name: Youngjin Son **/

package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	// Check port number
	if len(os.Args) < 2 {
		fmt.Println("Please enter port number.")
		os.Exit(0)
	}

	serverPort := os.Args[1]

	listner, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer listner.Close()

	fmt.Println("Server is ready to receive on port", serverPort)

	// Exits when Ctrl-C is entered.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
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

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	var buffer bytes.Buffer
	_, err := io.Copy(&buffer, conn)
	if err != nil {
		fmt.Println("Error receive request:", err)
	}

	// extract command and file name
	firstLine, _ := buffer.ReadString('\n')
	firstLine = strings.TrimSpace(firstLine)

	parts := strings.SplitN(firstLine, " ", 2)
	if len(parts) != 2 {
		fmt.Println("Invalid request format")
		return
	}

	command := parts[0]
	fileName := parts[1]

	switch command {
	case "put":
		// extract file content
		var contentBuffer bytes.Buffer
		_, err = io.Copy(&contentBuffer, &buffer)
		if err != nil {
			fmt.Println("Error reading remaining request:", err)
			return
		}
		content := contentBuffer.Bytes()

		err = receiveFile(fileName, content)
		if err != nil {
			fmt.Println("Error receiving file:", err)
		} else {
			fmt.Println("File received successfully:", fileName)
		}

	case "get":
		err = sendFile(conn, fileName)
		if err != nil {
			fmt.Println("Error sending file:", err)
		} else {
			fmt.Println("File sent successfully:", fileName)
		}

	default:
		fmt.Println("Invalid command received.")
	}
}

/** Create a file and save its contents **/
func receiveFile(fileName string, content []byte) error {
	// Create file
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer file.Close()

	// Write to file
	_, err = file.Write(content)
	if err != nil {
		return fmt.Errorf("could not write to file: %w", err)
	}

	// Get file information
	fileInfo, err := os.Stat(fileName)
	if err != nil {
		return fmt.Errorf("could not stat file: %w", err)
	}
	fileSize := fileInfo.Size()

	// Check EOF marker
	buffer := make([]byte, 3)
	file.Seek(fileSize-3, 0)
	file.Read(buffer)

	if string(buffer) == "EOF" { // If EOF mark exists, delete the mark
		file.Truncate(fileSize - 3)
	} else { // If EOF is missing, delete file and return error
		file.Close()
		os.Remove(fileName)
		return fmt.Errorf("file transfer incomplete: missing EOF marker")
	}

	return nil
}

func sendFile(conn net.Conn, fileName string) error {
	// Open file
	file, err := os.Open(fileName)
	if err != nil { // file does not exist
		conn.Write([]byte("X"))
		return fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	// Send file
	_, err = io.Copy(conn, file)
	if err != nil {
		return fmt.Errorf("could not send file content: %w", err)
	}

	conn.Close()
	return nil
}
