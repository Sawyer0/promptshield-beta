# Kubernetes Sidecar Egress (Example)

This example shows how to run the egress Envoy as a sidecar and transparently route outbound HTTP traffic from your application through PromptShield.

## Pattern
- iptables REDIRECT sends all outbound TCP/80 and TCP/443 to Envoy 8080
- Envoy injects tenant identity headers and calls the Gateway via ext_authz/ext_proc
- No changes in the application code

## Example Deployment (simplified)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-llm-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-llm-app
  template:
    metadata:
      labels:
        app: my-llm-app
    spec:
      containers:
      - name: app
        image: myorg/my-llm-app:latest
        env:
        - name: NO_PROXY
          value: "127.0.0.1,localhost"
        ports:
        - containerPort: 8081
      - name: envoy-egress
        image: envoyproxy/envoy:v1.28-latest
        args: ["-c","/etc/envoy/envoy.yaml","--service-cluster","egress","--log-level","info"]
        env:
        - name: TENANT_ID
          valueFrom:
            secretKeyRef:
              name: ps-egress-identity
              key: tenant-id
        - name: FRONTEND_TOKEN
          valueFrom:
            secretKeyRef:
              name: ps-egress-identity
              key: frontend-token
        - name: GATEWAY_HTTP_HOST
          value: promptshield-gateway
        - name: GATEWAY_HTTP_PORT
          value: "9090"
        - name: GATEWAY_GRPC_HOST
          value: promptshield-gateway
        - name: GATEWAY_GRPC_PORT
          value: "9091"
        volumeMounts:
        - name: envoy-config
          mountPath: /etc/envoy
        - name: iptables
          mountPath: /usr/local/bin/iptables-setup.sh
          subPath: iptables-setup.sh
        lifecycle:
          postStart:
            exec:
              command: ["/bin/sh","/usr/local/bin/iptables-setup.sh"]
      initContainers:
      - name: init-iptables
        image: alpine:3.20
        securityContext:
          capabilities:
            add: ["NET_ADMIN"]
        command: ["/bin/sh","-c","cp /scripts/iptables-setup.sh /work/iptables-setup.sh && chmod +x /work/iptables-setup.sh"]
        volumeMounts:
        - name: iptables
          mountPath: /work
        - name: iptables-scripts
          mountPath: /scripts
      volumes:
      - name: envoy-config
        configMap:
          name: ps-egress-config
      - name: iptables
        emptyDir: {}
      - name: iptables-scripts
        configMap:
          name: ps-egress-iptables
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ps-egress-config
data:
  envoy.yaml: |
    # Paste the rendered deploy/envoy/egress-dfp.tmpl.yaml content here for Kubernetes
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ps-egress-iptables
data:
  iptables-setup.sh: |
    #!/bin/sh
    set -euo pipefail
    # Redirect outbound 80/443 to Envoy 8080 (simplified example)
    iptables -t nat -A OUTPUT -p tcp --dport 80 -j REDIRECT --to-ports 8080
    iptables -t nat -A OUTPUT -p tcp --dport 443 -j REDIRECT --to-ports 8080
```

