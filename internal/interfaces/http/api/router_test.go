package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersion(t *testing.T) {
	srv := httptest.NewServer(NewMux(Options{AllowInsecureAdmin: true}))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/version")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}

func TestRulepacksListEmpty(t *testing.T) {
	srv := httptest.NewServer(NewMux(Options{AllowInsecureAdmin: true, RulepackManager: NewRulepackManager()}))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/rulepacks/")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}

func TestConfigGetPutReset(t *testing.T) {
	srv := httptest.NewServer(NewMux(Options{AllowInsecureAdmin: true, AdminToken: "x"}))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/config/")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	req, _ := json.Marshal(map[string]any{"enforcement_mode": "enforce"})
	reqResp, err := http.NewRequest(http.MethodPut, srv.URL+"/config/", bytes.NewReader(req))
	if err != nil {
		t.Fatal(err)
	}
	reqResp.Header.Set("Authorization", "Bearer x")
	res2, err := http.DefaultClient.Do(reqResp)
	if err != nil {
		t.Fatal(err)
	}
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", res2.StatusCode)
	}
	resetReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/config/reset", nil)
	resetReq.Header.Set("Authorization", "Bearer x")
	res3, err := http.DefaultClient.Do(resetReq)
	if err != nil {
		t.Fatal(err)
	}
	if res3.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", res3.StatusCode)
	}
}

func TestCheckAllow(t *testing.T) {
	srv := httptest.NewServer(NewMux(Options{AllowInsecureAdmin: true}))
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/check", "text/plain", bytes.NewBufferString("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
}