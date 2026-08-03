package constant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPath2RelayModeCodexBackendResponses(t *testing.T) {
	if got := Path2RelayMode("/backend-api/codex/responses"); got != RelayModeResponses {
		t.Fatalf("unexpected relay mode for codex responses: got %d want %d", got, RelayModeResponses)
	}
	if got := Path2RelayMode("/backend-api/codex/responses/compact"); got != RelayModeResponsesCompact {
		t.Fatalf("unexpected relay mode for codex compact responses: got %d want %d", got, RelayModeResponsesCompact)
	}
}

func TestPath2RelayMode(t *testing.T) {
	tests := []struct {
		path string
		want int
	}{
		{path: "/v1/alpha/search", want: RelayModeAlphaSearch},
		{path: "/v1/alpha/search?foo=1", want: RelayModeAlphaSearch},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, Path2RelayMode(tt.path))
		})
	}
}
