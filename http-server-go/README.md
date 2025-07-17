# HTTP Server Go

A simple HTTP server implementation in Go for the CodeCrafters challenge.

## Features

- HTTP/1.1 compliant server
- Concurrent connection handling
- Support for GET and POST requests
- Gzip compression support
- File serving and creation
- Multiple endpoints: `/`, `/echo/*`, `/user-agent`, `/files/*`

## Running the Server

```bash
go run http-server-go.go
```

The server will start on port 4221.

## Running Tests

Run all tests:
```bash
go test -v
```

Run benchmarks:
```bash
go test -bench=.
```

Run specific test:
```bash
go test -v -run TestEchoEndpoint
```

## Test Coverage

The test suite includes:

- **Unit tests** for individual functions
- **Integration tests** for HTTP endpoints
- **Concurrency tests** for simultaneous requests
- **File operation tests** for GET/POST file handling
- **Compression tests** for gzip encoding
- **Connection tests** for proper connection handling
- **Benchmarks** for performance testing

## Endpoints

- `GET /` - Returns 200 OK
- `GET /echo/{text}` - Echoes back the text (supports gzip compression)
- `GET /user-agent` - Returns the User-Agent header value
- `GET /files/{filename}` - Serves files from the configured directory
- `POST /files/{filename}` - Creates files in the configured directory