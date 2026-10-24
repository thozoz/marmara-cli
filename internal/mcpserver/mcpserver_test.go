package mcpserver

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func reqWith(args any) mcp.CallToolRequest {
	var r mcp.CallToolRequest
	r.Params.Arguments = args
	return r
}

// MCP clients may send the numeric id as a string or a JSON number; both must
// reach api.ParseDersID unchanged.
func TestArgumentString(t *testing.T) {
	cases := []struct {
		name string
		args any
		want string
	}{
		{"string", map[string]any{"ders_id": "123456"}, "123456"},
		{"json number", map[string]any{"ders_id": float64(123456)}, "123456"},
		{"int", map[string]any{"ders_id": 42}, "42"},
		{"json.Number", map[string]any{"ders_id": json.Number("7")}, "7"},
		{"missing", map[string]any{}, ""},
		{"nil args", nil, ""},
		{"unsupported type", map[string]any{"ders_id": true}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := argumentString(reqWith(c.args), "ders_id"); got != c.want {
				t.Errorf("argumentString = %q, want %q", got, c.want)
			}
		})
	}
}
