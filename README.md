# Three-Tier Web App Deployment with Kubernetes

![CI/CD Pipeline](https://img.shields.io/badge/CI%2FCD-GitHub%20Actions-2088FF?logo=githubactions&logoColor=white)
![Kubernetes](https://img.shields.io/badge/Orchestration-Kubernetes-326CE5?logo=kubernetes&logoColor=white)
![Go](https://img.shields.io/badge/API-GoLang-00ADD8?logo=go&logoColor=white)
![React](https://img.shields.io/badge/Frontend-React-61DAFB?logo=react&logoColor=black)
![MongoDB](https://img.shields.io/badge/Database-MongoDB-47A248?logo=mongodb&logoColor=white)
![Docker](https://img.shields.io/badge/Containers-Docker-2496ED?logo=docker&logoColor=white)

A three-tier microservice architecture deployed on Kubernetes, with a GitHub Actions pipeline that builds, validates, and deploys on every push to `main`. Each tier runs as an independent Kubernetes workload with resource limits, health probes, and inter-service communication via internal DNS.

---

## Architecture

```
                    ┌─────────────────────────────────────────────────────┐
                    │              Kubernetes Cluster (minikube)          │
                    │                                                     │
Browser ──────────► │  frontend-service (NodePort :30080)                 │
                    │         │                                           │
                    │         ▼                                           │
                    │  ┌─────────────────┐                                │
                    │  │  React Frontend  │  Deployment · 2 replicas      │
                    │  │  nginx :80       │  serves static build          │
                    │  └────────┬────────┘  proxies /api/* to backend     │
                    │           │                                         │
                    │           ▼  backend-service (ClusterIP :8080)      │
                    │  ┌─────────────────┐                                │
                    │  │   Go REST API   │  Deployment · 2 replicas       │
                    │  │   :8080         │  /api/tasks GET + POST         │
                    │  └────────┬────────┘  /health liveness probe        │
                    │           │                                         │
                    │           ▼  mongo-service (ClusterIP :27017)       │
                    │  ┌─────────────────┐                                │
                    │  │    MongoDB 7.0  │  StatefulSet · 1 replica       │
                    │  │    :27017       │  PersistentVolumeClaim 1Gi     │
                    │  └─────────────────┘                                │
                    └─────────────────────────────────────────────────────┘
```

**Why three separate tiers?** Each tier scales, updates, and fails independently. You can roll out a new API version without touching the frontend or database. You can scale the Go API to 10 replicas under load without scaling MongoDB. This separation of concerns is the core value of microservice architecture.

---

## Tech Stack

| Layer | Technology | Kubernetes Object |
|---|---|---|
| Frontend | React + Vite, served by nginx | Deployment + NodePort Service |
| API | Go 1.22, stdlib net/http | Deployment + ClusterIP Service |
| Database | MongoDB 7.0 | StatefulSet + ClusterIP Service + PVC |
| Containers | Docker (multi-stage builds) | — |
| Orchestration | Kubernetes (minikube locally) | — |
| CI/CD | GitHub Actions | — |
| Registry | GitHub Container Registry (GHCR) | — |

---

## Project Structure

```
three-tier-k8s/
├── .github/
│   └── workflows/
│       └── ci.yml                  # GitHub Actions pipeline
├── backend/
│   ├── main.go                     # Go API — tasks CRUD + /health
│   ├── go.mod
│   └── Dockerfile                  # multi-stage: builder + alpine
├── frontend/
│   ├── src/
│   │   └── App.jsx                 # React task UI
│   ├── nginx.conf                  # proxy /api/* + React Router fallback
│   └── Dockerfile                  # multi-stage: node builder + nginx
├── k8s/
│   ├── backend-deployment.yaml
│   ├── backend-service.yaml
│   ├── frontend-deployment.yaml
│   ├── frontend-service.yaml
│   ├── mongo-statefulset.yaml
│   ├── mongo-service.yaml
│   └── mongo-pvc.yaml
└── docker-compose.yml              # local dev only
```

---

## CI/CD Pipeline

```
git push to main
        │
        ▼
┌─────────────────────────────────┐
│        build-and-test           │
│  • docker build backend         │
│  • docker build frontend        │
│  • push both to GHCR            │
│  • smoke test /health endpoint  │
└──────────────┬──────────────────┘
               │ needs: build-and-test
               ▼
┌─────────────────────────────────┐
│       validate-manifests        │
│  • kubectl apply --dry-run      │
│  • validates all 7 manifests    │
│  • no cluster needed            │
└──────────────┬──────────────────┘
               │ needs: validate-manifests
               │ only on push to main
               ▼
┌─────────────────────────────────┐
│            deploy               │
│  • injects commit SHA image tag │
│  • kubectl apply -f k8s/        │
└─────────────────────────────────┘
```

**Key pipeline decisions:**

`--dry-run=client` validation runs on every PR — catches manifest syntax errors and wrong API versions before they reach any cluster.

Images tagged with `github.sha` (commit hash), never `:latest`. Every deployed image is traceable to the exact commit that produced it.

Multi-stage Docker builds: Go binary compiled in `golang:1.22-alpine`, final image is `alpine:3.19` (~15MB). React built in `node:20-alpine`, served by `nginx:alpine`. No build tooling in production images.

---

## Running Locally

### Prerequisites

- Docker Desktop
- minikube
- kubectl

### Option 1 — Docker Compose (fastest for development)

```bash
git clone https://github.com/vishesh3011/k8s-wrangler.git
cd three-tier-k8s

docker compose up --build
# Frontend: http://localhost:3000
# API:      http://localhost:8080
```

### Option 2 — Kubernetes on minikube

```bash
# Start minikube
minikube start

# Deploy all manifests
kubectl apply -f k8s/

# Watch pods come up
kubectl get pods -w

# Get the frontend URL (use this, not the ClusterIP)
minikube service frontend-service --url
```

---

## Verifying Inter-tier Communication

After deploying, verify each connection point explicitly rather than assuming the UI working means everything is fine:

```bash
# 1. All pods running and ready?
kubectl get pods
kubectl get deployments

# 2. All services exist with correct types?
kubectl get services
# frontend-service → NodePort
# backend-service  → ClusterIP
# mongo-service    → ClusterIP

# 3. Services have found their pods? (selector match check)
kubectl get endpoints
# If ENDPOINTS shows <none> → Service selector doesn't match pod labels

# 4. Backend can reach MongoDB?
kubectl exec -it $(kubectl get pod -l app=backend -o jsonpath='{.items[0].metadata.name}') -- \
  wget -qO- http://localhost:8080/api/tasks
# Should return [] or existing tasks — proves MongoDB connection works

# 5. Frontend nginx can reach backend service by DNS name?
kubectl exec -it $(kubectl get pod -l app=frontend -o jsonpath='{.items[0].metadata.name}') -- \
  wget -qO- http://backend-service:8080/health
# Should return {"status":"ok"}
```

---

## Kubernetes Concepts Demonstrated

**Deployment vs StatefulSet** — React and Go API use Deployments (stateless, pods are interchangeable). MongoDB uses a StatefulSet because each replica needs a stable identity (`mongo-0`) and its own persistent storage. Restarting `mongo-0` reattaches to the same PVC — data survives.

**Service types** — ClusterIP for internal services (backend, MongoDB) — not reachable from outside the cluster by design. NodePort for the frontend — opens a port on the minikube node IP, reachable from the host machine.

**Service discovery** — pods reach each other by Service name (`backend-service`, `mongo-service`), not by IP. Kubernetes internal DNS resolves these names to the correct ClusterIP. Pod IPs are ephemeral; Service names are stable.

**Resource requests vs limits** — `requests` is the guaranteed minimum used for scheduling. `limits` is the hard ceiling — exceed memory limit and the container is OOMKilled (exit code 137). Both are set on all containers.

**Liveness vs readiness probes** — liveness: "is this container alive?" (restart if not). Readiness: "is this container ready for traffic?" (remove from Service load balancer if not). A container can be alive but not ready — e.g. still establishing the MongoDB connection on startup.

**PersistentVolumeClaim** — MongoDB's data directory (`/data/db`) is mounted from a PVC. The PVC survives pod restarts and rescheduling. `kubectl delete pod mongo-0` recreates the pod, which reattaches to the same PVC — no data loss.

**`kubectl apply --dry-run=client`** — validates manifest syntax and API schema without touching a cluster. Used in CI to catch errors on every PR.

---

## Debugging Reference

Real errors encountered and resolved during this project:

### CrashLoopBackOff

```bash
kubectl logs <pod-name> --previous   # logs from the crashed container
kubectl describe pod <pod-name>      # check Events section
```

Common causes: wrong environment variable (MONGO_URI typo), image pull failure, app panicking on startup before DB is ready.

### OOMKilled (exit code 137)

```bash
kubectl describe pod <pod-name>
# Look for: Last State: Terminated, Reason: OOMKilled
```

Fix: increase `resources.limits.memory`. MongoDB needs at least `256Mi` under normal load. Setting limits too low triggers OOMKilled in a loop, which presents as CrashLoopBackOff.

### Service not routing traffic

```bash
kubectl get endpoints <service-name>
# If <none>: the Service selector doesn't match any pod labels
# Fix: ensure spec.selector in Service matches metadata.labels in pod template
```

### Pod stuck in Pending

```bash
kubectl describe pod <pod-name>
# Check Events for: Insufficient memory, Insufficient cpu, or no nodes available
# Fix: reduce resource requests or increase minikube resources
# minikube start --memory=4096 --cpus=4
```

---

## Key Learnings

- Why Deployments are used for stateless tiers and StatefulSets for databases
- How Kubernetes service discovery works via internal DNS (not IPs)
- The difference between ClusterIP, NodePort, and LoadBalancer Service types
- Why `kubectl get endpoints` shows pod IPs you can't reach from your Mac — and how to actually access services on minikube
- How liveness and readiness probes prevent traffic from hitting unhealthy pods
- Why resource limits exist and how OOMKilled differs from a regular crash
- How `--dry-run=client` catches manifest errors without a live cluster
- Why multi-stage Docker builds produce dramatically smaller production images