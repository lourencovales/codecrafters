package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	text   = "text/plain"
	stream = "application/octet-stream"
	dir    = "/tmp/data/codecrafters.io/http-server-tester/"
)

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221") // starting the listener
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for { // for loop so it keeps on running
		conn, err := l.Accept() // accepting connections
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go connectionHandler(conn) // concurrently handle the connections
	}
}

func connectionHandler(conn net.Conn) {

	buf := make([]byte, 1000) // creating a buffer to keep data
	answer := ""

	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading input: ", err.Error())
		os.Exit(1)
	}

	// Here we handle the types of requests
	body := strings.Split(string(buf), "\r\n") // separating lines by CRLF

	st := strings.Split(body[0], " ") // isolating the request line
	typeRequest := st[0]              // this is the request type
	path := st[1]                     // this is the path of the request
	if typeRequest == "GET" {
		answer = getRequest(path, body)
	}
	if typeRequest == "POST" {
		answer = postRequest(path, body)
	}
	if typeRequest != "GET" && typeRequest != "POST" {
		answer = answerType(404, "", "")
	}

	conn.Write([]byte(answer))
	conn.Close()

}

func getRequest(path string, body []string) string {

	answer := ""
	switch {
	case path == "/":
		answer = answerType(200, "", "")
	case strings.HasPrefix(path, "/echo"):
		echo := strings.TrimPrefix(path, "/echo/") // path of the GET req
		answer = answerType(200, echo, text)
	case strings.HasPrefix(path, "/user-agent"):
		line := strings.Split(body[2], " ") // Headers part of the request
		ua := strings.Trim(line[1], "\r\n") // trimming the User-Agent header
		answer = answerType(200, ua, text)
	case strings.HasPrefix(path, "/files"): // path of the GET req
		fileName := strings.TrimPrefix(path, "/files/")
		if !fileCheck(fileName) { // if file doesn't exist
			answer = answerType(404, "", "")
		} else {
			content := fileInfo(fileName)
			answer = answerType(200, string(content), stream)
		}
	default:
		answer = answerType(404, "", "")
	}
	return answer
}

func postRequest(path string, body []string) string {

	answer := ""
	if !strings.HasPrefix(path, "/files") {
		answer = answerType(404, "", "")
	}
	fileName := strings.TrimPrefix(path, "/files/")
	fileContent := strings.Trim(body[5], "\x00") // trimming the extra from the buffer
	if fileCreate(fileName, fileContent) {
		answer = answerType(201, "", "")
	}

	return answer
}

func answerType(value int, text, cType string) string {
	switch {
	case value == 200 && text == "":
		return "HTTP/1.1 200 OK\r\n\r\n"
	case value == 200:
		return fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n%s", cType, len(text), text)
	case value == 404:
		return "HTTP/1.1 404 Not Found\r\n\r\n"
	case value == 201:
		return "HTTP/1.1 201 Created\r\n\r\n"
	}
	return fmt.Sprintln("Error getting the answer")
}

func fileCheck(file string) bool {
	path := dir + file
	_, err := os.Stat(path)
	if err != nil {
		fmt.Sprintf("The file %s doesn't exist", file)
		return false
	}
	return true
}

func fileInfo(file string) []byte {
	path := dir + file
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Sprintf("Error accessing file %s", file)
		return nil
	}
	return data
}

func fileCreate(name string, fileContent string) bool {
	err := os.WriteFile(dir+name, []byte(fileContent), 0640)
	if err != nil {
		return false
	}
	return true
}
