# Phase 2: Multi-Service Architecture

## Goal
Add Order Service, User Service, and Frontend. Wire them together with Ingress. Add health probes and resource management.

## Prerequisites
- [ ] Phase 1 complete — Product Service running in cluster

---

## Step 1: Build the Order Service

Create `services/order-service/` — a Go HTTP server that:
- Listens on port 8082
- `POST /api/orders` — creates an order (calls Product Service internally to validate)
- `GET /api/orders/{id}` — get order by ID
- `GET /api/orders/user/{userId}` — list orders for a user
- Connects to PostgreSQL for order storage
- Calls Product Service via `http://product-service.shopstream.svc.cluster.local/api/products/{id}`

**K8s concept: Service Discovery**
> Services in K8s get DNS names automatically. From any pod in the cluster, you can reach another service at `<service-name>.<namespace>.svc.cluster.local`. The Order Service finds the Product Service this way — no hardcoded IPs.

---

## Step 2: Build the User Service

Create `services/user-service/` — a Go HTTP server that:
- Listens on port 8083
- `POST /api/users/register` — register user
- `POST /api/users/login` — login, return JWT
- `GET /api/users/me` — get profile (validate JWT)
- Connects to PostgreSQL

---

## Step 3: Build the Frontend

Create `services/frontend/` — a simple React app (or even static HTML) that:
- Lists products
- Has a login/register form
- Can place orders

Serve it via Nginx. Create `services/frontend/nginx.conf` and `Dockerfile`.

---

## Step 4: Containerize and Load All Services

```bash
docker build -t order-service:v1 services/order-service/
docker build -t user-service:v1 services/user-service/
docker build -t frontend:v1 services/frontend/

kind load docker-image order-service:v1 --name shopstream
kind load docker-image user-service:v1 --name shopstream
kind load docker-image frontend:v1 --name shopstream
```

---

## Step 5: Add Health Probes to All Services

Update every deployment to include probes:

```yaml
spec:
  containers:
    - name: product-service
      # ... existing config ...
      livenessProbe:
        httpGet:
          path: /health
          port: 8081
        initialDelaySeconds: 10
        periodSeconds: 15
        failureThreshold: 3
      readinessProbe:
        httpGet:
          path: /health
          port: 8081
        initialDelaySeconds: 5
        periodSeconds: 10
        failureThreshold: 3
```

**K8s concepts: Liveness Probe, Readiness Probe**
> - Liveness: "Is this container alive?" If it fails, K8s kills and restarts the pod.
> - Readiness: "Is this container ready to receive traffic?" If it fails, K8s removes it from the Service endpoints (no traffic routed to it).
> - Always implement both. A service that's alive but not ready (e.g., still connecting to DB) shouldn't get requests.

---

## Step 6: Add Resource Requests and Limits

```yaml
spec:
  containers:
    - name: product-service
      resources:
        requests:
          cpu: "100m"       # 0.1 CPU core guaranteed
          memory: "128Mi"   # 128 MB guaranteed
        limits:
          cpu: "250m"       # max 0.25 CPU core
          memory: "256Mi"   # max 256 MB (OOMKilled if exceeded)
```

**K8s concepts: Resource Requests, Resource Limits**
> - Requests: what the scheduler guarantees. Pod won't be scheduled if node can't provide this.
> - Limits: the ceiling. CPU gets throttled; memory gets OOMKilled.
> - Always set both. Without requests, the scheduler can overcommit nodes.

---

## Step 7: Add Init Container for DB Migration

```yaml
spec:
  initContainers:
    - name: db-migrate
      image: product-service:v1
      command: ["./product-service", "migrate"]
      env:
        # same DB env vars as main container
  containers:
    - name: product-service
      # ...
```

**K8s concept: Init Containers**
> Init containers run before the main container starts. They must complete successfully. Use them for:
> - Database migrations
> - Waiting for dependencies
> - Downloading config files

---

## Step 8: Install Ingress Controller

```bash
# For kind
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

# Wait for it
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s
```

---

## Step 9: Create Ingress Resource

Create `k8s/base/ingress/ingress.yaml`:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: shopstream-ingress
  namespace: shopstream
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
    - http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: frontend
                port:
                  number: 80
          - path: /api/products
            pathType: Prefix
            backend:
              service:
                name: product-service
                port:
                  number: 80
          - path: /api/orders
            pathType: Prefix
            backend:
              service:
                name: order-service
                port:
                  number: 80
          - path: /api/users
            pathType: Prefix
            backend:
              service:
                name: user-service
                port:
                  number: 80
```

**K8s concepts: Ingress, Ingress Controller**
> - Ingress Controller (NGINX): the actual reverse proxy running in the cluster
> - Ingress Resource: the routing rules ("send /api/products to product-service")
> - This replaces the need for port-forwarding — one entry point for everything

---

## Step 10: Deploy and Verify

```bash
kubectl apply -f k8s/base/order-service/
kubectl apply -f k8s/base/user-service/
kubectl apply -f k8s/base/frontend/
kubectl apply -f k8s/base/ingress/

# Test (kind with port mapping)
curl http://localhost/api/products
curl http://localhost/api/users/register -X POST -H "Content-Type: application/json" -d '{"email":"test@test.com","password":"pass123","name":"Test"}'
```

---

## Exercises

1. **Kill a pod and watch readiness:** Remove a product-service pod and observe the endpoints update
2. **Trigger OOMKill:** Set memory limit to 10Mi and watch what happens
3. **Break the liveness probe:** Make /health return 500 and watch K8s restart the pod
4. **Test service discovery:** Exec into the order-service pod and curl the product-service by DNS name

---

## Checklist

- [ ] Order Service, User Service, Frontend deployed
- [ ] All services have liveness and readiness probes
- [ ] All services have resource requests and limits
- [ ] Init container runs DB migrations
- [ ] Ingress routes traffic to all services
- [ ] Can access everything via `http://localhost`
- [ ] Understand: Service Discovery, Ingress, Probes, Resources, Init Containers
