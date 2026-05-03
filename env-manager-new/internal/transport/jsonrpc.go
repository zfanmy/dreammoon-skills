package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/zfanmy/dreammoon-skills/skills/env-manager/internal/scanner"
)

// Handler processes JSON-RPC messages over stdio
type Handler struct {
	handleFunc func(ctx context.Context, method string, params json.RawMessage) (interface{}, error)
}

func NewHandler(fn func(ctx context.Context, method string, params json.RawMessage) (interface{}, error)) *Handler {
	return &Handler{handleFunc: fn}
}

// JSON-RPC 2.0 message structures
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Notification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

func (h *Handler) Serve(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	reader := bufio.NewReader(stdin)
	encoder := json.NewEncoder(stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read line from stdin
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read stdin: %w", err)
		}

		// Skip empty lines
		if len(line) == 1 { // just newline
			continue
		}

		// Parse request
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			// Send parse error
			resp := Response{
				JSONRPC: "2.0",
				Error: &RPCError{
					Code:    -32700,
					Message: "Parse error",
					Data:    err.Error(),
				},
			}
			encoder.Encode(resp)
			continue
		}

		// Handle the call
		result, err := h.handleFunc(ctx, req.Method, req.Params)

		// Build response (only if ID is present - not a notification)
		if req.ID != nil {
			resp := Response{
				JSONRPC: "2.0",
				ID:      req.ID,
			}
			if err != nil {
				resp.Error = &RPCError{
					Code:    -32603,
					Message: err.Error(),
				}
			} else {
				resp.Result = result
			}
			if err := encoder.Encode(resp); err != nil {
				log.Printf("encode response: %v", err)
			}
		}
	}
}

// SendNotification sends an unsolicited notification to stdout
func (h *Handler) SendNotification(stdout io.Writer, method string, params interface{}) error {
	notif := Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return json.NewEncoder(stdout).Encode(notif)
}

// AlertPayload is sent as notification when thresholds are breached
type AlertPayload struct {
	Node      string                `json:"node"`
	Timestamp string                `json:"timestamp"`
	Alerts    []scanner.Alert       `json:"alerts"`
	Snapshot  *scanner.NodeSnapshot `json:"snapshot,omitempty"`
}
