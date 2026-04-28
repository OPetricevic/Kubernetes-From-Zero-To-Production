# Phase 7: Observability

## Goal
Full monitoring, logging, and tracing stack. See what's happening inside your cluster at all times.

## Prerequisites
- [ ] Phase 6 complete

---

## Step 1: Install Prometheus + Grafana via Helm

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

kubectl create namespace monitoring
```

Create `helm/prometheus-values.yaml`:
```yaml
prometheus:
  prometheusSpec:
    serviceMonitorSelectorNilUsesHelmValues: false
    podMonitorSelectorNilUsesHelmValues: false
    retention: 7d
    resources:
      requests:
        cpu: 200m
        memory: 512Mi
      limits:
        cpu: 500m
        memory: 1Gi

grafana:
  adminPassword: admin123
  persistence:
    enabled: true
    size: 2Gi

alertmanager:
  enabled: true
```

```bash
helm install monitoring prometheus-community/kube-prometheus-stack \
  -n monitoring \
  -f helm/prometheus-values.yaml

# Access Grafana
kubectl port-forward svc/monitoring-grafana -n monitoring 3000:80
# Open http://localhost:3000 — admin / admin123
```

**K8s concepts: Prometheus, Grafana, DaemonSet**
> The kube-prometheus-stack installs:
> - Prometheus: scrapes metrics from pods/nodes
> - Grafana: dashboards and visualization
> - Node Exporter (DaemonSet): runs on every node, exports node-level metrics
> - kube-state-metrics: exports K8s object metrics (pod status, deployment replicas, etc.)

---

## Step 2: Add Metrics to Your Go Services

Add a `/metrics` endpoint using the Prometheus Go client:

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

// In your main():
http.Handle("/metrics", promhttp.Handler())
```

Add custom metrics:
```go
var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total HTTP requests",
        },
        []string{"method", "path", "status"},
    )
    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path"},
    )
)
```

---

## Step 3: Create ServiceMonitor

Create `k8s/base/product-service/servicemonitor.yaml`:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: product-service
  namespace: shopstream
  labels:
    release: monitoring
spec:
  selector:
    matchLabels:
      app: product-service
  endpoints:
    - port: http
      path: /metrics
      interval: 15s
```

Add port name to your Service:
```yaml
ports:
  - name: http      # ServiceMonitor references this
    port: 80
    targetPort: 8081
```

**K8s concept: ServiceMonitor (CRD)**
> ServiceMonitor is a Custom Resource Definition (CRD) from the Prometheus Operator. It tells Prometheus "scrape metrics from pods matching this selector, at this path, every 15s." No manual Prometheus config needed.

---

## Step 4: Install Loki for Logging

```bash
helm repo add grafana https://grafana.github.io/helm-charts

helm install loki grafana/loki-stack \
  -n monitoring \
  --set promtail.enabled=true \
  --set loki.persistence.enabled=true \
  --set loki.persistence.size=5Gi
```

**K8s concept: DaemonSet (Promtail)**
> Promtail runs as a DaemonSet — one pod per node. It tails container logs from `/var/log/pods/` and ships them to Loki. DaemonSets guarantee exactly one pod per node, perfect for log collectors, monitoring agents, and network plugins.

Add Loki as a data source in Grafana:
- URL: `http://loki:3100`
- Type: Loki

Now you can query logs in Grafana:
```
{namespace="shopstream", app="product-service"}
```

---

## Step 5: Add Distributed Tracing (Jaeger)

```bash
helm install jaeger bitnami/jaeger -n monitoring
```

Add OpenTelemetry tracing to your Go services:
```go
import "go.opentelemetry.io/otel"
// Initialize tracer, instrument HTTP handlers
```

When the Order Service calls the Product Service, the trace ID propagates through headers, giving you a full request timeline across services.

```bash
# Access Jaeger UI
kubectl port-forward svc/jaeger-query -n monitoring 16686:16686
```

---

## Step 6: Create Grafana Dashboards

Build dashboards for:
1. **Service Overview:** Request rate, error rate, latency (RED metrics) per service
2. **Infrastructure:** CPU, memory, disk per node
3. **Kubernetes:** Pod restarts, deployment status, HPA activity
4. **Business:** Orders per hour, popular products (from custom metrics)

---

## Step 7: Set Up Alerts

Create `monitoring/alerts/product-service-alerts.yaml`:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: product-service-alerts
  namespace: monitoring
  labels:
    release: monitoring
spec:
  groups:
    - name: product-service
      rules:
        - alert: HighErrorRate
          expr: |
            sum(rate(http_requests_total{app="product-service",status=~"5.."}[5m]))
            /
            sum(rate(http_requests_total{app="product-service"}[5m]))
            > 0.05
          for: 5m
          labels:
            severity: critical
          annotations:
            summary: "Product Service error rate > 5%"
        - alert: HighLatency
          expr: |
            histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{app="product-service"}[5m])) by (le))
            > 1
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "Product Service p95 latency > 1s"
```

---

## Exercises

1. **Generate load and watch dashboards:** See request rates spike in real-time
2. **Trigger an alert:** Make a service return 500s and watch the alert fire
3. **Trace a request:** Place an order and follow the trace through Order → Product → DB
4. **Query logs:** Find all error logs from the last hour across all services
5. **Check DaemonSet:** `kubectl get daemonset -n monitoring` — verify one promtail per node

---

## Checklist

- [ ] Prometheus scraping all services
- [ ] Grafana dashboards for services and infrastructure
- [ ] Loki collecting logs from all pods
- [ ] Jaeger showing distributed traces
- [ ] Alerts configured for error rate and latency
- [ ] Understand: Prometheus, Grafana, Loki, Jaeger, ServiceMonitor, PrometheusRule, DaemonSet
