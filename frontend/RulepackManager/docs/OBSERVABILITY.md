# Curated monitoring (hybrid approach)

This app now exposes a tenant-safe metrics API backed by Prometheus and a basic UI page.

Endpoints
- GET /api/metrics/enforcer?range=6h&step=60s
  - Returns:
    - requests: rate(promhttp_metric_handler_requests_total{job="promptshield-enforcer",region=...}[5m])
    - redactions: increase(ps_extproc_redactions_total{...}[5m])
    - cpu: sum by (pod) (rate(process_cpu_seconds_total{...}[5m]))
    - heap: go_memstats_alloc_bytes{...}

Config
- PROM_URL: Prometheus base URL (defaults to http://localhost:9090)
- PS_REGION_DEFAULT: default region label (defaults to us-east-1)

UI
- /monitoring/enforcer shows a simple JSON view of chart-ready series. Swap in charts when ready.

Dev quickstart
1) Option A: Let the BFF auto-start port-forwards by setting PS_AUTO_PORT_FORWARD=true in your .env.local.
   Option B: Start manually:
   - Prometheus: kubectl -n monitoring port-forward svc/kube-prometheus-stack-prometheus 9090:9090
   - Grafana:    kubectl -n monitoring port-forward svc/kube-prometheus-stack-grafana 3000:80
2) Start the BFF/UI:
   - cd frontend/RulepackManager
   - copy .env.example to .env.local and adjust as needed
   - npm run dev
3) Visit http://localhost:8096/monitoring/enforcer (or your configured PORT) while signed in.

