# Phase 4: Async Communication & Background Jobs

## Goal
Add RabbitMQ for event-driven communication. Build the Notification Service. Create CronJobs for scheduled tasks. Learn Helm.

## Prerequisites
- [ ] Phase 3 complete — persistent storage working

---

## Step 1: Install RabbitMQ via Helm

This is your introduction to Helm — the package manager for Kubernetes.

```bash
# Add the Bitnami repo
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Create a values file
```

Create `helm/rabbitmq-values.yaml`:
```yaml
auth:
  username: shopstream
  password: shopstream123
persistence:
  enabled: true
  size: 2Gi
resources:
  requests:
    cpu: 100m
    memory: 256Mi
  limits:
    cpu: 250m
    memory: 512Mi
```

```bash
helm install rabbitmq bitnami/rabbitmq \
  -n shopstream-data \
  -f helm/rabbitmq-values.yaml

# Check it
kubectl get pods -n shopstream-data
helm list -n shopstream-data
```

**K8s concepts: Helm, Helm Charts, Helm Values**
> - Helm Chart: a package of K8s manifests with templating (like apt/brew for K8s)
> - Values file: your customizations (passwords, resource limits, storage size)
> - `helm install` renders the templates with your values and applies them
> - Charts handle complex setups (StatefulSets, Services, ConfigMaps, RBAC) that would be tedious to write manually

---

## Step 2: Build the Notification Service

Create `services/notification-service/` — a Go service that:
- Connects to RabbitMQ
- Consumes messages from `order.placed` queue
- Logs the notification (simulates sending email)
- Has a `GET /health` endpoint for probes
- Listens on port 8084

This service has NO HTTP API endpoints — it's purely event-driven.

---

## Step 3: Update Order Service to Publish Events

When an order is placed (`POST /api/orders`), publish a message to RabbitMQ:
```json
{
  "event": "order.placed",
  "order_id": "uuid",
  "user_id": "uuid",
  "total": 29.99,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

Add RabbitMQ env vars to Order Service deployment:
```yaml
env:
  - name: RABBITMQ_HOST
    value: "rabbitmq.shopstream-data.svc.cluster.local"
  - name: RABBITMQ_PORT
    value: "5672"
  - name: RABBITMQ_USER
    valueFrom:
      secretKeyRef:
        name: rabbitmq-secret
        key: RABBITMQ_USER
  - name: RABBITMQ_PASSWORD
    valueFrom:
      secretKeyRef:
        name: rabbitmq-secret
        key: RABBITMQ_PASSWORD
```

---

## Step 4: Deploy Notification Service

Create K8s manifests in `k8s/base/notification-service/`. Same pattern as other services but note: this service doesn't need to be in the Ingress (no external traffic).

---

## Step 5: Create a CronJob

Create `k8s/base/jobs/sales-report-cronjob.yaml`:
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: daily-sales-report
  namespace: shopstream
spec:
  schedule: "0 2 * * *"    # 2 AM daily
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: report
              image: order-service:v1
              command: ["./order-service", "report"]
              env:
                - name: DB_HOST
                  value: "postgres.shopstream-data.svc.cluster.local"
                # ... other DB env vars
          restartPolicy: OnFailure
      backoffLimit: 3
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
```

**K8s concepts: Job, CronJob**
> - Job: runs a container to completion (not forever like a Deployment). Retries on failure.
> - CronJob: creates Jobs on a schedule (cron syntax). Great for reports, cleanup, backups.
> - `restartPolicy: OnFailure` — retry the container if it fails
> - `backoffLimit: 3` — give up after 3 failures

---

## Step 6: Add Pod Disruption Budget

Create `k8s/base/product-service/pdb.yaml`:
```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: product-service-pdb
  namespace: shopstream
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: product-service
```

**K8s concept: Pod Disruption Budget (PDB)**
> PDBs tell K8s "during voluntary disruptions (node drain, cluster upgrade), keep at least N pods running." Without a PDB, a node drain could kill all your replicas at once.

---

## Step 7: Test the Async Pipeline

```bash
# Place an order
curl -X POST http://localhost/api/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id":"some-uuid","items":[{"product_id":"some-uuid","quantity":2}]}'

# Check notification service logs
kubectl logs -l app=notification-service -n shopstream -f
# Should see: "Notification: Order <id> placed for user <id>, total: $X.XX"

# Trigger CronJob manually for testing
kubectl create job --from=cronjob/daily-sales-report manual-report -n shopstream
kubectl get jobs -n shopstream
kubectl logs job/manual-report -n shopstream
```

---

## Exercises

1. **Explore the Helm release:** `helm get manifest rabbitmq -n shopstream-data` — see all the K8s resources Helm created
2. **Upgrade RabbitMQ:** Change a value and run `helm upgrade`
3. **Scale notification consumers:** Set replicas to 3 — multiple consumers share the queue
4. **Test PDB:** Try draining a node: `kubectl drain <node> --ignore-daemonsets` — observe PDB preventing all pods from dying

---

## Checklist

- [ ] RabbitMQ installed via Helm
- [ ] Order Service publishes events to RabbitMQ
- [ ] Notification Service consumes events
- [ ] CronJob runs daily sales report
- [ ] PDB protects critical services
- [ ] Understand: Helm, Jobs, CronJobs, PDB, event-driven architecture in K8s
