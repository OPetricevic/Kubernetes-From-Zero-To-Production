# Troubleshooting Guide

Common issues you'll hit during this project and how to fix them.

---

## Pod Issues

### Pod stuck in `Pending`
**Cause:** Not enough resources, or no node matches scheduling constraints.
```bash
kubectl describe pod <pod-name>
# Look at the Events section at the bottom
```
**Fixes:**
- Check resource requests vs available node resources: `kubectl describe node`
- Check taints/tolerations if using node affinity
- Check PVC binding if pod uses persistent storage

### Pod stuck in `CrashLoopBackOff`
**Cause:** Container starts and immediately crashes.
```bash
kubectl logs <pod-name> --previous    # logs from the crashed instance
kubectl describe pod <pod-name>       # check exit code
```
**Fixes:**
- Exit code 1: Application error — check your Go code / env vars
- Exit code 137: OOMKilled — increase memory limits
- Exit code 0 but restarting: Container exits successfully but K8s expects it to keep running

### Pod stuck in `ImagePullBackOff`
**Cause:** Can't pull the container image.
```bash
kubectl describe pod <pod-name>   # look for image pull error
```
**Fixes:**
- Check image name and tag are correct
- If using local images with kind: `kind load docker-image <image> --name shopstream`
- If using minikube: `eval $(minikube docker-env)` then build

### Pod stuck in `Init:0/1`
**Cause:** Init container hasn't completed.
```bash
kubectl logs <pod-name> -c <init-container-name>
```
**Fixes:**
- Init container waiting for a dependency (DB not ready yet)
- Check if the service the init container depends on is running

---

## Service / Networking Issues

### Can't reach service from another pod
```bash
# Test DNS resolution
kubectl exec -it <pod-name> -- nslookup <service-name>

# Test connectivity
kubectl exec -it <pod-name> -- wget -qO- http://<service-name>:<port>/health
```
**Fixes:**
- Check service selector matches pod labels: `kubectl get endpoints <service-name>`
- If endpoints list is empty, labels don't match
- Check the pod is Ready (readiness probe passing)
- Check Network Policies aren't blocking traffic

### Ingress not routing traffic
```bash
kubectl get ingress
kubectl describe ingress <ingress-name>
```
**Fixes:**
- Check Ingress controller is running: `kubectl get pods -n ingress-nginx`
- Check Ingress class annotation matches your controller
- For kind: make sure you created the cluster with port mappings
- For minikube: run `minikube tunnel` in a separate terminal

---

## Database Issues

### PostgreSQL pod won't start with PVC
```bash
kubectl get pvc -n shopstream-data
kubectl describe pvc <pvc-name>
```
**Fixes:**
- PVC stuck in Pending: no StorageClass or PV available
- For kind/minikube: use the default StorageClass
- Check `kubectl get storageclass`

### Can't connect to PostgreSQL from app
**Fixes:**
- Check Secret values are correct (base64 encoded properly)
- Check ConfigMap has correct host: `postgres.shopstream-data.svc.cluster.local`
- Port-forward and test locally: `kubectl port-forward svc/postgres 5432:5432 -n shopstream-data`

---

## Resource Issues

### OOMKilled
```bash
kubectl describe pod <pod-name>   # look for OOMKilled in Last State
```
**Fix:** Increase memory limit in deployment manifest.

### CPU Throttling
```bash
kubectl top pod <pod-name>
```
**Fix:** Increase CPU limit, or check if your code has a hot loop.

---

## Helm Issues

### Helm install fails
```bash
helm install <name> <chart> --debug --dry-run   # preview without installing
```
**Fixes:**
- Check values file syntax (YAML indentation)
- Check namespace exists
- Check if a release with the same name already exists: `helm list -A`

---

## General Debugging Workflow

When something isn't working, follow this order:

1. `kubectl get pods` — is the pod running?
2. `kubectl describe pod <name>` — why isn't it running?
3. `kubectl logs <name>` — what's the app saying?
4. `kubectl get events` — what happened recently?
5. `kubectl get endpoints <svc>` — is the service wired to pods?
6. `kubectl exec -it <pod> -- sh` — get inside and poke around
