package netunix

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type Request struct {
	Method Method `json:"method"`
	Path   string `json:"path"`
	Body   []byte `json:"body"`
}

type Method string

const (
	MethodGet    Method = "GET"
	MethodPut    Method = "PUT"
	MethodDelete Method = "DELETE"
)

type Response struct {
	StatusCode int    `json:"status_code"`
	Error      []byte `json:"error"`
	Body       []byte `json:"body"`
}

const (
	StatusCodeSuccessful = iota
	StatusCodeInvalidPath
	StatusCodeInvalidRequest
	StatusCodeInvalidRequestBody
	StatusCodeResourceNotFound
	StatusCodeUnauthorizedAccess
	StatusCodeNothingChanged
	StatusCodeInternalServerError
	StatusCodeError
)

type Router map[string]func(requestBody []byte)Response

func (r Router) HandleFunc(pattern string, handler func([]byte) Response) {
	r[pattern] = handler
}

type Server struct {
	SocketPath string
	Router     Router
	listener	net.Listener
}

func (s *Server) Listen() error {
	// Clean up the socket path before starting
	os.Remove(s.SocketPath)

	listener, err := net.Listen("unix", s.SocketPath)
	if err != nil {
		return fmt.Errorf("error listening on Unix domain socket \"%s\": %w", s.SocketPath, err)
	}
	defer listener.Close()
	s.listener = listener

	// Server loop that continuously accepts new connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("error accepting connection: %w", err)
		}

		// Handle the connection in a separate goroutine
		go func(conn net.Conn) {
			defer conn.Close()

			decoder := json.NewDecoder(conn)
			encoder := json.NewEncoder(conn)

			var request Request
			if err := decoder.Decode(&request); err != nil {
				// Log the error but keep the server running
				fmt.Fprintf(os.Stderr, "error decoding request: %v\n", err)
				return
			}

			// Construct the route key
			routeKey := fmt.Sprintf("%s %s", request.Method, request.Path)
			handler, exists := s.Router[routeKey]
			if !exists {
				// If the route doesn't exist, respond with a custom status code
				if err := encoder.Encode(Response{
					StatusCode: 1, // Use 1 to indicate "Not Found" or custom error
					Body:       nil,
				}); err != nil {
					fmt.Fprintf(os.Stderr, "error encoding response: %v\n", err)
				}
				return
			}

			// Call the handler for the route and send the response
			response := handler(request.Body)
			if err := encoder.Encode(response); err != nil {
				fmt.Fprintf(os.Stderr, "error encoding response: %v\n", err)
			}

		}(conn)
	}
}

func (s *Server) Close() error {
	return s.listener.Close()
}

type Client struct {
	SocketPath string
}

func (c *Client) Send(request Request) (Response, error) {
	conn, err := net.Dial("unix", c.SocketPath)
	if err != nil {
		return Response{}, fmt.Errorf("error dialing to Unix domain socket \"%s\": %w", c.SocketPath, err)
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	// Send the request
	if err := encoder.Encode(request); err != nil {
		return Response{}, fmt.Errorf("error encoding request: %w", err)
	}

	// Receive the response
	var response Response
	if err := decoder.Decode(&response); err != nil {
		return Response{}, fmt.Errorf("error decoding response: %w", err)
	}

	return response, nil
}
