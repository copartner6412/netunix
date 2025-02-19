package netunix_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/copartner6412/netunix"
)

func TestServerClientInteraction(t *testing.T) {
	// Step 1: Create a temporary Unix domain socket
	socketFile, err := os.CreateTemp("", "unix_socket")
	if err != nil {
		t.Fatalf("error creating temporary socket file: %v", err)
	}

	socketPath := socketFile.Name()
	defer os.Remove(socketPath) // Clean up after test
	socketFile.Close()

	// Step 2: Create a router and add a simple handler
	router := make(netunix.Router)
	router.HandleFunc("GET /hello", func(body []byte) []byte {
		return []byte("Hello, world!")
	})

	// Step 3: Create and start the server
	server := netunix.Server{
		SocketPath: socketPath,
		Router:     router,
	}
	defer server.Close()
	
	go func() {
		if err := server.Listen(); err != nil {
			return
		}
	}()

	// Give the server a moment to start listening
	time.Sleep(100 * time.Millisecond)

	// Step 4: Create a client
	client := netunix.Client{
		SocketPath: socketPath,
	}

	// Step 5: Create a request and send it from the client to the server
	request := netunix.Request{
		Method: netunix.MethodGet,
		Path:   "/hello",
		Body:   nil, // No body required for this request
	}

	response, err := client.Send(request)
	if err != nil {
		t.Fatalf("error sending request: %v", err)
	}

	// Step 6: Validate the response
	expectedBody := []byte("Hello, world!")
	if response.StatusCode != 0 {
		t.Errorf("expected status code 0, got %d", response.StatusCode)
	}
	if !bytes.Equal(response.Body, expectedBody) {
		t.Errorf("expected body %q, got %q", expectedBody, response.Body)
	}
}
