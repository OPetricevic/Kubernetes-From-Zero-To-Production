# Phase 10: Live Cloud Deployment

## Goal
Deploy the entire system to Oracle Cloud Free Tier. A real Kubernetes cluster running on real infrastructure with a public endpoint.

## Prerequisites
- [ ] Phase 9 complete
- [ ] Oracle Cloud account (free tier)
- [ ] `oci` CLI installed
- [ ] A domain name (optional — can use the public IP directly)

---

## Step 1: Set Up Oracle Cloud Account

1. Sign up at https://cloud.oracle.com (free tier — no credit card charge)
2. Choose your home region (closest to you)
3. Wait for account provisioning (can take up to 30 minutes)

**What you get for free (forever):**
- 4 ARM-based CPUs (Ampere A1)
- 24 GB RAM
- 200 GB block storage
- 10 TB/month outbound data
- Oracle Kubernetes Engine (OKE) — managed K8s control plane is free

> This is more than enough for our entire system.

---

## Step 2: Install OCI CLI

```bash
# Windows (PowerShell)
Set-ExecutionPolicy RemoteSigned
Invoke-WebRequest https://raw.githubusercontent.com/oracle/oci-cli/master/scripts/install/install.ps1 -OutFile install.ps1
./install.ps1

# Configure
oci setup config
# Follow prompts — needs your tenancy OCID, user OCID, region, and API key
```

Verify:
```bash
oci iam region list --output table
```

---

## Step 3: Create OKE Cluster

### Option A: Via Console (easier for first time)
1. Go to Oracle Cloud Console → Developer Services → Kubernetes Clusters (OKE)
2. Create Cluster → Quick Create
3. Settings:
   - Name: `shopstream-prod`
   - Kubernetes version: latest stable
   - Node shape: `VM.Standard.A1.Flex` (ARM, free tier)
   - OCPUs per node: 2
   - Memory per node: 12 GB
   - Number of nodes: 2
4. Create and wait (~10 minutes)

### Option B: Via CLI
```bash
# Create VCN and cluster (simplified)
oci ce cluster create \
  --compartment-id <your-compartment-id> \
  --name shopstream-prod \
  --kubernetes-version v1.28.0 \
  --vcn-id <your-vcn-id> \
  --service-lb-subnet-ids '["<subnet-id>"]'
```

### Connect kubectl to OKE
```bash
oci ce cluster create-kubeconfig \
  --cluster-id <cluster-ocid> \
  --file $HOME/.kube/config \
  --region <your-region> \
  --token-version 2.0.0

# Verify
kubectl get nodes
```

---

## Step 4: Push Images to Container Registry

Oracle Cloud has a free container registry (OCIR).

```bash
# Login to OCIR
docker login <region>.ocir.io -u <tenancy-namespace>/<username>

# Tag and push images
docker tag product-service:v2 <region>.ocir.io/<tenancy-namespace>/shopstream/product-service:v2
docker push <region>.ocir.io/<tenancy-namespace>/shopstream/product-service:v2

# Repeat for all services
```

Create an image pull secret in the cluster:
```bash
kubectl create secret docker-registry ocir-secret \
  --docker-server=<region>.ocir.io \
  --docker-username='<tenancy-namespace>/<username>' \
  --docker-password='<auth-token>' \
  -n shopstream
```

---

## Step 5: Update Manifests for Cloud

Key changes from local (kind) to cloud (OKE):

1. **Image references** — change from `product-service:v2` to `<region>.ocir.io/<namespace>/shopstream/product-service:v2`
2. **imagePullPolicy** — change from `Never` to `IfNotPresent`
3. **imagePullSecrets** — add OCIR pull secret to deployments
4. **Storage** — PVCs will use Oracle Block Storage instead of local-path
5. **Ingress** — LoadBalancer type gets a real public IP from Oracle

Create a cloud overlay:
```
k8s/overlays/cloud/kustomization.yaml
```

This overlay patches:
- Image references to OCIR
- imagePullPolicy to IfNotPresent
- Adds imagePullSecrets
- Adjusts resource requests for ARM architecture

---

## Step 6: Install Ingress Controller on OKE

```bash
helm install ingress-nginx ingress-nginx/ingress-nginx \
  -n ingress-nginx --create-namespace \
  --set controller.service.type=LoadBalancer
```

Wait for the external IP:
```bash
kubectl get svc -n ingress-nginx -w
```

Oracle will provision a real Load Balancer with a public IP. This is your entry point.

---

## Step 7: TLS with cert-manager (Optional — requires domain)

```bash
helm install cert-manager jetstack/cert-manager \
  -n cert-manager --create-namespace \
  --set installCRDs=true
```

Create ClusterIssuer:
```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: omar.petricevic@gmail.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
      - http01:
          ingress:
            class: nginx
```

Update Ingress with TLS:
```yaml
metadata:
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
    - hosts:
        - shopstream.yourdomain.com
      secretName: shopstream-tls
```

> Without a domain, you can still access the app via the Load Balancer's public IP — just no HTTPS.

---

## Step 8: Deploy Everything

```bash
# Apply the cloud overlay
kubectl apply -k k8s/overlays/cloud/

# Or connect ArgoCD to the cloud cluster and let it sync
```

Verify:
```bash
kubectl get pods -n shopstream
kubectl get pods -n shopstream-data
kubectl get svc -n ingress-nginx  # check external IP
curl http://<external-ip>/api/products
```

---

## Step 9: Connect ArgoCD (Optional)

Install ArgoCD on the cloud cluster:
```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

Create an Application pointing at the cloud overlay:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: shopstream-cloud
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/OPetricevic/Kubernetes-From-Zero-To-Production.git
    targetRevision: main
    path: k8s/overlays/cloud
  destination:
    server: https://kubernetes.default.svc
    namespace: shopstream
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - ServerSideApply=true
```

---

## Exercises

1. **Hit the public endpoint** — curl the Load Balancer IP from your phone or another network
2. **Scale and watch** — increase replicas, watch Oracle provision resources
3. **Break something** — delete a pod, watch it recover on cloud infrastructure
4. **Check costs** — verify everything stays within free tier limits

---

## Checklist

- [ ] Oracle Cloud account created
- [ ] OKE cluster running with 2 nodes
- [ ] Images pushed to OCIR
- [ ] Manifests updated for cloud (overlay)
- [ ] Ingress controller with public IP
- [ ] All services running and accessible
- [ ] (Optional) TLS with cert-manager
- [ ] (Optional) ArgoCD syncing from Git
- [ ] Verify free tier — no charges

---

## What You've Achieved

Your app is live on the internet. A real Kubernetes cluster, running real services, accessible by anyone with the URL. The same system you built locally across 9 phases is now running on cloud infrastructure.

From `kubectl apply` on your laptop to a live cloud deployment — that's the full Kubernetes journey.
