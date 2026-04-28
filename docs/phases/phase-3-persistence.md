# Phase 3: State & Persistence

## Goal
Replace the throwaway PostgreSQL Deployment with a proper StatefulSet using persistent storage. Add Redis for caching.

## Prerequisites
- [ ] Phase 2 complete — multiple services running with Ingress

---

## Step 1: Understand the Problem

Right now, if the PostgreSQL pod restarts, all data is lost. That's because the data lives inside the container's filesystem, which is ephemeral.

```bash
# Prove it: insert some data, then delete the pod
curl -X POST http://localhost/api/products -H "Content-Type: application/json" -d '{"name":"Will I Survive?","price":1.00,"stock":1}'
kubectl delete pod -l app=postgres -n shopstream-data
# Wait for new pod, then:
curl http://localhost/api/products
# Data is gone.
```

---

## Step 2: Create a StorageClass (if needed)

kind and minikube come with a default StorageClass. Check:
```bash
kubectl get storageclass
```

If you need one:
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: standard
provisioner: rancher.io/local-path   # kind default
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
```

**K8s concept: StorageClass**
> StorageClass defines what kind of storage to provision (SSD, HDD, network-attached). The provisioner handles creating the actual volume. In production, this maps to cloud provider storage (EBS, GCE PD, Azure Disk).

---

## Step 3: Convert PostgreSQL to StatefulSet

Replace `k8s/base/postgres/deployment.yaml` with a StatefulSet:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: shopstream-data
spec:
  serviceName: postgres
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
            - name: PGDATA
              value: /var/lib/postgresql/data/pgdata
          volumeMounts:
            - name: postgres-storage
              mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
    - metadata:
        name: postgres-storage
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 5Gi
```

**K8s concepts: StatefulSet, PersistentVolumeClaim, PersistentVolume**
> - StatefulSet: like a Deployment but for stateful workloads. Pods get stable names (postgres-0, postgres-1) and stable storage.
> - volumeClaimTemplates: each pod gets its own PVC automatically
> - PVC: "I need 5Gi of ReadWriteOnce storage" — the StorageClass provisions a PV to satisfy it
> - PV: the actual storage volume

---

## Step 4: Create Headless Service

Update `k8s/base/postgres/service.yaml`:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: shopstream-data
spec:
  clusterIP: None    # This makes it headless
  selector:
    app: postgres
  ports:
    - port: 5432
      targetPort: 5432
```

**K8s concept: Headless Service**
> A headless service (clusterIP: None) doesn't get a virtual IP. Instead, DNS returns the pod IPs directly. This is required for StatefulSets so each pod gets a predictable DNS name: `postgres-0.postgres.shopstream-data.svc.cluster.local`

---

## Step 5: Deploy and Verify Persistence

```bash
# Delete old deployment first
kubectl delete deployment postgres -n shopstream-data

# Apply StatefulSet
kubectl apply -f k8s/base/postgres/

# Wait for it
kubectl get pods -n shopstream-data -w

# Insert data
curl -X POST http://localhost/api/products -H "Content-Type: application/json" -d '{"name":"I Will Survive","price":1.00,"stock":1}'

# Delete the pod (StatefulSet will recreate it)
kubectl delete pod postgres-0 -n shopstream-data

# Wait for new pod, then check
kubectl get pods -n shopstream-data -w
curl http://localhost/api/products
# Data survives!

# Check PVC
kubectl get pvc -n shopstream-data
```

---

## Step 6: Add Redis

Create `k8s/base/redis/deployment.yaml`:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: shopstream-data
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
        - name: redis
          image: redis:7-alpine
          ports:
            - containerPort: 6379
          resources:
            requests:
              cpu: "50m"
              memory: "64Mi"
            limits:
              cpu: "100m"
              memory: "128Mi"
```

Create `k8s/base/redis/service.yaml`:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: shopstream-data
spec:
  selector:
    app: redis
  ports:
    - port: 6379
      targetPort: 6379
  type: ClusterIP
```

---

## Step 7: Update Product Service to Use Redis

Add Redis env vars to the Product Service deployment:
```yaml
env:
  - name: REDIS_HOST
    value: "redis.shopstream-data.svc.cluster.local"
  - name: REDIS_PORT
    value: "6379"
```

Update your Go code to cache product listings in Redis with a TTL.

---

## Exercises

1. **Inspect the PV:** `kubectl get pv` — see the actual volume K8s provisioned
2. **Scale the StatefulSet:** Try `kubectl scale statefulset postgres --replicas=2` — observe the ordered creation (postgres-0, then postgres-1)
3. **Delete PVC:** Delete the StatefulSet, then check if PVC still exists (it does — PVCs outlive pods)
4. **Test Redis caching:** Hit `/api/products` twice, check logs to see if second request hits cache

---

## Checklist

- [ ] PostgreSQL running as StatefulSet with PVC
- [ ] Data survives pod deletion
- [ ] Headless Service for PostgreSQL
- [ ] Redis deployed and accessible
- [ ] Product Service uses Redis for caching
- [ ] Understand: StatefulSet, PV, PVC, StorageClass, Headless Service
