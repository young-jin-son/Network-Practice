/** SplitFileClient.go
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
	server1 := "nsl2.cau.ac.kr:40768"
	server2 := "nsl2.cau.ac.kr:50768"

	// Check command and file name
	if len(os.Args) < 3 {
		fmt.Println("Please enter command and file name.")
		os.Exit(0)
	}

	command := os.Args[1]
	fileName := os.Args[2]

	if command != "get" && command != "put" {
		fmt.Println("Please enter a valid command.")
		os.Exit(0)
	}

	// Connect to servers
	conn1, err := net.Dial("tcp", server1)
	if err != nil {
		fmt.Println("Error connecting to server 1:", err)
		return
	}
	defer conn1.Close()

	conn2, err := net.Dial("tcp", server2)
	if err != nil {
		fmt.Println("Error connecting to server 2:", err)
		return
	}
	defer conn2.Close()

	// Exits when Ctrl-C is entered.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		exit(conn1, conn2)
	}()

	switch command {
	case "put":
		err := splitAndSendFile(conn1, conn2, fileName)
		if err != nil {
			fmt.Println("\nError putting file:", err)
			exit(conn1, conn2)
		}
		fmt.Println("\nSuccess put file")

	case "get":
		err := getAndMergeFile(conn1, conn2, fileName)
		if err != nil {
			fmt.Println("Error getting file:", err)
			exit(conn1, conn2)
		}
		fmt.Println("Success get file")

	default:
		fmt.Println("Invalid command.")
		exit(conn1, conn2)
	}
}

/** Generate part file names **/
func generatePartFileNames(filePath string) (string, string) {
	ext := ""
	base := filePath
	if dot := strings.LastIndex(filePath, "."); dot != -1 {
		ext = filePath[dot:]
		base = filePath[:dot]
	}
	part1FileName := base + "-part1" + ext
	part2FileName := base + "-part2" + ext
	return part1FileName, part2FileName
}

/** File split and send them to each server **/
func splitAndSendFile(conn1, conn2 net.Conn, filePath string) error {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	// Get the file size
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("could not get file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// Generate the part file names
	part1FileName, part2FileName := generatePartFileNames(filePath)

	// Send the command and file name
	_, err = conn1.Write([]byte("put " + part1FileName + "\n"))
	if err != nil {
		return fmt.Errorf("could not send file name to server 1: %w", err)
	}

	_, err = conn2.Write([]byte("put " + part2FileName + "\n"))
	if err != nil {
		return fmt.Errorf("could not send file name to server 2: %w", err)
	}

	// Read file byte by byte and send to servers alternately
	buffer := make([]byte, 1)
	sendToServer1 := true
	totalSent := int64(0)

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("could not read file: %w", err)
		}
		if n == 0 {
			break
		}

		if sendToServer1 {
			_, err = conn1.Write(buffer[:n])
			if err != nil {
				return fmt.Errorf("could not send data to server 1: %w", err)
			}
		} else {
			_, err = conn2.Write(buffer[:n])
			if err != nil {
				return fmt.Errorf("could not send data to server 2: %w", err)
			}
		}
		sendToServer1 = !sendToServer1

		// Show progress
		totalSent += int64(n)
		progress := float64(totalSent) / float64(fileSize) * 100
		fmt.Printf("\rProgress: %.2f%%", progress)
	}

	// Notifies the server that the file transfer is complete.
	_, err = conn1.Write([]byte("EOF"))
	if err != nil {
		fmt.Println("could not send EOF to server 1: %w", err)
	}

	_, err = conn2.Write([]byte("EOF"))
	if err != nil {
		fmt.Println("could not send EOF to server 2: %w", err)
	}

	return nil
}

/** Send get request to server. **/
func getFile(conn net.Conn, fileName string) error {
	_, err := conn.Write([]byte("get " + fileName + "\n"))
	if err != nil {
		return err
	}

	// closeWrite
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		err = tcpConn.CloseWrite()
		if err != nil {
			return fmt.Errorf("error closing write side: %w", err)
		}
	}

	return nil
}

/** Get file parts from servers and merge them. **/
func getAndMergeFile(conn1, conn2 net.Conn, filePath string) error {
	part1FileName, part2FileName := generatePartFileNames(filePath)

	// Request parts
	err := getFile(conn1, part1FileName)
	if err != nil {
		return fmt.Errorf("could not request file part 1: %w", err)
	}

	err = getFile(conn2, part2FileName)
	if err != nil {
		return fmt.Errorf("could not request file part 2: %w", err)
	}

	// Receive parts
	part1, err := receiveFile(conn1)
	if err != nil {
		return fmt.Errorf("could not receive file part 1: %w", err)
	}

	part2, err := receiveFile(conn2)
	if err != nil {
		return fmt.Errorf("could not receive file part 2: %w", err)
	}

	// Merge parts
	mergedContent := mergeFileParts(part1, part2)

	// Create merged file
	ext := ""
	if dot := strings.LastIndex(filePath, "."); dot != -1 {
		ext = filePath[dot:]
	}
	mergedFileName := strings.TrimSuffix(filePath, ext) + "-merged" + ext

	mergedFile, err := os.Create(mergedFileName)
	if err != nil {
		return fmt.Errorf("could not create merged file: %w", err)
	}
	defer mergedFile.Close()

	// Write merged content
	_, err = mergedFile.Write(mergedContent)
	if err != nil {
		return fmt.Errorf("could not write to merged file: %w", err)
	}

	return nil
}

/** Merge part1 and part2 in order, 1 byte each. **/
func mergeFileParts(part1, part2 []byte) []byte {
	merged := make([]byte, len(part1)+len(part2))
	i, j := 0, 0
	for k := 0; k < len(merged); k++ {
		if i < len(part1) {
			merged[k] = part1[i]
			i++
		}
		k++
		if j < len(part2) && k < len(merged) {
			merged[k] = part2[j]
			j++
		}
	}
	return merged
}

/** Receive file from server. **/
func receiveFile(conn net.Conn) ([]byte, error) {
	var buffer bytes.Buffer

	_, err := io.Copy(&buffer, conn)
	if err != nil {
		return nil, fmt.Errorf("could not read from connection: %w", err)
	}

	fileData := buffer.Bytes()
	if len(fileData) == 1 && fileData[0] == 'X' {
		return nil, fmt.Errorf("no such file on server")
	}

	return fileData, nil
}

/** Disconnect and exit program. */
func exit(conn1 net.Conn, conn2 net.Conn) {
	conn1.Close()
	conn2.Close()
	os.Exit(0)
}
