// Package mcp provides a minimal read-only MCP stdio server.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Blackman99/failshut/audit"
	"github.com/Blackman99/failshut/report"
)

// Run starts the JSON-RPC MCP loop on stdin/stdout.
func Run(in io.Reader, out io.Writer) error {
	br := bufio.NewReader(in)
	for {
		line, err := br.ReadBytes('\n')
		if err == io.EOF {
			if len(line) == 0 {
				return nil
			}
		} else if err != nil {
			return err
		}
		line = trimNL(line)
		if len(line) == 0 {
			continue
		}
		var req map[string]any
		if err := json.Unmarshal(line, &req); err != nil {
			_ = writeErr(out, nil, -32700, "parse error")
			continue
		}
		id := req["id"]
		method, _ := req["method"].(string)
		params, _ := req["params"].(map[string]any)
		switch method {
		case "initialize":
			_ = writeResult(out, id, map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "failshut", "version": "0.1.0"},
			})
		case "notifications/initialized", "initialized":
		case "tools/list":
			_ = writeResult(out, id, map[string]any{
				"tools": []map[string]any{
					{
						"name":        "audit_report",
						"description": "Audit local AI coding-agent permission configs for fail-open / fail-shut. Read-only; never mutates configs.",
						"inputSchema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"project": map[string]any{"type": "string", "description": "Optional project directory to include"},
							},
						},
					},
					{
						"name":        "list_findings",
						"description": "Same as audit_report but returns only FAIL OPEN findings.",
						"inputSchema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"project": map[string]any{"type": "string"},
							},
						},
					},
				},
			})
		case "tools/call":
			name, _ := params["name"].(string)
			args, _ := params["arguments"].(map[string]any)
			project, _ := args["project"].(string)
			res, err := audit.Run(audit.Options{Project: project})
			if err != nil {
				_ = writeErr(out, id, -32000, err.Error())
				continue
			}
			switch name {
			case "audit_report":
				body, _ := json.Marshal(res)
				_ = writeResult(out, id, map[string]any{
					"content": []map[string]any{{"type": "text", "text": string(body)}},
				})
			case "list_findings":
				opens := make([]report.Finding, 0)
				for _, f := range res.Findings {
					if f.Severity == report.FailOpen {
						opens = append(opens, f)
					}
				}
				body, _ := json.Marshal(opens)
				_ = writeResult(out, id, map[string]any{
					"content": []map[string]any{{"type": "text", "text": string(body)}},
				})
			default:
				_ = writeErr(out, id, -32601, "unknown tool: "+name)
			}
		case "ping":
			_ = writeResult(out, id, map[string]any{})
		default:
			if id != nil {
				_ = writeErr(out, id, -32601, "method not found: "+method)
			}
		}
		if err == io.EOF {
			return nil
		}
	}
}

func trimNL(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}

func writeResult(w io.Writer, id any, result any) error {
	return writeMsg(w, map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func writeErr(w io.Writer, id any, code int, msg string) error {
	return writeMsg(w, map[string]any{
		"jsonrpc": "2.0", "id": id,
		"error": map[string]any{"code": code, "message": msg},
	})
}

func writeMsg(w io.Writer, msg map[string]any) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", b)
	return err
}
