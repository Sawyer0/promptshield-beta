package toolrunner

import (
    "context"
    "encoding/json"
    "io"
    "net/http"
    "time"

    "github.com/promptshield/promptshield/internal/shared/contracts"
)

type Runner struct{
    httpClient *http.Client
}

func New() *Runner {
    return &Runner{ httpClient: &http.Client{ Timeout: 10 * time.Second } }
}

func (r *Runner) Execute(ctx context.Context, req contracts.ToolExecRequest) (contracts.ToolExecResult, error) {
    start := time.Now()
    // Minimal built-in: http_fetch and echo
    switch req.ToolID {
    case "http_fetch":
        var a struct{ URL string `json:"url"`; Method string `json:"method"`; Headers map[string]string `json:"headers"` }
        _ = json.Unmarshal(req.Args, &a)
        if a.Method == "" { a.Method = "GET" }
        httpReq, err := http.NewRequestWithContext(ctx, a.Method, a.URL, nil)
        if err != nil { return contracts.ToolExecResult{}, err }
        for k, v := range a.Headers { httpReq.Header.Set(k, v) }
        resp, err := r.httpClient.Do(httpReq)
        if err != nil { return contracts.ToolExecResult{}, err }
        defer resp.Body.Close()
        b, _ := io.ReadAll(resp.Body)
        payload := map[string]any{"status": resp.StatusCode, "body": string(b)}
        raw, _ := json.Marshal(payload)
        return contracts.ToolExecResult{
            Result: raw,
            ContentType: "application/json",
            StartedAt: start,
            CompletedAt: time.Now(),
            LatencyMs: time.Since(start).Milliseconds(),
            Provider: "builtin",
            Model: "http_fetch",
        }, nil
    default:
        // Echo args for unknown tools as JSON
        payload := map[string]any{"echo": json.RawMessage(req.Args)}
        raw, _ := json.Marshal(payload)
        return contracts.ToolExecResult{
            Result: raw,
            ContentType: "application/json",
            StartedAt: start,
            CompletedAt: time.Now(),
            LatencyMs: time.Since(start).Milliseconds(),
            Provider: "builtin",
            Model: req.ToolID,
        }, nil
    }
}

