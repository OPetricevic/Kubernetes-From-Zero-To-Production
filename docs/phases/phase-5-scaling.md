# Phase 5: Scaling & Resilience

## Goal
Make the system auto-scale under load, survive failures, and handle deployments gracefully.

## Prerequisites
- [ ] Phase 4 complete — async pipeline working
- [ ] Install metrics-server (required for HPA)

---

## Step 1: Install Metrics Server

HPA needs metrics to make scaling decisions.

```bash
# For kind
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# For kind, you may need to patch it to work without TLS
kubectl patch deployment metrics-server -n kube-system --type='json' \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'

# Verify
kubectl top nodes
kubectl top pods -n shopstream
```

---

## Step 2: Create Horizontal Pod Autoscaler

Create `k8s/base/product-service/hpa.yaml`:
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: product-service-hpa
  namespace: shopstream
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: product-service
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 30
      policies:
        - type: Pods
          value: 2
          periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Pods
          value: 1
          periodSeconds: 120
```

**K8s concept: Horizontal Pod Autoscaler (HPA)**
> HPA watches metrics (CPU, memory, custom) and adjusts replica count automatically.
> - `averageUtilization: 70` — scale up when average CPU across pods exceeds 70% of requests
> - `behavior` controls how fast scaling happens (prevent flapping)
> - Scale up fast (30s window), scale down slow (300s window) — this is production best practice

---

## Step 3: Load Test and Watch Scaling

```bash
# Watch HPA in one terminal
kubectl get hpa -n shopstream -w

# Watch pods in another terminal
kubectl get pods -n shopstream -w

# Generate load (from inside the cluster)
kubectl run load-generator --image=busybox --rm -it -- /bin/sh
# Inside the pod:
while true; do wget -q -O- http://product-service.shopstream.svc.cluster.local/api/products; done
```

---

## Step 4: Add Pod Anti-Affinity

Spread replicas across nodes so a node failure doesn't kill all instances:

```yaml
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchExpressions:
                    - key: app
                      operator: In
                      values:
                        - product-service
                topologyKey: kubernetes.io/hostname
```

**K8s concept: Pod Anti-Affinity**
> "Don't put two product-service pods on the same node." Using `preferred` (soft) instead of `required` (hard) means K8s will try but won't fail scheduling if it can't.

---

## Step 5: Add Network Policies

Create `k8s/base/network-policies/default-deny.yaml`:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-all
  namespace: shopstream
spec:
  podSelector: {}
  policyTypes:
    - Ingress
    - Egress
```

Create `k8s/base/network-policies/allow-product-service.yaml`:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-product-service
  namespace: shopstream
spec:
  podSelector:
    matchLabels:
      app: product-service
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              project: shopstream
          podSelector:
            matchLabels:
              app: order-service
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: ingress-nginx
      ports:
        - port: 8081
  egress:
    - to:
        - namespaceSelector:
            matchLabels:
              project: shopstream
          podSelector:
            matchLabels:
              app: postgres
      ports:
        - port: 5432
    - to:
        - namespaceSelector:
            matchLabels:
              project: shopstream
          podSelector:
            matchLabels:
              app: redis
      ports:
        - port: 6379
    - ports:    # DNS
        - port: 53
          protocol: UDP
        - port: 53
          protocol: TCP
```

**K8s concept: Network Policies**
> Default-deny + explicit allow = zero-trust networking. Each service can only talk to what it needs. This is how production clusters should work.

---

## Step 6: Configure Rolling Updates

```yaml
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0       # never reduce below desired count
      maxSurge: 1             # add 1 extra pod during update
  template:
    spec:
      terminationGracePeriodSeconds: 30
      containers:
        - name: product-service
          lifecycle:
            preStop:
              exec:
                command: ["/bin/sh", "-c", "sleep 10"]
```

**K8s concepts: Rolling Update, Lifecycle Hooks**
> - `maxUnavailable: 0` + `maxSurge: 1`: new pod starts, becomes ready, then old pod terminates. Zero downtime.
> - `preStop` hook: gives in-flight requests time to complete before the pod shuts down
> - `terminationGracePeriodSeconds`: how long K8s waits before force-killing

---

## Step 7: Practice Rollbacks

```bash
# Deploy a "bad" version
kubectl set image deployment/product-service product-service=product-service:v2-broken -n shopstream

# Watch it fail
kubectl rollout status deployment/product-service -n shopstream

# Check history
kubectl rollout history deployment/product-service -n shopstream

# Rollback
kubectl rollout undo deployment/product-service -n shopstream

# Rollback to specific revision
kubectl rollout undo deployment/product-service --to-revision=1 -n shopstream
```

---

## Exercises

1. **Observe HPA decisions:** `kubectl describe hpa product-service-hpa -n shopstream` — see the scaling events
2. **Test network policy:** Exec into a pod that shouldn't have access and try to curl a blocked service
3. **Simulate node failure:** `kubectl cordon <node>` then `kubectl drain <node>` — watch pods reschedule
4. **Test graceful shutdown:** Watch logs during a rolling update to confirm no dropped requests

---

## Checklist

- [ ] HPA configured and tested under load
- [ ] Pod anti-affinity spreading replicas across nodes
- [ ] Network policies enforcing zero-trust
- [ ] Rolling updates with zero downtime
- [ ] Rollback tested successfully
- [ ] Understand: HPA, Anti-Affinity, Network Policies, Rolling Updates, Rollbacks, Lifecycle Hooks
