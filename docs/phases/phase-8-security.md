# Phase 8: Security Hardening

## Goal
Lock down the cluster with least-privilege access, encrypted secrets, and restricted pod capabilities.

## Prerequisites
- [ ] Phase 7 complete

---

## Step 1: RBAC — Role-Based Access Control

Create a ServiceAccount for each service:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: product-service
  namespace: shopstream
```

Create a Role (namespace-scoped permissions):
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: product-service-role
  namespace: shopstream
rules:
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["secrets"]
    resourceNames: ["product-secret"]
    verbs: ["get"]
```

Bind the Role to the ServiceAccount:
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: product-service-binding
  namespace: shopstream
subjects:
  - kind: ServiceAccount
    name: product-service
    namespace: shopstream
roleRef:
  kind: Role
  name: product-service-role
  apiGroup: rbac.authorization.k8s.io
```

Assign ServiceAccount to the Deployment:
```yaml
spec:
  template:
    spec:
      serviceAccountName: product-service
      automountServiceAccountToken: false   # don't mount token unless needed
```

**K8s concepts: RBAC, ServiceAccount, Role, RoleBinding**
> - ServiceAccount: identity for pods (like a user account but for workloads)
> - Role: what actions are allowed (get secrets, list pods, etc.)
> - RoleBinding: connects a ServiceAccount to a Role
> - ClusterRole/ClusterRoleBinding: same but cluster-wide (use sparingly)
> - `automountServiceAccountToken: false` — don't give pods API access unless they need it

---

## Step 2: Pod Security Standards

Apply security contexts to all pods:

```yaml
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 1000
        fsGroup: 1000
      containers:
        - name: product-service
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop:
                - ALL
          volumeMounts:
            - name: tmp
              mountPath: /tmp
      volumes:
        - name: tmp
          emptyDir: {}
```

**K8s concepts: Security Context, Pod Security Standards**
> - `runAsNonRoot`: container can't run as root
> - `readOnlyRootFilesystem`: container can't write to its filesystem (use emptyDir for /tmp)
> - `capabilities.drop.ALL`: remove all Linux capabilities
> - `allowPrivilegeEscalation: false`: prevent gaining more privileges than parent process

---

## Step 3: Pod Security Admission

Label namespaces to enforce security standards:

```bash
kubectl label namespace shopstream \
  pod-security.kubernetes.io/enforce=restricted \
  pod-security.kubernetes.io/warn=restricted \
  pod-security.kubernetes.io/audit=restricted
```

**K8s concept: Pod Security Admission**
> Replaces the old PodSecurityPolicy. Three modes:
> - `enforce`: reject pods that violate
> - `warn`: allow but show warning
> - `audit`: allow but log violation
> Three levels: `privileged` (anything goes), `baseline` (reasonable defaults), `restricted` (hardened)

---

## Step 4: Sealed Secrets

Install Sealed Secrets controller:
```bash
helm repo add sealed-secrets https://bitnami-labs.github.io/sealed-secrets
helm install sealed-secrets sealed-secrets/sealed-secrets -n kube-system
```

Install kubeseal CLI:
```bash
# Download from https://github.com/bitnami-labs/sealed-secrets/releases
```

Encrypt a secret:
```bash
# Create a regular secret
kubectl create secret generic product-secret \
  --from-literal=DB_USER=shopstream \
  --from-literal=DB_PASSWORD=shopstream123 \
  --dry-run=client -o yaml > secret.yaml

# Seal it
kubeseal --format yaml < secret.yaml > k8s/base/product-service/sealed-secret.yaml

# The sealed secret is safe to commit to Git
cat k8s/base/product-service/sealed-secret.yaml
```

**K8s concept: Sealed Secrets**
> Regular K8s Secrets are just base64 encoded — not encrypted. Sealed Secrets encrypts them with a cluster-specific key. Only the controller in the cluster can decrypt them. Safe to store in Git.

---

## Step 5: Network Policies (Tighten)

If you haven't already from Phase 5, ensure every namespace has:
1. Default deny all ingress and egress
2. Explicit allow rules for each service

Verify:
```bash
# From product-service pod, try to reach something it shouldn't
kubectl exec -it <product-pod> -n shopstream -- wget -qO- http://user-service.shopstream.svc.cluster.local/api/users/me
# Should be blocked by network policy
```

---

## Step 6: Image Security

Scan images for vulnerabilities:
```bash
# Install Trivy
# Scan your images
trivy image product-service:v1
trivy image order-service:v1
```

Add to CI pipeline:
```yaml
- name: Scan image
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: ghcr.io/${{ github.repository }}/product-service:${{ github.sha }}
    exit-code: 1
    severity: CRITICAL,HIGH
```

---

## Exercises

1. **Test RBAC:** Try to access the K8s API from inside a pod with `automountServiceAccountToken: false`
2. **Break Pod Security:** Try to deploy a pod running as root in the restricted namespace
3. **Seal and unseal:** Create a sealed secret, apply it, verify the controller decrypts it
4. **Audit:** Check `kubectl get events` for Pod Security Admission violations

---

## Checklist

- [ ] ServiceAccounts created for each service
- [ ] RBAC Roles and RoleBindings configured
- [ ] Security contexts on all pods (non-root, read-only FS, no capabilities)
- [ ] Pod Security Admission enforcing restricted profile
- [ ] Sealed Secrets replacing plain Secrets in Git
- [ ] Image scanning in CI pipeline
- [ ] Understand: RBAC, ServiceAccounts, Security Contexts, Pod Security Admission, Sealed Secrets
