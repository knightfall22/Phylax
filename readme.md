# Phylax - Sensor Monitoring System

![AI Assistance](https://img.shields.io/badge/Documentation-Assisted%20by%20Gemini-blue?style=flat-square)

Phylax is a high-velocity IoT monitoring platform designed to ingest, process, and store massive streams of sensor data with minimal latency. Built in Go, it leverages NATS JetStream to decouple rapid ingestion from storage, buffering millions of events before efficiently flushing them to PostgreSQL in optimized batches using `pgx` copy protocols. The system is architected for scale on Kubernetes using Helm charts and CloudNativePG, with a comprehensive observability layer powered by Prometheus and Grafana to track throughput, data lag, and sensor health in real-time.

## Tech Stack

- Protobuf - Compact message serialization.
- NATS JetStream - Low-latency, persistent message streaming.
- K8s and Helm - Container orchestration and package management.
- CloudNativePG: Production-grade PostgreSQL automation on K8s.

## Architecture

![System Architecture](assets/architecture.png)

- **NATS Server:** Serves as an ingestion layer. Provinding a light-weight persistent message broker.
- **Processor:** Consumes messages from NATS stream and processes batches of messages concurrently using a worker pool. Batches are flushed to the DB when a memory limit is exceeded or after a time interval. Processors exposes metrics to be scraped by Prometheus and Grafana.
- **Postgress Cluster:** Is controlled by CloudNativePG and stores raw sensor data. To conserve space an aggregation CronJob compresses raw data into hourly summaries and purges old raw records

## Getting Started

### 1. Prerequisites

Ensure you have the following installed:

- Docker Desktop (with Kubernetes enabled) or Minikube.
- Go (1.23+).
- Helm (Package manager for Kubernetes).
- Kubectl (Kubernetes CLI).

### 2. Install the Postgres Operator

We first install the CloudNativePG Operator. This is a cluster-wide controller that manages our database instances. We install it separately to avoid "ownership" conflicts during redeployments.

```bash
# Add the repo and install the operator
helm repo add cnpg https://cloudnative-pg.io/charts
helm repo update
helm install cnpg cnpg/cloudnative-pg \
  --namespace cnpg-system --create-namespace --wait
```

### 3. Build & Prepare the Application

```bash
# Build the Docker image locally
docker build -t phylax:latest .
```

### 4. Deploy the Stack

We use Helm chart to deploy NATS, Prometheus, Grafana, the Database Cluster, and the Phylax Application.

```bash
# 1. Download dependencies (NATS, Prometheus) into the charts/ folder
helm dependency build ./deploy

# 2. Install the full stack
helm install phylax ./deploy -f ./deploy/values.yaml
```

### 5. Run Simulator

Since the Simulator runs locally on your machine while the NATS broker runs inside Kubernetes, you must expose the connection and update the simulator's configuration.

1. Expose NATS to Localhost:
   Open a new terminal window and create a tunnel to the NATS service:

```bash
# Forward local port 4222 to the NATS service in Kubernetes
kubectl port-forward svc/phylax-nats 4222:4222
```

2. Update the Configuration:
   Open simulator/simulation-config.yaml and ensure the nats_url points to your local forwarded port:

```bash
# simulator/simulation-config.yaml
nats_url: "nats://localhost:4222"
```

3. Run Simulator:
   Now that the everything is ready, start the simulator to flood the system with synthetic data.

```bash
# Run the simulator locally
go run cmd/simulator/main.go
```

### 6. Access Observability (Grafana)

```bash
# 1. Port-forward Grafana to your local browser
# (Note: Adjust the service name if yours differs, check with 'kubectl get svc')
kubectl port-forward svc/phylax-kube-prometheus-stack-grafana 8080:80
```

Import preconfigured dashboard from the json file in the dashboard folder

**Password:** prom-operator

## Dashboard

![Dashboard](assets/dash.gif)

### Normal scenario

![Normal Scenario](assets/regular.png)

### Disaster scenario

![Disaster Scenario](assets/disaster.png)
