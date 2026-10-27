# Twelve-Factor HTTP Server — Homework 1

A small HTTP server written in Go and deployed to Kubernetes. The application uses Redis to maintain a shared visit counter across multiple replicas and follows the Twelve-Factor App methodology.

## Features

- `GET /` increments a shared Redis counter and returns the configured greeting, visit number, and serving pod hostname.
- `GET /healthz` returns `200 OK` with the body `ok`, independently of Redis availability.
- `GET /slow` waits three seconds before responding, demonstrating graceful shutdown.
- `reset` runs the same executable as a one-off administrative process, resets the visit counter in Redis, and exits.
- JSON structured logging to standard output.
- Graceful shutdown on `SIGTERM` and `SIGINT`.
- Kubernetes deployment with two application replicas, a ClusterIP Service, health probes, and resource limits.

## Prerequisites

Install the following tools:

- [Go](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kind](https://kind.sigs.k8s.io/) (Kubernetes in Docker)

The commands below assume a Linux shell, a local kind cluster, and the default Kubernetes namespace.

## 1. Clone the repository

```bash
git clone <REPOSITORY_URL>
cd <REPOSITORY_DIRECTORY>
```

Replace the placeholders with the repository URL and directory name.

## 2. Run the tests

Run the Go test suite:

```bash
go test ./...
```

## 3. Build the Docker image

Build the application image:

```bash
docker build -t first-hw:1.0.0 .
```

The Dockerfile compiles the Go application and packages the executable into a minimal `scratch` image.

## 4. Create a Kubernetes cluster

If you do not already have a suitable kind cluster, create one:

```bash
kind create cluster --name first-hw
```

Load the locally built image into the cluster:

```bash
kind load docker-image first-hw:1.0.0 --name first-hw
```

If you already have a cluster, ensure that kubectl is using the intended context:

```bash
kubectl config current-context
```

For the cluster created above, the context is `kind-first-hw`.

## 5. Deploy Redis and the application

Apply the Kubernetes manifests:

```bash
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/deployment.yaml
```

Wait for Redis and the application to become ready:

```bash
kubectl rollout status deployment/redis --timeout=120s
kubectl rollout status deployment/hello --timeout=120s
```

Check the running pods and services:

```bash
kubectl get pods -o wide
kubectl get services
kubectl get endpoints hello
```

The application Deployment should have at least two ready replicas. The `hello` Service routes HTTP traffic to those replicas, while both replicas connect to the same Redis Service.

## 6. Access the application

Forward a local port to the application Service:

```bash
kubectl port-forward service/hello 8080:80
```

Keep this terminal open. In another terminal, send requests:

```bash
curl http://localhost:8080/
curl http://localhost:8080/
curl http://localhost:8080/healthz
curl http://localhost:8080/slow
```

An example response from the root endpoint is:

```text
hello from the cluster (visit 1, served by hello-6998577895-6q6bv)
```

The visit number increases with each successful request to `/`. The hostname identifies the pod that served the request. Requests routed to different replicas share the same Redis counter.

The health endpoint returns:

```text
ok
```

The `/slow` endpoint takes approximately three seconds to respond.

## 7. Reset the visit counter

The application supports an administrative `reset` argument. Run it as a one-off Kubernetes pod using the same image and Redis configuration:

```bash
kubectl run hello-reset \
  --restart=Never \
  --image=first-hw:1.0.0 \
  --image-pull-policy=IfNotPresent \
  --env="REDIS_URL=redis://redis:6379/0" \
  -- reset
```

Wait for the process to finish:

```bash
kubectl wait pod/hello-reset \
  --for=jsonpath='{.status.phase}'=Succeeded \
  --timeout=60s
```

Inspect its log and exit code:

```bash
kubectl logs hello-reset

kubectl get pod hello-reset \
  -o jsonpath='{.status.phase}{" exit code: "}{.status.containerStatuses[0].state.terminated.exitCode}{"\n"}'
```

The process logs a successful reset and exits with code `0`. The next request to `/` should return `visit 1`.

Remove the completed administrative pod:

```bash
kubectl delete pod hello-reset --wait=true
```

The Redis key `visits` is set to `0`; the next root request increments it to `1`.

## 8. Configuration

The application is configured using environment variables.

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listening port |
| `GREETING` | `hello from my container` | Greeting returned by `GET /` |
| `REDIS_URL` | `redis://localhost:6379/0` | Redis connection URL |

The Kubernetes Deployment overrides these defaults with the cluster-specific greeting and Redis Service address.

## 9. Logging and graceful shutdown

The application writes structured JSON log entries to standard output. Request logs include the HTTP method, path, response status, and duration.

When the process receives `SIGTERM` or `SIGINT`, it stops accepting new connections and attempts to finish in-flight requests before shutting down. Kubernetes is configured with a termination grace period to allow this process to complete.

## 10. Kubernetes resources

| File | Purpose |
|---|---|
| `k8s/redis.yaml` | Redis Deployment and Service |
| `k8s/service.yaml` | ClusterIP Service exposing the application internally |
| `k8s/deployment.yaml` | Application Deployment, replicas, environment variables, probes, and resource limits |

The application Deployment defines startup, readiness, and liveness probes using `/healthz`. It also sets CPU and memory requests and limits.

## 11. Twelve-Factor App compliance

| # | Factor | Implementation in this project |
|---:|---|---|
| 1 | **Codebase** | One Git repository contains the Go application, tests, Dockerfile, and Kubernetes manifests. The repository represents one application codebase deployed to different environments. |
| 2 | **Dependencies** | Go dependencies are declared in `go.mod` and pinned through `go.sum`. The Docker build produces a self-contained executable in a minimal `scratch` image. |
| 3 | **Config** | Runtime configuration is supplied through `PORT`, `GREETING`, and `REDIS_URL` environment variables rather than hardcoded deployment-specific values. |
| 4 | **Backing services** | Redis is treated as an attached network service accessed through its URL. The application does not store the shared visit counter in its own process memory. |
| 5 | **Build, release, run** | The Docker build creates the application image, Kubernetes manifests describe its runtime configuration, and the Deployment runs the image. A fully automated, immutable release pipeline is not included. |
| 6 | **Processes** | The HTTP server runs as a stateless Go process. Application state is externalized to Redis, allowing application replicas to share the counter. |
| 7 | **Port binding** | The application listens on its configured HTTP port. Kubernetes exposes it internally through a Service, and `kubectl port-forward` provides local access during development. |
| 8 | **Concurrency** | Kubernetes runs multiple application replicas, allowing requests to be served concurrently and capacity to be increased by scaling the Deployment. |
| 9 | **Disposability** | The server starts quickly, handles termination signals, and gracefully completes in-flight requests. The `/slow` endpoint is used to test shutdown behavior. |
| 10 | **Dev/prod parity** | Docker provides a consistent application image, and Kubernetes runs that same image with Redis as a backing service. Separate development and production environments, and automated parity checks between them, are not provided. |
| 11 | **Logs** | The application emits structured JSON logs to standard output rather than relying on local log files. Container and Kubernetes tooling can collect these logs. |
| 12 | **Admin processes** | The `reset` argument runs a one-off administrative process using the same application image and Redis configuration. It resets the counter and exits without starting the HTTP server. |

## 12. Run the automated self-check

After building and deploying the application, run the provided checker:

```bash
bash scripts/check-12factor.sh first-hw:1.0.0
```
If the initial check fails and mentions the reset command not being right, then modify the check script like this:

1. find this line: `for i in 1 2 3 4 5 6 7 8; do curl -s -m 5 http://hello/; done`
2. Change it to this: `for i in 1 2 3 4 5 6 7 8; do curl -s -m 5 http://hello/; echo; done`

Then remove the pods created by the previous run and then run the script again.

This is needed because during testing, I found out that the checker incorrectly failed the traffic and reset checks because the HTTP responses weren't separated by newlines. I have opened a Github issue in the course repository addressing this problem.

The script checks the application in Docker and the deployed Kubernetes resources, including health endpoints, backing-service failure handling, graceful shutdown, logging, replicas, probes, environment configuration, the administrative reset process, and shared counter behavior.

## Cleanup

Delete the Kubernetes resources:

```bash
kubectl delete -f k8s/deployment.yaml
kubectl delete -f k8s/service.yaml
kubectl delete -f k8s/redis.yaml
```

If you created the kind cluster specifically for this project and no longer need it:

```bash
kind delete cluster --name first-hw
```

This removes the cluster and its resources.

## 13. What is not included

This project intentionally focuses on the core requirements of the assignment rather than providing a production-ready database-backed service.

- **Durable storage:** The application does not implement its own disk-based persistence. The visit counter is stored in Redis, and the current deployment does not configure durable Redis storage. Consequently, the counter is not guaranteed to survive Redis restarts, pod replacement, or cluster recreation.

- **Unique visitor tracking:** The current implementation counts requests to the root endpoint rather than unique visitors. Redis HyperLogLog could be used to estimate the number of distinct visitors with low memory overhead. However, this would require defining how visitors are identified, for example through client-provided identifiers or application-issued tokens. This functionality was considered but left out of scope for the current assignment.