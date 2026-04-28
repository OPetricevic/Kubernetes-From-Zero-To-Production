# ShopStream — Project Specification

## 1. Overview

ShopStream is a microservices-based e-commerce platform. The application itself is intentionally simple — the complexity lives in how it's deployed, scaled, observed, and operated on Kubernetes.

You will build 5 Go microservices, wire them together with databases and message queues, and progressively layer on every major Kubernetes concept across 9 phases.

---

## 2. Services

### 2.1 Product Service
- **Purpose:** CRUD for product catalog
- **Endpoints:**
  - `GET /api/products` — list products
  - `GET /api/products/{id}` — get single product
  - `POST /api/products` — create product (admin)
  - `PUT /api/products/{id}` — update product (admin)
  - `DELETE /api/products/{id}` — delete product (admin)
- **Database:** PostgreSQL (table: `products`)
- **Cache:** Redis (cache product listings)
- **Port:** 8081

### 2.2 Order Service
- **Purpose:** Place and track orders
- **Endpoints:**
  - `POST /api/orders` — place an order
  - `GET /api/orders/{id}` — get order details
  - `GET /api/orders/user/{userId}` — list user's orders
- **Database:** PostgreSQL (table: `orders`, `order_items`)
- **Publishes:** `order.placed` event to RabbitMQ
- **Port:** 8082

### 2.3 User Service
- **Purpose:** User registration and authentication
- **Endpoints:**
  - `POST /api/users/register` — register
  - `POST /api/users/login` — login (returns JWT)
  - `GET /api/users/me` — get profile (authenticated)
- **Database:** PostgreSQL (table: `users`)
- **Port:** 8083

### 2.4 Notification Service
- **Purpose:** Consume order events and send notifications
- **Consumes:** `order.placed` from RabbitMQ
- **Action:** Logs notification (simulates email/SMS)
- **No HTTP endpoints** — purely event-driven
- **Port:** 8084 (health check only)

### 2.5 Frontend
- **Purpose:** Simple React UI served via Nginx
- **Routes through Ingress** to backend APIs
- **Port:** 80

---

## 3. Data Models

### Product
```
id          UUID
name        string
description string
price       decimal
stock       integer
created_at  timestamp
updated_at  timestamp
```

### Order
```
id          UUID
user_id     UUID
status      string (pending, confirmed, shipped, delivered)
total       decimal
created_at  timestamp
```

### OrderItem
```
id          UUID
order_id    UUID
product_id  UUID
quantity    integer
price       decimal
```

### User
```
id          UUID
email       string (unique)
password    string (bcrypt hashed)
name        string
created_at  timestamp
```

---

## 4. Phase Breakdown

### Phase 1: Foundation
- **Build:** Product Service + PostgreSQL
- **K8s Focus:** Pods, Deployments, Services, ConfigMaps, Secrets, Namespaces
- **Deliverable:** Product API running in local cluster

### Phase 2: Multi-Service Architecture
- **Build:** Add Order, User, Frontend services
- **K8s Focus:** Service discovery, Ingress, resource limits, probes, init containers
- **Deliverable:** 4+ services communicating, single Ingress entry point

### Phase 3: State & Persistence
- **Build:** Persistent PostgreSQL, add Redis caching
- **K8s Focus:** PV, PVC, StorageClasses, StatefulSets, Headless Services
- **Deliverable:** Data survives pod restarts

### Phase 4: Async & Background Jobs
- **Build:** RabbitMQ, Notification Service, CronJob for reports
- **K8s Focus:** Jobs, CronJobs, Helm, Pod Disruption Budgets
- **Deliverable:** Async order processing pipeline

### Phase 5: Scaling & Resilience
- **Build:** Load testing, autoscaling, failure injection
- **K8s Focus:** HPA, VPA, anti-affinity, Network Policies, rolling updates
- **Deliverable:** Auto-scaling system that recovers from failures

### Phase 6: CI/CD & GitOps
- **Build:** Automated build/deploy pipeline
- **K8s Focus:** Kustomize, ArgoCD, image policies, canary deployments
- **Deliverable:** Git push triggers deployment

### Phase 7: Observability
- **Build:** Monitoring, logging, tracing stack
- **K8s Focus:** Prometheus, Grafana, Loki, Jaeger, DaemonSets, ServiceMonitors
- **Deliverable:** Full observability dashboards

### Phase 8: Security
- **Build:** Lock down the cluster
- **K8s Focus:** RBAC, ServiceAccounts, Pod Security, Sealed Secrets, Network Policies
- **Deliverable:** Least-privilege, zero-trust cluster

### Phase 9: Production Readiness
- **Build:** Move to managed K8s or multi-node simulation
- **K8s Focus:** Node pools, taints/tolerations, Cluster Autoscaler, TLS, cert-manager
- **Deliverable:** Production-grade cluster

---

## 5. Project Structure

```
shopstream/
├── docs/                          # All documentation
│   ├── README.md                  # Project overview
│   ├── spec.md                    # This file
│   ├── architecture.md            # System architecture
│   ├── k8s-concepts-map.md        # K8s concept tracker
│   ├── cheatsheet.md              # kubectl/Helm commands
│   ├── troubleshooting.md         # Common issues
│   └── phases/                    # Per-phase guides
│       ├── phase-1-foundation.md
│       ├── phase-2-multi-service.md
│       ├── phase-3-persistence.md
│       ├── phase-4-async.md
│       ├── phase-5-scaling.md
│       ├── phase-6-cicd.md
│       ├── phase-7-observability.md
│       ├── phase-8-security.md
│       └── phase-9-production.md
├── services/                      # Go microservices
│   ├── product-service/
│   ├── order-service/
│   ├── user-service/
│   ├── notification-service/
│   └── frontend/
├── k8s/                           # Kubernetes manifests
│   ├── base/                      # Base manifests (Kustomize)
│   │   ├── product-service/
│   │   ├── order-service/
│   │   ├── user-service/
│   │   ├── notification-service/
│   │   ├── frontend/
│   │   ├── postgres/
│   │   ├── redis/
│   │   ├── rabbitmq/
│   │   └── ingress/
│   └── overlays/                  # Environment overrides
│       ├── dev/
│       ├── staging/
│       └── prod/
├── helm/                          # Helm values (Phase 4+)
├── monitoring/                    # Prometheus/Grafana configs (Phase 7)
├── scripts/                       # Helper scripts
└── Makefile                       # Common commands
```

---

## 6. Non-Goals

- This is NOT about building a polished e-commerce app
- The Go code should be minimal and functional — just enough to exercise K8s concepts
- No frontend polish needed — a basic UI that calls the APIs is sufficient
- No payment processing, no real email sending
- Focus is 100% on Kubernetes operations and concepts
