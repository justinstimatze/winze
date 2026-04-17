package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// a2aRequest is a JSON-RPC 2.0 request.
type a2aRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	ID      any            `json:"id"`
	Params  map[string]any `json:"params"`
}

// a2aResponse is a JSON-RPC 2.0 response.
type a2aResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

// a2aHandler returns an http.HandlerFunc that dispatches JSON-RPC 2.0 requests
// to the shared handler core. apiSecret may be empty for dev mode (no auth).
func a2aHandler(h *handler, apiSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}

		// Auth
		if apiSecret != "" {
			auth := r.Header.Get("Authorization")
			token := strings.TrimPrefix(auth, "Bearer ")
			if !strings.HasPrefix(auth, "Bearer ") ||
				subtle.ConstantTimeCompare([]byte(token), []byte(apiSecret)) != 1 {
				writeA2AError(w, nil, -32000, "Authorization: Bearer <token> required")
				return
			}
		}

		var req a2aRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 65536)).Decode(&req); err != nil {
			writeA2AError(w, nil, -32700, "Parse error")
			return
		}
		if req.JSONRPC != "2.0" {
			writeA2AError(w, req.ID, -32600, "Invalid Request: jsonrpc must be 2.0")
			return
		}

		s := req.Params
		action := str(s, "action")

		switch req.Method {
		case "agent/info":
			writeA2AResult(w, req.ID, map[string]any{
				"name":        "winze",
				"version":     "0.1.0",
				"description": "Epistemic knowledge base — typed entities, claims, provenance, disputes",
				"methods":     []string{"agent/info", "winze/query"},
				"dir":         h.dir,
				"stats":       h.coreStats(),
			})

		case "winze/query":
			switch action {
			case "search":
				q := str(s, "query")
				if q == "" {
					writeA2AError(w, req.ID, -32602, "query param required")
					return
				}
				writeA2AResult(w, req.ID, h.coreSearch(q))

			case "theories":
				concept := str(s, "concept")
				if concept == "" {
					writeA2AError(w, req.ID, -32602, "concept param required")
					return
				}
				writeA2AResult(w, req.ID, h.coreTheories(concept))

			case "claims":
				entity := str(s, "entity")
				if entity == "" {
					writeA2AError(w, req.ID, -32602, "entity param required")
					return
				}
				writeA2AResult(w, req.ID, h.coreClaims(entity))

			case "provenance":
				q := str(s, "query")
				if q == "" {
					writeA2AError(w, req.ID, -32602, "query param required")
					return
				}
				writeA2AResult(w, req.ID, h.coreProvenance(q))

			case "disputes":
				writeA2AResult(w, req.ID, h.coreDisputes())

			case "stats":
				writeA2AResult(w, req.ID, h.coreStats())

			default:
				writeA2AError(w, req.ID, -32601,
					fmt.Sprintf("unknown action %q — use: search, theories, claims, provenance, disputes, stats", action))
			}

		default:
			writeA2AError(w, req.ID, -32601,
				fmt.Sprintf("unknown method %q — use: agent/info, winze/query", req.Method))
		}
	}
}

func writeA2AResult(w http.ResponseWriter, id any, result any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a2aResponse{JSONRPC: "2.0", ID: id, Result: result}) //nolint:errcheck
}

func writeA2AError(w http.ResponseWriter, id any, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a2aResponse{ //nolint:errcheck
		JSONRPC: "2.0",
		ID:      id,
		Error:   map[string]any{"code": code, "message": msg},
	})
}

func str(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}
