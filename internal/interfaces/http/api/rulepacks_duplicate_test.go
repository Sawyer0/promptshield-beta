package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/promptshield/promptshield/internal/application/services"
	"github.com/promptshield/promptshield/internal/infrastructure/persistence/memory"
)

func TestRulepackUploadRejectsDuplicateName(t *testing.T) {
	svc := services.RulepackServiceCstor(memory.NewRulepackRepository(), nil)
	srv := httptest.NewServer(NewMux(Options{AllowInsecureAdmin: true, RulepackService: svc}))
	defer srv.Close()

	pack := `apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: dup-pack
  version: 1.0.0
rules:
  - id: kw
    level: 1
    severity: WARNING
    keywords: ["hello"]
`
	upload := func() *http.Response {
		t.Helper()
		resp, err := http.Post(srv.URL+"/rulepacks/", "application/x-yaml", strings.NewReader(pack))
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	first := upload()
	_ = first.Body.Close()
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("first upload: want 201, got %d", first.StatusCode)
	}

	second := upload()
	defer second.Body.Close()
	if second.StatusCode != http.StatusConflict {
		t.Fatalf("second upload: want 409, got %d", second.StatusCode)
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(second.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Error.Code != "ALREADY_EXISTS" || !strings.Contains(body.Error.Message, `"dup-pack"`) {
		t.Fatalf("unexpected error body: %+v", body)
	}
}
