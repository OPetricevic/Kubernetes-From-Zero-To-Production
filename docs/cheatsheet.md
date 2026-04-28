# kubectl & Helm Cheatsheet

Commands you'll use constantly throughout this project.

---

## Cluster Setup

```bash
# Create a local cluster with kind
kind create cluster --name shopstream --config kind-config.yaml

# Create a local cluster with minikube
minikube start --profile shopstream --cpus 4 --memory 8192

# Check cluster info
kubectl cluster-info
kubectl get nodes
```

## Namespace Operations

```bash
# Create namespace
kubectl create namespace shopstream
kubectl create namespace shopstream-data

# Set default namespace for context
kubectl config set-context --current --namespace=shopstream

# List all namespaces
kubectl get namespaces
```

## Deployments & Pods

```bash
# Apply manifests
kubectl apply -f k8s/base/product-service/
kubectl apply -f k8s/base/product-service/deployment.yaml

# Check status
kubectl get deployments
kubectl get pods
kubectl get pods -o wide                    # show node + IP
kubectl get pods -w                         # watch in real-time

# Describe (detailed info + events)
kubectl describe pod <pod-name>
kubectl describe deployment product-service

# Logs
kubectl logs <pod-name>
kubectl logs <pod-name> -f                  # follow/stream
kubectl logs <pod-name> -c <container>      # specific container
kubectl logs <pod-name> --previous          # crashed container

# Exec into a pod
kubectl exec -it <pod-name> -- /bin/sh

# Port forward (quick local access)
kubectl port-forward svc/product-service 8081:80

# Delete
kubectl delete -f k8s/base/product-service/
kubectl delete pod <pod-name>               # will be recreated by Deployment
```

## Services & Networking

```bash
# List services
kubectl get svc
kubectl get svc -A                          # all namespaces
kubectl get endpoints product-service       # see backing pod IPs

# Test DNS from inside cluster
kubectl run tmp --image=busybox --rm -it -- nslookup product-service.shopstream.svc.cluster.local

# Test connectivity from inside cluster
kubectl run tmp --image=curlimages/curl --rm -it -- curl http://product-service.shopstream.svc.cluster.local/api/products
```

## ConfigMaps & Secrets

```bash
# Create from literal
kubectl create configmap app-config --from-literal=DB_HOST=postgres --from-literal=DB_PORT=5432
kubectl create secret generic db-secret --from-literal=DB_PASSWORD=mysecret

# Create from file
kubectl create configmap app-config --from-file=config.yaml
kubectl create secret generic tls-secret --from-file=tls.crt --from-file=tls.key

# View
kubectl get configmaps
kubectl describe configmap app-config
kubectl get secret db-secret -o jsonpath='{.data.DB_PASSWORD}' | base64 -d
```

## Storage

```bash
# List PVs and PVCs
kubectl get pv
kubectl get pvc
kubectl get storageclass
```

## Scaling

```bash
# Manual scale
kubectl scale deployment product-service --replicas=5

# Autoscaler
kubectl get hpa
kubectl describe hpa product-service

# Watch scaling in action
kubectl get pods -w
```

## Rollouts

```bash
# Check rollout status
kubectl rollout status deployment/product-service

# View history
kubectl rollout history deployment/product-service

# Rollback
kubectl rollout undo deployment/product-service
kubectl rollout undo deployment/product-service --to-revision=2

# Restart (rolling restart)
kubectl rollout restart deployment/product-service
```

## Debugging

```bash
# Events (cluster-wide)
kubectl get events --sort-by='.lastTimestamp'
kubectl get events -n shopstream

# Resource usage
kubectl top pods
kubectl top nodes

# Check why a pod isn't scheduling
kubectl describe pod <pod-name>   # look at Events section

# Check RBAC
kubectl auth can-i create pods --as=system:serviceaccount:shopstream:default
```

## Helm

```bash
# Add repos
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Search
helm search repo rabbitmq

# Install
helm install rabbitmq bitnami/rabbitmq -n shopstream-data -f helm/rabbitmq-values.yaml

# Upgrade
helm upgrade rabbitmq bitnami/rabbitmq -n shopstream-data -f helm/rabbitmq-values.yaml

# List releases
helm list -A

# Uninstall
helm uninstall rabbitmq -n shopstream-data
```

## Kustomize

```bash
# Preview what will be applied
kubectl kustomize k8s/overlays/dev/

# Apply overlay
kubectl apply -k k8s/overlays/dev/
kubectl apply -k k8s/overlays/prod/

# Diff before applying
kubectl diff -k k8s/overlays/dev/
```

## Quick Diagnostics

```bash
# "Is everything running?"
kubectl get all -n shopstream
kubectl get all -n shopstream-data

# "What went wrong?"
kubectl get events -n shopstream --sort-by='.lastTimestamp' | tail -20
kubectl logs -l app=product-service --tail=50
```
