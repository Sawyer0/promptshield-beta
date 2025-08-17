-- Simple POST script for wrk to hit /v1/scan with JSON body
wrk.method = "POST"
wrk.body   = '{"content":"test content to scan"}'
wrk.headers["Content-Type"] = "application/json"


