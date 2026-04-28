# Phase 9: Production Readiness

## Goal
Move to a production-grade setup with TLS, node management, resource governance, and cluster autoscaling.

## Prerequisites
- [ ] Phase 8 complete
- [ ] (Optional) Cloud account for managed K8s (EKS/GKE/AKS) or continue with kind multi-node

---

## Step 1: Multi-Node Cluster Setup

### Option A: Managed Kubernetes (recommended for full experience)

```bash
# EKS
eksctl create cluster --name shopstream-prod --region us-east-1 --nodes 3 --node-type t3.medium

# GKE
gcloud container clusters create shopstream-prod --num-nodes=3 --machine-type=e2-medium

# AKS
az aks create --resource-group shopstream --name shopstream-prod --node-count 3 --node-vm-size Standard_B2s
```

### Option B: kind with multiple nodes
```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
    labels:
      node-type: app
  - role: worker
    labels:
      node-type: app
  - role: worker
    labels:
      node-type: data
```

---

## Step 2: Node Pools with Taints and Tolerations

Dedicate nodes for specific workloads:

```bash
# Taint data nodes — only data workloads can schedule here
kubectl taint nodes <data-node> workload=data:NoSchedule
```

Add tolerations to PostgreSQL/Redis:
```yaml
spec:
  template:
    spec:
      tolerations:
        - key: "workload"
          operator: "Equal"
          value: "data"
          effect: "NoSchedule"
      nodeSelector:
        node-type: data
```

**K8s concepts: Taints, Tolerations, Node Selectors**
> - Taint: "This node repels pods unless they tolerate me"
> - Toleration: "I can handle this taint, schedule me here"
> - Node Selector: "Only schedule me on nodes with this label"
> - Together they let you dedicate nodes: app nodes for services, data nodes for databases, GPU nodes for ML

---

## Step 3: Cluster Autoscaler

For managed K8s:
```bash
# EKS — install cluster autoscaler
helm repo add autoscaler https://kubernetes.github.io/autoscaler
helm install cluster-autoscaler autoscaler/cluster-autoscaler \
  --set autoDiscovery.clusterName=shopstream-prod \
  --set awsRegion=us-east-1 \
  -n kube-system
```

**K8s concept: Cluster Autoscaler**
> HPA scales pods. Cluster Autoscaler scales nodes. When HPA wants more pods but no node has capacity, Cluster Autoscaler adds a node. When nodes are underutilized, it removes them. This is how you handle variable load cost-effectively.

---

## Step 4: TLS with cert-manager

```bash
helm repo add jetstack https://charts.jetstack.io
helm install cert-manager jetstack/cert-manager \
  -n cert-manager --create-namespace \
  --set installCRDs=true
```

Create a ClusterIssuer:
```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
      - http01:
          ingress:
            class: nginx
```

Update Ingress for TLS:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: shopstream-ingress
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - shopstream.yourdomain.com
      secretName: shopstream-tls
  rules:
    - host: shopstream.yourdomain.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: frontend
                port:
                  number: 80
```

**K8s concept: cert-manager**
> cert-manager automates TLS certificate management. It requests certificates from Let's Encrypt, stores them as K8s Secrets, and auto-renews before expiry. No more manual certificate management.

---

## Step 5: Resource Quotas and LimitRanges

Create `k8s/base/resource-quota.yaml`:
```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: shopstream-quota
  namespace: shopstream
spec:
  hard:
    requests.cpu: "4"
    requests.memory: 8Gi
    limits.cpu: "8"
    limits.memory: 16Gi
    pods: "50"
    services: "20"
    persistentvolumeclaims: "10"
```

Create `k8s/base/limit-range.yaml`:
```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: shopstream-limits
  namespace: shopstream
spec:
  limits:
    - type: Container
      default:
        cpu: "200m"
        memory: "256Mi"
      defaultRequest:
        cpu: "100m"
        memory: "128Mi"
      max:
        cpu: "1"
        memory: "1Gi"
      min:
        cpu: "50m"
        memory: "64Mi"
```

**K8s concepts: ResourceQuota, LimitRange**
> - ResourceQuota: caps total resources a namespace can consume (prevents one team from eating the whole cluster)
> - LimitRange: sets default and max/min for individual containers (catches pods deployed without resource specs)

---

## Step 6: Priority Classes

```yaml
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: critical
value: 1000000
globalDefault: false
description: "Critical services that must always run"
---
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: standard
value: 100000
globalDefault: true
description: "Default priority for most workloads"
```

Assign to deployments:
```yaml
spec:
  template:
    spec:
      priorityClassName: critical   # for product-service, order-service
```

**K8s concept: Priority Classes**
> When the cluster is full, K8s needs to decide which pods to evict. Higher priority pods preempt lower priority ones. Critical services (your API) should have higher priority than batch jobs.

---

## Step 7: External DNS (Optional, requires cloud)

```bash
helm install external-dns bitnami/external-dns \
  --set provider=aws \
  --set domainFilters[0]=yourdomain.com \
  -n kube-system
```

**K8s concept: External DNS**
> Automatically creates DNS records when you create Ingress resources or LoadBalancer Services. No manual DNS management.

---

## Final Verification

```bash
# Everything running?
kubectl get all -A | grep shopstream

# TLS working?
curl -v https://shopstream.yourdomain.com

# Autoscaling working?
kubectl get hpa -n shopstream
kubectl get nodes

# Security?
kubectl auth can-i --list --as=system:serviceaccount:shopstream:product-service -n shopstream

# Observability?
# Check Grafana dashboards, Loki logs, Jaeger traces

# GitOps?
# Push a change, watch ArgoCD sync
```

---

## Exercises

1. **Exhaust the quota:** Try to deploy more pods than the ResourceQuota allows
2. **Test priority preemption:** Fill the cluster, then deploy a critical pod — watch it evict a standard pod
3. **Node failure:** Cordon and drain a node, watch pods reschedule and Cluster Autoscaler react
4. **Certificate renewal:** Check cert-manager logs, verify certificate expiry dates

---

## Checklist

- [ ] Multi-node cluster (managed or kind)
- [ ] Taints/tolerations separating app and data nodes
- [ ] Cluster Autoscaler (if on cloud)
- [ ] TLS via cert-manager
- [ ] Resource Quotas and LimitRanges
- [ ] Priority Classes assigned
- [ ] Understand: Taints, Tolerations, Cluster Autoscaler, cert-manager, ResourceQuota, LimitRange, Priority Classes

---

## Congratulations

You've built and operated a production-grade Kubernetes platform. You've touched:
- Every core workload type (Pod, Deployment, StatefulSet, DaemonSet, Job, CronJob)
- Networking (Services, Ingress, Network Policies, DNS)
- Storage (PV, PVC, StorageClass)
- Configuration (ConfigMap, Secret, Sealed Secrets)
- Scaling (HPA, VPA, Cluster Autoscaler)
- Security (RBAC, Pod Security, Security Contexts)
- Observability (Prometheus, Grafana, Loki, Jaeger)
- CI/CD (GitHub Actions, ArgoCD, Kustomize, Helm)
- Production ops (TLS, Resource Quotas, Priority Classes, Taints/Tolerations)

You didn't just learn Kubernetes — you operated it.
