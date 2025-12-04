package mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

type ToolHandler func(arguments map[string]interface{}) (*CallToolResult, error)

type Server struct {
	Tools map[string]Tool
	Handlers map[string]ToolHandler
	sessions sync.Map // map[string]chan string
}

func NewServer() *Server {
	return &Server{
		Tools:    make(map[string]Tool),
		Handlers: make(map[string]ToolHandler),
	}
}

func (s *Server) RegisterTool(tool Tool, handler ToolHandler) {
	s.Tools[tool.Name] = tool
	s.Handlers[tool.Name] = handler
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Simple router
	if r.URL.Path == "/sse" {
		s.handleSSE(w, r)
		return
	}
	if r.URL.Path == "/message" {
		s.handleMessage(w, r)
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	sessionID := uuid.New().String()
	msgChan := make(chan string, 10)
	s.sessions.Store(sessionID, msgChan)
	defer s.sessions.Delete(sessionID)

	// Send endpoint event
	endpoint := fmt.Sprintf("/message?sessionId=%s", sessionID)
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpoint)
	flusher.Flush()

	log.Printf("New session: %s", sessionID)

	// Keep connection open and send messages
	ctx := r.Context()
	for {
		select {
		case msg := <-msgChan:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		case <-ctx.Done():
			log.Printf("Session closed: %s", sessionID)
			return
		}
	}
}

func (s *Server) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId", http.StatusBadRequest)
		return
	}

	val, ok := s.sessions.Load(sessionID)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}
	msgChan := val.(chan string)

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Respond with 202 Accepted
	w.WriteHeader(http.StatusAccepted)
	
	// Process request asynchronously
	go s.processRequest(req, msgChan)
}

func (s *Server) processRequest(req JSONRPCRequest, msgChan chan string) {
	var response JSONRPCResponse
	response.JSONRPC = "2.0"
	response.ID = req.ID

	switch req.Method {
	case "tools/list":
		tools := make([]Tool, 0, len(s.Tools))
		for _, t := range s.Tools {
			tools = append(tools, t)
		}
		response.Result = map[string]interface{}{
			"tools": tools,
		}
	case "tools/call":
		var callReq CallToolRequest
		// We need to re-marshal params to decode into CallToolRequest because Params is RawMessage
		// Actually, standard MCP params for tools/call are: { "name": "...", "arguments": {...} }
		if err := json.Unmarshal(req.Params, &callReq); err != nil {
			response.Error = &JSONRPCError{Code: -32602, Message: "Invalid params"}
			break
		}

		handler, ok := s.Handlers[callReq.Name]
		if !ok {
			response.Error = &JSONRPCError{Code: -32601, Message: "Method not found"}
			break
		}

		result, err := handler(callReq.Arguments)
		if err != nil {
			response.Result = CallToolResult{
				Content: []Content{{Type: "text", Text: err.Error()}},
				IsError: true,
			}
		} else {
			response.Result = result
		}
	case "initialize":
		// Handle initialize
		response.Result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name": "adomcp",
				"version": "1.0.0",
			},
		}
	case "notifications/initialized":
		// No response needed for notifications
		return
	default:
		// Ignore other methods or return error
		// For MCP, we should probably return MethodNotFound if we don't handle it
		// But for ping/etc we might want to be silent or generic.
		// Let's return error for now.
		response.Error = &JSONRPCError{Code: -32601, Message: "Method not found: " + req.Method}
	}

	respBytes, _ := json.Marshal(response)
	msgChan <- string(respBytes)
}
