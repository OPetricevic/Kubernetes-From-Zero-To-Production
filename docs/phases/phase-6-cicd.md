# Phase 6: CI/CD & GitOps

## Goal
Automate the build-and-deploy pipeline. Push code → build image → deploy to cluster. Learn Kustomize and ArgoCD.

## Prerequisites
- [ ] Phase 5 complete
- [ ] GitHub repository for the project

---

## Step 1: Set Up Kustomize

Kustomize lets you maintain a base set of manifests and overlay environment-specific changes.

Create `k8s/base/kustomization.yaml`:
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - namespaces.yaml
  - product-service/deployment.yaml
  - product-service/service.yaml
  - product-service/hpa.yaml
  - order-service/deployment.yaml
  - order-service/service.yaml
  - user-service/deployment.yaml
  - user-service/service.yaml
  - notification-service/deployment.yaml
  - notification-service/service.yaml
  - frontend/deployment.yaml
  - frontend/service.yaml
  - ingress/ingress.yaml
commonLabels:
  project: shopstream
```

Create `k8s/overlays/dev/kustomization.yaml`:
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base
namePrefix: dev-
namespace: shopstream-dev
patches:
  - target:
      kind: Deployment
    patch: |
      - op: replace
        path: /spec/replicas
        value: 1
```

Create `k8s/overlays/prod/kustomization.yaml`:
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base
namePrefix: prod-
namespace: shopstream-prod
patches:
  - target:
      kind: Deployment
      name: product-service
    patch: |
      - op: replace
        path: /spec/replicas
        value: 3
  - target:
      kind: Deployment
    patch: |
      - op: replace
        path: /spec/template/spec/containers/0/resources/requests/cpu
        value: "200m"
      - op: replace
        path: /spec/template/spec/containers/0/resources/requests/memory
        value: "256Mi"
```

```bash
# Preview what dev overlay produces
kubectl kustomize k8s/overlays/dev/

# Preview prod
kubectl kustomize k8s/overlays/prod/

# Apply
kubectl apply -k k8s/overlays/dev/
```

**K8s concept: Kustomize**
> Kustomize is built into kubectl. It lets you customize manifests without templating (unlike Helm). Base + overlays = DRY config management across environments.

---

## Step 2: Install ArgoCD

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Wait for it
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=argocd-server -n argocd --timeout=120s

# Get initial admin password
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d

# Port forward to access UI
kubectl port-forward svc/argocd-server -n argocd 8080:443
# Open https://localhost:8080 — login with admin / <password>
```

**K8s concept: ArgoCD (GitOps)**
> GitOps principle: Git is the single source of truth. ArgoCD watches your Git repo and ensures the cluster state matches what's in Git. If someone manually changes something in the cluster, ArgoCD detects the drift and can auto-sync.

---

## Step 3: Create ArgoCD Application

Create `k8s/argocd/application.yaml`:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: shopstream-dev
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/<your-username>/shopstream.git
    targetRevision: main
    path: k8s/overlays/dev
  destination:
    server: https://kubernetes.default.svc
    namespace: shopstream-dev
  syncPolicy:
    automated:
      prune: true       # delete resources removed from Git
      selfHeal: true    # revert manual cluster changes
    syncOptions:
      - CreateNamespace=true
```

```bash
kubectl apply -f k8s/argocd/application.yaml
```

---

## Step 4: Set Up GitHub Actions CI

Create `.github/workflows/ci.yaml`:
```yaml
name: CI/CD

on:
  push:
    branches: [main]
    paths:
      - 'services/**'

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: [product-service, order-service, user-service, notification-service]
    steps:
      - uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: services/${{ matrix.service }}
          push: true
          tags: |
            ghcr.io/${{ github.repository }}/${{ matrix.service }}:${{ github.sha }}
            ghcr.io/${{ github.repository }}/${{ matrix.service }}:latest

  update-manifests:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Update image tags in Kustomize
        run: |
          for service in product-service order-service user-service notification-service; do
            cd k8s/base/${service}
            kustomize edit set image ${service}=ghcr.io/${{ github.repository }}/${service}:${{ github.sha }}
            cd ../../..
          done

      - name: Commit and push
        run: |
          git config user.name "GitHub Actions"
          git config user.email "actions@github.com"
          git add k8s/
          git commit -m "Update image tags to ${{ github.sha }}"
          git push
```

**K8s concept: Image Pull Policies**
> - `Always`: pull every time (use with `:latest` tag)
> - `IfNotPresent`: only pull if not cached (use with specific tags like `:v1.2.3` or `:abc123`)
> - `Never`: only use local images (for local dev with kind)
> - In production, always use specific image tags (commit SHA), never `:latest`

---

## Step 5: Canary Deployments with Argo Rollouts (Optional)

```bash
kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml
```

Convert a Deployment to a Rollout:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: product-service
  namespace: shopstream
spec:
  replicas: 3
  selector:
    matchLabels:
      app: product-service
  template:
    # same as Deployment template
  strategy:
    canary:
      steps:
        - setWeight: 20      # send 20% traffic to new version
        - pause: {duration: 60s}
        - setWeight: 50
        - pause: {duration: 60s}
        - setWeight: 80
        - pause: {duration: 30s}
```

---

## Exercises

1. **Push a change and watch ArgoCD sync:** Modify a manifest, push to Git, watch ArgoCD UI
2. **Manually drift:** `kubectl scale deployment product-service --replicas=10` — watch ArgoCD detect and revert
3. **Compare overlays:** `kubectl kustomize k8s/overlays/dev/` vs `kubectl kustomize k8s/overlays/prod/`
4. **Canary rollout:** Deploy a new version and watch traffic gradually shift

---

## Checklist

- [ ] Kustomize base + overlays for dev and prod
- [ ] ArgoCD installed and syncing from Git
- [ ] GitHub Actions building and pushing images
- [ ] Image tags updated automatically in manifests
- [ ] Understand: Kustomize, GitOps, ArgoCD, CI/CD pipeline, Image Pull Policies
