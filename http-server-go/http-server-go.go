package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	text   = "text/plain"
	stream = "application/octet-stream"
	dir    = "/tmp/data/codecrafters.io/http-server-tester/"
)

type Request struct {
	Message  []string
	Path     string
	Body     string
	Headers  Header
	Response int
	Answer   string
	Method   string
	Compress []byte
}

type Header struct {
	CType    string
	CLength  int
	Encoding string
	Close    bool
}

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

	defer conn.Close()

	for {
		buf := make([]byte, 1024) // creating a buffer to keep data
		answer := ""

		size, err := conn.Read(buf)
		if err != nil {
			// Connection closed by client is normal, don't log EOF errors
			if err.Error() != "EOF" {
				fmt.Println("Error reading input: ", err.Error())
			}
			return
		}

		request := &Request{}

		// Here we handle the types of requests
		request.Message = strings.Split(string(buf[:size]), "\r\n") // separating lines by CRLF

		requestLine := strings.Split(request.Message[0], " ") // isolating the request line
		request.Method = requestLine[0]                       // this is the request method
		request.Path = requestLine[1]                         // this is the path of the request

		if request.Method == "GET" {
			answer = request.getRequest()
		}
		if request.Method == "POST" {
			answer = request.postRequest()
		}
		if request.Method != "GET" && request.Method != "POST" {
			request.Response = 404
			answer = request.answer()
		}

		conn.Write([]byte(answer))

		if request.Headers.Close {
			return
		}
		
		// For HTTP/1.1, close connection after each request by default
		// This prevents the infinite loop from trying to read from closed connections
		return
	}
}

func (request *Request) getRequest() string {

	// Dealing with files endpoint
	if strings.HasPrefix(request.Path, "/files/") {
		fileName := strings.TrimPrefix(request.Path, "/files/")
		if !fileCheck(fileName) {
			request.Response = 404
			return request.answer()
		}
		request.Headers.CType = stream
		request.Body = string(fileInfo(fileName))
		request.Response = 200
		request.Headers.CLength = len(request.Body)
		return request.answer()
	}

	if strings.Contains(request.Path, "/echo") {

		for _, line := range request.Message {
			// Dealing with enconding headers
			if strings.HasPrefix(line, "Accept-Encoding:") {
				if strings.Contains(line, "gzip") {
					request.Headers.Encoding = "gzip"
					request.Compress = gzipBody(strings.TrimPrefix(request.Path, "/echo/"))
					request.Headers.CLength = len(request.Compress)
				}
				request.Headers.CType = text
				request.Response = 200
				return request.answer()
			}
			if strings.Contains(line, "Connection: close") {
				request.Headers.Close = true
			}
		}

		// default case
		request.Body = strings.TrimPrefix(request.Path, "/echo/")
		request.Headers.CType = text
		request.Headers.CLength = len(request.Body)
		request.Response = 200
		return request.answer()
	}

	// Dealing with user-agent headers
	if request.Path == "/user-agent" {
		for _, line := range request.Message {
			if strings.HasPrefix(line, "User-Agent:") {
				request.Body = strings.TrimPrefix(line, "User-Agent: ")
				request.Headers.CType = text
				request.Response = 200
				request.Headers.CLength = len(request.Body)
				return request.answer()
			}
		}
	}

	// Simple case, no endpoint
	if request.Path == "/" {
		for _, line := range request.Message {
			if strings.Contains(line, "Connection: close") {
				request.Headers.Close = true
			}
		}
		request.Response = 200
		return request.answer()
	}

	// Sanity check
	request.Response = 404
	return request.answer()

}

func (request *Request) postRequest() string {

	// We only accept POST from the /files endpoint
	if !strings.HasPrefix(request.Path, "/files") {
		request.Response = 404
		return request.answer()
	}

	fileName := strings.TrimPrefix(request.Path, "/files/")
	fileContent := request.Message[5]
	if fileCreate(fileName, fileContent) {
		request.Response = 201
		return request.answer()
	}

	request.Response = 404
	return request.answer()
}

func (request *Request) answer() string {

	// strings.Builder seems to be the best way to build strings at this point
	var statusLine, cEncoding, cTypeField, cLengthField, answer strings.Builder

	if request.Response == 201 {
		statusLine.WriteString("HTTP/1.1 201 Created\r\n\r\n")
		return statusLine.String()
	}
	if request.Response == 404 {
		statusLine.WriteString("HTTP/1.1 404 Not Found\r\n\r\n")
		return statusLine.String()
	}
	if request.Response == 200 {
		statusLine.WriteString("HTTP/1.1 200 OK\r\n")
		answer.WriteString(statusLine.String())
		if request.Headers.Encoding != "" {
			cEncoding.WriteString("Content-Encoding: ")
			cEncoding.WriteString(request.Headers.Encoding)
			cEncoding.WriteString("\r\n")
			answer.WriteString(cEncoding.String())
		}
		if request.Headers.CType != "" {
			cTypeField.WriteString("Content-Type: ")
			cTypeField.WriteString(request.Headers.CType)
			cTypeField.WriteString("\r\n")
			answer.WriteString(cTypeField.String())
		}
		if request.Headers.CLength != 0 {
			cLengthField.WriteString("Content-Length: ")
			cLengthField.WriteString(strconv.Itoa(request.Headers.CLength))
			cLengthField.WriteString("\r\n")
			answer.WriteString(cLengthField.String())
		}
		if request.Headers.Close {
			answer.WriteString("Connection: close\r\n")
		}
		answer.WriteString("\r\n")
		if request.Body != "" {
			answer.WriteString(request.Body)
		}
		if request.Compress != nil {
			answer.Write(request.Compress)
		}
	}
	return answer.String()

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

func gzipBody(body string) []byte {
	var buff bytes.Buffer
	gzipW := gzip.NewWriter(&buff)
	_, err := gzipW.Write([]byte(body))
	if err != nil {
		fmt.Sprintln("Error in the gzip enconding phase")
		os.Exit(1)
	}
	err = gzipW.Close()
	if err != nil {
		fmt.Sprintln("Error closing gzip writer")
		os.Exit(1)
	}
	return buff.Bytes()
}
