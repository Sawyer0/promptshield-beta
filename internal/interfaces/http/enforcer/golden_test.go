package enforcerhttp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// normalize replaces dynamic fields with placeholders so golden comparisons are stable.
func normalize(resp map[string]any) map[string]any {
	if resp == nil {
		return resp
	}
	if _, ok := resp["request_id"]; ok {
		resp["request_id"] = "<id>"
	}
	return resp
}

func loadGolden(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	g, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	return bytes.TrimSpace(g)
}

func writeGolden(t *testing.T, name string, data []byte) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write golden: %v", err)
	}
}

func TestGolden_DecisionBodies(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "plain_allow", body: "hello", wantStatus: http.StatusOK},
		{name: "chunked_allow", body: strings.Repeat("a", 128<<10), wantStatus: http.StatusOK},
	}

	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	update := os.Getenv("UPDATE_GOLDEN") == "1"

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, srv.URL+"/check", bytes.NewBufferString(tc.body))
			// Send as chunked by omitting Content-Length on large body
			if len(tc.body) > 10<<10 {
				req.TransferEncoding = []string{"chunked"}
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status: want %d got %d", tc.wantStatus, resp.StatusCode)
			}
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			var obj map[string]any
			if err := json.Unmarshal(b, &obj); err != nil {
				t.Fatalf("json: %v", err)
			}
			norm, _ := json.Marshal(normalize(obj))
			norm = bytes.TrimSpace(norm)

			if update {
				writeGolden(t, tc.name, norm)
				return
			}
			want := loadGolden(t, tc.name)
			if !bytes.Equal(norm, want) {
				t.Fatalf("golden mismatch.\nwant: %s\n got: %s", string(want), string(norm))
			}
		})
	}
}
