.PHONY: cluster-create cluster-delete build load deploy deploy-all port-forward clean

# ============================================================
# Phase 1 Commands
# ============================================================

# Create the kind cluster with 1 control-plane + 2 workers
cluster-create:
	kind create cluster --name shopstream --config kind-config.yaml

# Delete the cluster
cluster-delete:
	kind delete cluster --name shopstream

# Build the product-service Docker image
build:
	docker build -t product-service:v1 services/product-service/

# Load the image into the kind cluster (kind can't pull from local Docker)
load:
	kind load docker-image product-service:v1 --name shopstream

# Deploy namespaces, then postgres, then product-service
deploy:
	kubectl apply -f k8s/base/namespaces.yaml
	kubectl apply -f k8s/base/postgres/
	@echo "Waiting for postgres to be ready..."
	kubectl wait --for=condition=ready pod -l app=postgres -n shopstream-data --timeout=120s
	kubectl apply -f k8s/base/product-service/

# Build + load + deploy in one command
deploy-all: build load deploy

# Port-forward product-service to localhost:8081
port-forward:
	kubectl port-forward svc/product-service 8081:80 -n shopstream

# Tear down all resources (but keep the cluster)
clean:
	kubectl delete -f k8s/base/product-service/ --ignore-not-found
	kubectl delete -f k8s/base/postgres/ --ignore-not-found
	kubectl delete -f k8s/base/namespaces.yaml --ignore-not-found
