# ShopStream — Architecture

## System Diagram

```
                    ┌─────────────────────────────────────────────┐
                    │              Kubernetes Cluster              │
                    │                                             │
   Internet ──────►│  ┌──────────┐                               │
                    │  │  Ingress │                               │
                    │  │ (NGINX)  │                               │
                    │  └────┬─────┘                               │
                    │       │                                     │
                    │  ┌────┴──────────────────────┐              │
                    │  │         Routes             │              │
                    │  │  /           → Frontend    │              │
                    │  │  /api/products → Product   │              │
                    │  │  /api/orders  → Order      │              │
                    │  │  /api/users   → User       │              │
                    │  └──┬────┬────┬──────────────┘              │
                    │     │    │    │                              │
                    │  ┌──▼┐ ┌─▼──┐ ┌▼────┐  ┌──────────────┐   │
                    │  │FE │ │Prod│ │Order│  │ Notification │   │
                    │  │   │ │Svc │ │Svc  │  │ Service      │   │
                    │  └───┘ └─┬──┘ └──┬──┘  └──────▲───────┘   │
                    │          │       │             │            │
                    │     ┌────▼───┐   │      ┌─────┴──────┐    │
                    │     │ Redis  │   │      │  RabbitMQ   │    │
                    │     │ Cache  │   │      │  (Events)   │    │
                    │     └────────┘   │      └─────▲──────┘    │
                    │                  │            │            │
                    │          ┌───────▼────────────┘            │
                    │          │                                  │
                    │     ┌────▼──────────┐                      │
                    │     │  PostgreSQL    │                      │
                    │     │  (StatefulSet) │                      │
                    │     └───────────────┘                      │
                    └─────────────────────────────────────────────┘
```

## Service Communication

| From | To | Method | Purpose |
|------|----|--------|---------|
| Frontend | Product Svc | HTTP (via Ingress) | Display products |
| Frontend | Order Svc | HTTP (via Ingress) | Place/view orders |
| Frontend | User Svc | HTTP (via Ingress) | Auth |
| Order Svc | Product Svc | HTTP (ClusterIP) | Validate product & stock |
| Order Svc | RabbitMQ | AMQP | Publish `order.placed` |
| Notification Svc | RabbitMQ | AMQP | Consume `order.placed` |
| Product Svc | PostgreSQL | TCP/5432 | Product data |
| Product Svc | Redis | TCP/6379 | Cache |
| Order Svc | PostgreSQL | TCP/5432 | Order data |
| User Svc | PostgreSQL | TCP/5432 | User data |

## Namespace Strategy

| Namespace | Contents |
|-----------|----------|
| `shopstream` | All application services |
| `shopstream-data` | PostgreSQL, Redis, RabbitMQ |
| `monitoring` | Prometheus, Grafana, Loki (Phase 7) |
| `argocd` | ArgoCD (Phase 6) |
| `ingress-nginx` | Ingress controller |

## Port Assignments

| Service | Container Port | Service Port |
|---------|---------------|-------------|
| Product Service | 8081 | 80 |
| Order Service | 8082 | 80 |
| User Service | 8083 | 80 |
| Notification Service | 8084 | 80 |
| Frontend | 80 | 80 |
| PostgreSQL | 5432 | 5432 |
| Redis | 6379 | 6379 |
| RabbitMQ | 5672 / 15672 | 5672 / 15672 |
