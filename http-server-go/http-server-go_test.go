package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const testDir = "/tmp/data/codecrafters.io/http-server-tester/"

func TestMain(m *testing.M) {
	os.MkdirAll(testDir, 0755)
	
	go func() {
		main()
	}()
	
	time.Sleep(100 * time.Millisecond)
	
	code := m.Run()
	
	os.RemoveAll(testDir)
	os.Exit(code)
}

func setupTestFile(name, content string) error {
	return os.WriteFile(testDir+name, []byte(content), 0644)
}

func makeRequest(method, path string, headers map[string]string, body string) (*http.Response, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	req, err := http.NewRequest(method, "http://localhost:4221"+path, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	return client.Do(req)
}

func TestRootEndpoint(t *testing.T) {
	resp, err := makeRequest("GET", "/", nil, "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestEchoEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"simple text", "/echo/hello", "hello"},
		{"with spaces", "/echo/hello%20world", "hello%20world"},
		{"special chars", "/echo/test!@", "test!@"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := makeRequest("GET", tt.path, nil, "")
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != 200 {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
			
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}
			
			if string(body) != tt.expected {
				t.Errorf("Expected body %q, got %q", tt.expected, string(body))
			}
			
			if resp.Header.Get("Content-Type") != "text/plain" {
				t.Errorf("Expected Content-Type text/plain, got %s", resp.Header.Get("Content-Type"))
			}
		})
	}
}

func TestEchoWithGzipCompression(t *testing.T) {
	headers := map[string]string{
		"Accept-Encoding": "gzip",
	}
	
	resp, err := makeRequest("GET", "/echo/hello", headers, "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	if resp.Header.Get("Content-Encoding") != "gzip" {
		t.Errorf("Expected Content-Encoding gzip, got %s", resp.Header.Get("Content-Encoding"))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer reader.Close()
	
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}
	
	if string(decompressed) != "hello" {
		t.Errorf("Expected decompressed body 'hello', got %q", string(decompressed))
	}
}

func TestUserAgentEndpoint(t *testing.T) {
	userAgent := "test-agent/1.0"
	headers := map[string]string{
		"User-Agent": userAgent,
	}
	
	resp, err := makeRequest("GET", "/user-agent", headers, "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	
	if string(body) != userAgent {
		t.Errorf("Expected body %q, got %q", userAgent, string(body))
	}
	
	if resp.Header.Get("Content-Type") != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestFilesEndpointGET(t *testing.T) {
	testContent := "test file content"
	testFile := "test.txt"
	
	err := setupTestFile(testFile, testContent)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	resp, err := makeRequest("GET", "/files/"+testFile, nil, "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	
	if string(body) != testContent {
		t.Errorf("Expected body %q, got %q", testContent, string(body))
	}
	
	if resp.Header.Get("Content-Type") != "application/octet-stream" {
		t.Errorf("Expected Content-Type application/octet-stream, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestFilesEndpointGETNotFound(t *testing.T) {
	resp, err := makeRequest("GET", "/files/nonexistent.txt", nil, "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestFilesEndpointPOST(t *testing.T) {
	testContent := "new file content"
	testFile := "new_file.txt"
	
	conn, err := net.Dial("tcp", "localhost:4221")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	request := fmt.Sprintf("POST /files/%s HTTP/1.1\r\nContent-Length: %d\r\n\r\n\r\n\r\n%s", testFile, len(testContent), testContent)
	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatalf("Failed to write request: %v", err)
	}
	
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	
	responseStr := string(response[:n])
	if !strings.Contains(responseStr, "201 Created") {
		t.Errorf("Expected 201 Created, got response: %s", responseStr)
	}
	
	content, err := os.ReadFile(testDir + testFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}
	
	if string(content) != testContent {
		t.Errorf("Expected file content %q, got %q", testContent, string(content))
	}
}

func TestInvalidEndpoint(t *testing.T) {
	resp, err := makeRequest("GET", "/invalid", nil, "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestUnsupportedMethod(t *testing.T) {
	resp, err := makeRequest("PUT", "/", nil, "")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestConcurrentRequests(t *testing.T) {
	const numRequests = 10
	results := make(chan error, numRequests)
	
	for i := 0; i < numRequests; i++ {
		go func(i int) {
			resp, err := makeRequest("GET", fmt.Sprintf("/echo/test%d", i), nil, "")
			if err != nil {
				results <- err
				return
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != 200 {
				results <- fmt.Errorf("expected status 200, got %d", resp.StatusCode)
				return
			}
			
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				results <- err
				return
			}
			
			expected := fmt.Sprintf("test%d", i)
			if string(body) != expected {
				results <- fmt.Errorf("expected body %q, got %q", expected, string(body))
				return
			}
			
			results <- nil
		}(i)
	}
	
	for i := 0; i < numRequests; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent request failed: %v", err)
		}
	}
}

func TestRequestStruct(t *testing.T) {
	req := &Request{
		Path:     "/test",
		Method:   "GET",
		Response: 200,
		Body:     "test body",
		Headers: Header{
			CType:   "text/plain",
			CLength: 9,
		},
	}
	
	answer := req.answer()
	
	if !strings.Contains(answer, "HTTP/1.1 200 OK") {
		t.Error("Response should contain status line")
	}
	
	if !strings.Contains(answer, "Content-Type: text/plain") {
		t.Error("Response should contain Content-Type header")
	}
	
	if !strings.Contains(answer, "Content-Length: 9") {
		t.Error("Response should contain Content-Length header")
	}
	
	if !strings.Contains(answer, "test body") {
		t.Error("Response should contain body")
	}
}

func TestFileOperations(t *testing.T) {
	testFile := "test_ops.txt"
	testContent := "test content"
	
	if fileCheck(testFile) {
		t.Error("File should not exist initially")
	}
	
	if !fileCreate(testFile, testContent) {
		t.Error("File creation should succeed")
	}
	
	if !fileCheck(testFile) {
		t.Error("File should exist after creation")
	}
	
	content := fileInfo(testFile)
	if string(content) != testContent {
		t.Errorf("Expected content %q, got %q", testContent, string(content))
	}
	
	os.Remove(testDir + testFile)
}

func TestGzipCompression(t *testing.T) {
	testText := "hello world"
	compressed := gzipBody(testText)
	
	if len(compressed) == 0 {
		t.Error("Compressed data should not be empty")
	}
	
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer reader.Close()
	
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}
	
	if string(decompressed) != testText {
		t.Errorf("Expected decompressed %q, got %q", testText, string(decompressed))
	}
}

func TestConnectionClose(t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:4221")
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	request := "GET / HTTP/1.1\r\nConnection: close\r\n\r\n"
	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatalf("Failed to write request: %v", err)
	}
	
	response := make([]byte, 1024)
	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	
	responseStr := string(response[:n])
	if !strings.Contains(responseStr, "Connection: close") {
		t.Error("Response should contain Connection: close header")
	}
}

func BenchmarkEchoEndpoint(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := makeRequest("GET", "/echo/benchmark", nil, "")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkRootEndpoint(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp, err := makeRequest("GET", "/", nil, "")
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}