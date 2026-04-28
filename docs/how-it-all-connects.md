# How It All Connects

## The Layers

```
Code → Docker → Kubernetes → CI/CD automates the flow
```

**1. Code** — You write Go/Python/whatever + database schemas. Runs on your laptop.

**2. Docker** — Wraps your code into an image. "My app + everything it needs to run." Portable — same image runs anywhere. Images live in a registry (like GitHub but for containers).

**3. Kubernetes** — Pulls images from the registry, runs them as pods. Handles scaling, healing, networking, secrets, rollbacks. Doesn't build images — just runs them.

**4. CI/CD (GitHub Actions)** — The glue. Automates: push code → build image → push to registry → deploy to K8s. Without it, you do all of that manually every time.

## Without CI/CD

```
Write code → manually build image → manually push image → manually edit manifest → manually kubectl apply
```

## With CI/CD

```
Write code → git push → ☕ done
```

GitHub Actions builds, tests, pushes the image. ArgoCD sees the change and tells Kubernetes to deploy it.

## The Flow

```
You → git push → GitHub Actions (build + test + push image)
                        ↓
               Container Registry (stores images)
                        ↓
               ArgoCD (watches Git for changes)
                        ↓
               Kubernetes (runs pods with new image)
                        ↓
               Users hit your app
```

## Key Distinction

- **Kubernetes** = how you RUN and MANAGE containers
- **CI/CD** = how you GET code changes INTO those containers automatically
- **Docker** = how you PACKAGE your code into something K8s can run

They're different layers, not alternatives. They work together.
