# Kubernetes Concepts Map

Track your progress. Check off each concept as you implement it.
Concepts marked with 💡 are ones you encounter but don't configure directly — you should understand what they are and how they fit.

## Core Workloads

| Concept | Phase | Where You Use It | Done |
|---------|-------|-------------------|------|
| Pod | 1 | Every service runs as a Pod | ☐ |
| Multi-container Pod | 2 | Sidecar pattern (log forwarder, proxy) | ☐ |
| Deployment | 1 | Product Service deployment | ☐ |
| ReplicaSet | 1 | 💡 Managed by Deployments — you rarely touch directly | ☐ |
| StatefulSet | 3 | PostgreSQL with stable identity | ☐ |
| DaemonSet | 7 | Log collectors on every node | ☐ |
| Job | 4 | One-off DB migration | ☐ |
| CronJob | 4 | Daily sales report | ☐ |

## Networking

| Concept | Phase | Where You Use It | Done |
|---------|-------|-------------------|------|
| ClusterIP Service | 1 | Internal service-to-service | ☐ |
| NodePort Service | 1 | Quick local testing | ☐ |
| LoadBalancer Service | 9 | 💡 Cloud-provided external IP (EKS/GKE/AKS) | ☐ |
| ExternalName Service | — | 💡 DNS alias to external service (e.g., managed RDS) | ☐ |
| Headless Service | 3 | PostgreSQL StatefulSet DNS | ☐ |
| Ingress | 2 | Route external traffic to services | ☐ |
| Ingress Controller | 2 | NGINX Ingress Controller | ☐ |
| Gateway API | — | 💡 Next-gen replacement for Ingress (know it exists) | ☐ |
| Network Policies | 5 | Isolate namespaces | ☐ |
| DNS / Service Discovery | 2 | Services find each other by name | ☐ |
| CoreDNS | 2 | 💡 The DNS server inside every cluster — powers service discovery | ☐ |

## Configuration & Secrets

| Concept | Phase | Where You Use It | Done |
|---------|-------|-------------------|------|
| ConfigMap | 1 | App config (DB host, ports) | ☐ |
| ConfigMap as Volume | 3 | Mount config files into pods | ☐ |
| Secret | 1 | DB passwords, JWT signing key | ☐ |
| Secret Types | 1 | 💡 Opaque, docker-registry, tls — different use cases | ☐ |
| Sealed Secrets | 8 | Encrypt secrets in Git | ☐ |
| External Secrets | 8 | Pull from cloud secret manager | ☐ |
| Environment Variables | 1 | Inject config into containers | ☐ |
| Downward API | — | 💡 Expose pod metadata (name, namespace, IP) as env vars | ☐ |

## Storage

| Concept | Phase | Where You Use It | Done |
|---------|-------|-------------------|------|
| PersistentVolume (PV) | 3 | PostgreSQL data directory | ☐ |
| PersistentVolumeClaim (PVC) | 3 | Request storage for Postgres | ☐ |
| StorageClass | 3 | Define storage provisioner | ☐ |
| Dynamic Provisioning | 3 | 💡 PVs created automatically when PVC is made | ☐ |
| Access Modes | 3 | 💡 ReadWriteOnce, ReadOnlyMany, ReadWriteMany | ☐ |
| Reclaim Policies | 3 | 💡 Retain vs Delete — what happens to PV when PVC is deleted | ☐ |
| emptyDir | 2 | Temp shared volume in multi-container pod | ☐ |
| hostPath | — | 💡 Mount host filesystem into pod (avoid in production) | ☐ |

## Scaling & Scheduling

| Concept | Phase | Where You Use It | Done |
|---------|-------|-------------------|------|
| Horizontal Pod Autoscaler | 5 | Scale Product Service on CPU | ☐ |
| Vertical Pod Autoscaler | 5 | Right-size resource requests | ☐ |
| Cluster Autoscaler | 9 | Add/remove nodes | ☐ |
| Resource Requests | 2 | Guarantee CPU/memory per pod | ☐ |
| Resource Limits | 2 | Cap CPU/memory per pod | ☐ |
| QoS Classes | 2 | 💡 Guaranteed/Burstable/BestEffort — determines eviction order | ☐ |
| Pod Affinity/Anti-Affinity | 5 | Spread replicas across nodes | ☐ |
| Taints & Tolerations | 9 | Dedicated nodes for databases | ☐ |
| Node Selectors | 9 | Pin workloads to node types | ☐ |
| Node Affinity | 9 | 💡 More expressive version of nodeSelector | ☐ |
| Priority Classes | 9 | Critical pods get scheduled first | ☐ |
| Preemption | 9 | 💡 High-priority pod evicts low-priority pod | ☐ |
| Pod Disruption Budgets | 4 | Minimum available during updates | ☐ |
| Pod Topology Spread | 5 | 💡 Distribute pods evenly across zones/nodes | ☐ |

## Health & Lifecycle

| Concept | Phase | Where You Use It | Done |
|---------|-------|-------------------|------|
| Liveness Probe | 2 | Restart unhealthy pods | ☐ |
| Readiness Probe | 2 | Don't route to unready pods | ☐ |
| Startup Probe | 2 | Slow-starting containers | ☐ |
| Init Containers | 2 | Run DB migrations before app starts | ☐ |
| Sidecar Containers | 7 | Log forwarder, service mesh proxy | ☐ |
| Lifecycle Hooks | 5 | Graceful shutdown (preStop) | ☐ |
| terminationGracePeriodSeconds | 5 | How long K8s waits before force-killing | ☐ |
| Pod Phases | 1 | 💡 Pending → Running → Succeeded/Failed | ☐ |
| Container States | 1 | 💡 Waiting, Running, Terminated — and why | ☐ |
| Restart Policies | 4 | 💡 Always (Deployments), OnFailure (Jobs), Never | ☐ |

## Deployment Strategies

| Concept | Phase | Where You Use It | Done |
|---------|-------|-------------------|------|
| Rolling Update | 5 | Default deployment strategy | ☐ |
| Recreate Strategy | — | 💡 Kill all old pods, then start new (downtime, but simple) | ☐ |
| Rollback | 5 | Revert bad deployment | ☐ |
| Revision History | 5 | 💡 K8s keeps N old ReplicaSets for rollback | ☐ |
| Blue/Green | 6 | Zero-downtime switch | ☐ |
| Canary | 6 | Gradual traffic shift (Argo Rollouts) | ☐ |

## Security

| Concept | Phase | Where You Use It | Done |
|---------|-------|-------------------|------|
| RBAC (Roles, RoleBindings) | 8 | Limit who can do what | ☐ |
| ClusterRole / ClusterRoleBinding | 8 | 💡 Cluster-wide permissions (vs namespace-scoped Role) | ☐ |
| ServiceAccounts | 8 | Pod identity | ☐ |
| Pod Security Standards | 8 | Restrict pod capabilities | ☐ |
| Pod Security Admission | 8 | Enforce/warn/audit security levels per namespace | ☐ |
| Security Contexts | 8 | Run as non-root, read-only FS | ☐ |
| Resource Quotas | 9 | Limit namespace resource usage | ☐ |
| LimitRanges | 9 | Default limits per namespace | ☐ |
| Image Pull Secrets | 6 | 💡 Credentials for private container registries | ☐ |

## Package Management & Config

| Concept | Phase | Where You Use It | Done |
|---------|-------|-------------------|------|
| Kustomize | 6 | Manage dev/staging/prod overlays | ☐ |
| Helm | 4 | Install RabbitMQ, Prometheus | ☐ |
| Helm Values | 4 | Customize chart installations | ☐ |
| Helm Hooks | — | 💡 Pre-install/post-install jobs in Helm charts | ☐ |

## Observability

| Concept | Phase | Where You Use It | Done |
|---------|-------|-------------------|------|
| Prometheus | 7 | Metrics collection | ☐ |
| Grafana | 7 | Dashboards | ☐ |
| Loki | 7 | Log aggregation | ☐ |
| Jaeger/Tempo | 7 | Distributed tracing | ☐ |
| ServiceMonitor (CRD) | 7 | Auto-discover scrape targets | ☐ |
| PrometheusRule (CRD) | 7 | Alerting rules | ☐ |
| Metrics Server | 5 | 💡 Provides CPU/memory metrics for HPA and `kubectl top` | ☐ |

## GitOps & CI/CD

| Concept | Phase | Where You Use It | Done |
|---------|-------|-------------------|------|
| ArgoCD | 6 | Sync cluster state from Git | ☐ |
| ArgoCD Application CRD | 6 | Define what to deploy | ☐ |
| Argo Rollouts | 6 | Canary/blue-green deployments | ☐ |
| Image Pull Policies | 6 | Control when images are pulled | ☐ |

## Cluster Architecture (💡 understand, don't configure)

| Concept | Phase | Where You Encounter It | Done |
|---------|-------|------------------------|------|
| Control Plane | 1 | 💡 The "brain" — API server, scheduler, controller manager, etcd | ☐ |
| API Server (kube-apiserver) | 1 | 💡 Every kubectl command talks to this | ☐ |
| etcd | 1 | 💡 The database that stores all cluster state | ☐ |
| Scheduler (kube-scheduler) | 2 | 💡 Decides which node a pod runs on | ☐ |
| Controller Manager | 1 | 💡 Runs control loops (Deployment controller, ReplicaSet controller) | ☐ |
| kubelet | 1 | 💡 Agent on every node — actually runs your containers | ☐ |
| kube-proxy | 2 | 💡 Handles networking rules on each node (Service → Pod routing) | ☐ |
| Container Runtime | 1 | 💡 containerd or CRI-O — actually pulls and runs images | ☐ |
| CNI (Container Network Interface) | 5 | 💡 Plugin that provides pod networking (Calico, Cilium, Flannel) | ☐ |
| CSI (Container Storage Interface) | 3 | 💡 Plugin that provides storage (EBS CSI, GCE PD CSI) | ☐ |

## Cluster Operations

| Concept | Phase | Where You Use It | Done |
|---------|-------|-------------------|------|
| Namespaces | 1 | Organize workloads | ☐ |
| Labels & Selectors | 1 | Connect Deployments to Pods to Services | ☐ |
| Annotations | 2 | Ingress config, Prometheus scrape | ☐ |
| kubectl contexts | 1 | Switch between clusters (dev/prod) | ☐ |
| kubeconfig | 1 | 💡 The file that stores cluster credentials (~/.kube/config) | ☐ |
| cert-manager | 9 | Automatic TLS certificates | ☐ |
| External DNS | 9 | Auto-create DNS records | ☐ |
| Node Drain / Cordon | 5 | Safely remove a node from service | ☐ |
| kubectl debug | — | 💡 Attach ephemeral debug container to running pod | ☐ |

## Extensibility (💡 know these exist)

| Concept | Phase | Where You Encounter It | Done |
|---------|-------|------------------------|------|
| Custom Resource Definitions (CRDs) | 7 | ServiceMonitor, PrometheusRule, ArgoCD Application | ☐ |
| Operators | 7 | 💡 Prometheus Operator, cert-manager — automate complex apps | ☐ |
| Admission Controllers | 8 | 💡 Intercept API requests (Pod Security Admission is one) | ☐ |
| Webhooks (Validating/Mutating) | 8 | 💡 Custom logic that runs when resources are created/updated | ☐ |
| Service Mesh (Istio/Linkerd) | — | 💡 Advanced traffic management, mTLS between services | ☐ |
