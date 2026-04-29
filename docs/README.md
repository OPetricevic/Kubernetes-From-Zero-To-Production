# Documentation

## Start Here

| Document | Description |
|----------|-------------|
| [Why Kubernetes?](./why-kubernetes.md) | The real-world problems K8s solves and why each feature exists |
| [How It All Connects](./how-it-all-connects.md) | Code → Docker → Kubernetes → CI/CD — the full picture |
| [Architecture](./architecture.md) | System diagram, service interactions, namespace strategy |
| [Project Spec](./spec.md) | Full project specification |

## Phase Guides

Work through these in order. Each phase builds on the previous one.

| Phase | Guide |
|-------|-------|
| 1 | [Foundation](./phases/phase-1-foundation.md) — Single service + database |
| 2 | [Multi-Service](./phases/phase-2-multi-service.md) — Service discovery + Ingress |
| 3 | [Persistence](./phases/phase-3-persistence.md) — StatefulSets + Redis caching |
| 4 | [Async](./phases/phase-4-async.md) — RabbitMQ + event-driven messaging |
| 5 | [Scaling](./phases/phase-5-scaling.md) — HPA + network policies + rollbacks |
| 6 | [CI/CD](./phases/phase-6-cicd.md) — Kustomize + ArgoCD + GitHub Actions |
| 7 | [Observability](./phases/phase-7-observability.md) — Prometheus + Grafana + Loki |
| 8 | [Security](./phases/phase-8-security.md) — RBAC + pod security + sealed secrets |
| 9 | [Production](./phases/phase-9-production.md) — TLS + cluster autoscaler + quotas |
| 10 | [Live Deployment](./phases/phase-10-live-deployment.md) — Oracle Cloud + public endpoint |

## Reference

| Document | Description |
|----------|-------------|
| [K8s Concepts Map](./k8s-concepts-map.md) | Every K8s concept mapped to the phase where you learn it |
| [Cheatsheet](./cheatsheet.md) | kubectl and Helm commands |
| [Troubleshooting](./troubleshooting.md) | Common issues and how to debug them |
| [Windows Setup](./setup-guide-windows.md) | Installing prerequisites on Windows |
