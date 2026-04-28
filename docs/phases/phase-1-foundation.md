# Phase 1: Foundation

## Goal
Get a single Go microservice (Product Service) running in a local Kubernetes cluster with a PostgreSQL database.

## Prerequisites
- [ ] Docker installed and running
- [ ] kubectl installed (`kubectl version --client`)
- [ ] kind or minikube installed
- [ ] Go 1.21+ installed
- [ ] A code editor

---

## Step 1: Create Your Local Cluster

### Option A: kind (recommended)

Create `kind-config.yaml` in project root:
```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraConfig:
            nodeStatusUpdateFrequency: 10s
    extraPortMappings:
      - containerPort: 80
        hostPort: 80
        protocol: TCP
      - containerPort: 443
        hostPort: 443
        protocol: TCP
  - role: worker
  - role: worker
```

```bash
kind create cluster --name shopstream --config kind-config.yaml
kubectl cluster-info --context kind-shopstream
```

### Option B: minikube
```bash
minikube start --profile shopstream --cpus 4 --memory 8192 --nodes 3
```

**K8s concept: Cluster, Nodes**
> A cluster is a set of nodes (machines) that run containerized applications. You just created a control plane node and 2 worker nodes.

---

## Step 2: Create Namespaces

```bash
kubectl create namespace shopstream
kubectl create namespace shopstream-data
kubectl config set-context --current --namespace=shopstream
```

Create `k8s/base/namespaces.yaml`:
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: shopstream
  labels:
    project: shopstream
---
apiVersion: v1
kind: Namespace
metadata:
  name: shopstream-data
  labels:
    project: shopstream
```

**K8s concept: Namespaces**
> Namespaces are virtual clusters within a cluster. They provide isolation and organization. We separate app services from data services.

---

## Step 3: Write the Product Service

Create a minimal Go HTTP server. The code should:
- Listen on port 8081
- Have a `GET /health` endpoint (returns 200)
- Have a `GET /api/products` endpoint (returns JSON list from DB)
- Have a `POST /api/products` endpoint (inserts into DB)
- Connect to PostgreSQL using env vars: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`

Keep it simple — use `net/http` and `lib/pq`. No frameworks needed.

**Directory:** `services/product-service/`

---

## Step 4: Containerize It

Create `services/product-service/Dockerfile`:
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o product-service .

# Run stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/product-service .
EXPOSE 8081
CMD ["./product-service"]
```

Build and load into kind:
```bash
docker build -t product-service:v1 services/product-service/
kind load docker-image product-service:v1 --name shopstream
```

---

## Step 5: Deploy PostgreSQL

Create `k8s/base/postgres/deployment.yaml`:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: shopstream-data
  labels:
    app: postgres
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
        - name: postgres
          image: postgres:16-alpine
          ports:
            - containerPort: 5432
          env:
            - name: POSTGRES_DB
              valueFrom:
                configMapKeyRef:
                  name: postgres-config
                  key: POSTGRES_DB
            - name: POSTGRES_USER
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: POSTGRES_USER
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: POSTGRES_PASSWORD
```

Create `k8s/base/postgres/service.yaml`:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: shopstream-data
spec:
  selector:
    app: postgres
  ports:
    - port: 5432
      targetPort: 5432
  type: ClusterIP
```

**K8s concepts: Deployment, Pod, Service (ClusterIP), Labels & Selectors**
> - A Deployment tells K8s "I want N replicas of this container running at all times"
> - It creates a ReplicaSet which creates Pods
> - A ClusterIP Service gives a stable internal DNS name (`postgres.shopstream-data.svc.cluster.local`)
> - Labels connect everything: Deployment → Pod template → Service selector

---

## Step 6: Create ConfigMap and Secret

Create `k8s/base/postgres/configmap.yaml`:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: postgres-config
  namespace: shopstream-data
data:
  POSTGRES_DB: shopstream
```

Create `k8s/base/postgres/secret.yaml`:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: shopstream-data
type: Opaque
data:
  POSTGRES_USER: c2hvcHN0cmVhbQ==        # base64 of "shopstream"
  POSTGRES_PASSWORD: c2hvcHN0cmVhbTEyMw== # base64 of "shopstream123"
```

To encode: `echo -n "shopstream" | base64`

**K8s concepts: ConfigMap, Secret**
> - ConfigMaps hold non-sensitive config (DB name, ports, feature flags)
> - Secrets hold sensitive data (passwords, API keys) — base64 encoded (not encrypted by default!)
> - Both are injected into pods as env vars or mounted as files

---

## Step 7: Deploy the Product Service

Create `k8s/base/product-service/deployment.yaml`:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: product-service
  namespace: shopstream
  labels:
    app: product-service
spec:
  replicas: 2
  selector:
    matchLabels:
      app: product-service
  template:
    metadata:
      labels:
        app: product-service
    spec:
      containers:
        - name: product-service
          image: product-service:v1
          imagePullPolicy: Never    # use local image
          ports:
            - containerPort: 8081
          env:
            - name: DB_HOST
              value: "postgres.shopstream-data.svc.cluster.local"
            - name: DB_PORT
              value: "5432"
            - name: DB_NAME
              valueFrom:
                configMapKeyRef:
                  name: product-config
                  key: DB_NAME
            - name: DB_USER
              valueFrom:
                secretKeyRef:
                  name: product-secret
                  key: DB_USER
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: product-secret
                  key: DB_PASSWORD
```

Create `k8s/base/product-service/service.yaml`:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: product-service
  namespace: shopstream
spec:
  selector:
    app: product-service
  ports:
    - port: 80
      targetPort: 8081
  type: ClusterIP
```

---

## Step 8: Apply Everything and Verify

```bash
# Apply in order
kubectl apply -f k8s/base/namespaces.yaml
kubectl apply -f k8s/base/postgres/
kubectl apply -f k8s/base/product-service/

# Wait for pods
kubectl get pods -n shopstream-data -w
kubectl get pods -n shopstream -w

# Check logs
kubectl logs -l app=product-service -n shopstream

# Test via port-forward
kubectl port-forward svc/product-service 8081:80 -n shopstream
# In another terminal:
curl http://localhost:8081/api/products
curl -X POST http://localhost:8081/api/products -H "Content-Type: application/json" -d '{"name":"Test Product","price":9.99,"stock":100}'
```

---

## Exercises

1. **Break it on purpose:** Delete a product-service pod and watch K8s recreate it
   ```bash
   kubectl delete pod -l app=product-service -n shopstream
   kubectl get pods -n shopstream -w
   ```

2. **Scale manually:** Change replicas to 5 and observe
   ```bash
   kubectl scale deployment product-service --replicas=5 -n shopstream
   kubectl get pods -n shopstream
   ```

3. **Inspect the ReplicaSet:** See the intermediate object K8s creates
   ```bash
   kubectl get replicaset -n shopstream
   kubectl describe replicaset -l app=product-service -n shopstream
   ```

4. **Check service endpoints:** Verify the Service knows about all pod IPs
   ```bash
   kubectl get endpoints product-service -n shopstream
   ```

---

## Checklist

- [ ] Local cluster running with 3 nodes
- [ ] Namespaces created (shopstream, shopstream-data)
- [ ] PostgreSQL running in shopstream-data
- [ ] Product Service running in shopstream (2 replicas)
- [ ] ConfigMap and Secret created and consumed
- [ ] Can hit the API via port-forward
- [ ] Understand: Pod, Deployment, ReplicaSet, Service, ConfigMap, Secret, Namespace, Labels
