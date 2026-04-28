# Windows Setup Guide

Getting the project running on Windows. Common issues and fixes.

---

## Prerequisites

Install all three via PowerShell:
```powershell
winget install GoLang.Go
winget install Kubernetes.kubectl
winget install Kubernetes.kind
```

Also required: **Docker Desktop** — install from https://www.docker.com/products/docker-desktop/

Restart your terminal after installing, then verify:
```powershell
go version
kubectl version --client
kind --version
docker --version
```

---

## Common Issues

### Go / kind not recognized after install

`winget` installs the binaries but your current terminal session doesn't pick up the new PATH. Two fixes:

**Quick fix (current session only):**
```powershell
$env:Path += ";C:\Program Files\Go\bin"
```

For kind, find it first:
```powershell
where.exe /R "C:\Users" kind.exe
```
Then add that folder to PATH:
```powershell
$env:Path += ";C:\Users\<YourUsername>\AppData\Local\Microsoft\WinGet\Packages\Kubernetes.kind_Microsoft.Winget.Source_8wekyb3d8bbwe"
```

**Permanent fix (run once in admin PowerShell):**
```powershell
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Program Files\Go\bin;C:\Users\<YourUsername>\AppData\Local\Microsoft\WinGet\Packages\Kubernetes.kind_Microsoft.Winget.Source_8wekyb3d8bbwe", "User")
```

Replace `<YourUsername>` with your Windows username.

### `make` not recognized

Windows doesn't ship with `make`. You don't need it — run the commands directly instead of using the Makefile. See the Makefile for what each target does and run those commands manually.

### go.sum checksum mismatch during `docker build`

```
SECURITY ERROR
This download does NOT match an earlier download recorded in go.sum.
```

This happens when `go.sum` was generated on a different Go version or platform than what the Docker image uses. Fix:

```powershell
cd services/<service-name>
del go.sum
go mod tidy
cd ../..
docker build -t <service-name>:v1 services/<service-name>/
```

Deleting `go.sum` and running `go mod tidy` regenerates it with correct checksums. This is safe — `go.sum` is always regeneratable.

### kind cluster nodes not visible to kubectl

If `kubectl get nodes` returns an error after creating a cluster:
```powershell
kubectl cluster-info --context kind-shopstream
```

If that doesn't work, check Docker Desktop is running and the kind containers are up.

### Docker build cache issues

If a build is using stale cached layers:
```powershell
docker build --no-cache -t product-service:v1 services/product-service/
```

### Port conflicts

If port 80 or 443 is already in use when creating the kind cluster, edit `kind-config.yaml` and change `hostPort` to something else (e.g., 8080):
```yaml
extraPortMappings:
  - containerPort: 80
    hostPort: 8080
```
