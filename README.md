# external-metrics-gateway (MVP)

Single owner of `external.metrics.k8s.io` for the cluster, with routing to multiple backends.

## What this service does

- Exposes External Metrics API endpoints:
  - `GET /apis/external.metrics.k8s.io/v1beta1`
  - `GET /apis/external.metrics.k8s.io/v1beta1/namespaces/{namespace}/{metric}`
- Routes requests by metric prefix.
- Supports two backends:
  - Prometheus (instant query)
  - KEDA external metrics API proxy

## Routing

- `prom_http_requests_total` -> Prometheus metric `http_requests_total`
- `keda_queueLength` -> KEDA metric `queueLength`

## Config

Copy `config.example.yaml` to `config.yaml` and adjust URLs.

## Run locally

```bash
go mod tidy
go run ./cmd/gateway -config ./config.yaml
```

## Kubernetes APIService ownership

There must be only one active owner for `external.metrics.k8s.io/v1beta1`.
This gateway is intended to be that owner.

## Kubernetes manifests

`manifests/` contains:

- Namespace
- ConfigMap
- cert-manager Issuer/Certificate
- ServiceAccount / RBAC
- Deployment
- Service
- APIService (`v1beta1.external.metrics.k8s.io`)

Apply:

```bash
kubectl apply -k manifests/
```

Prerequisites:

- cert-manager installed (serving cert + CA injection)
- image in `manifests/40-deployment.yaml` points to your built image
- no other active `external.metrics.k8s.io/v1beta1` owner

Check APIService:

```bash
kubectl get apiservice v1beta1.external.metrics.k8s.io -o yaml
```

## Query behavior

Prometheus query is built as:

`<metric>{namespace="<ns>",...labelSelector}`

Example:

`/apis/external.metrics.k8s.io/v1beta1/namespaces/prod/prom_http_requests_total?labelSelector=app=my-api`

becomes:

`http_requests_total{namespace="prod",app="my-api"}`

## Minikube runbook (validated)

This project was validated in Minikube with the following practical flow.

### 1) Deploy Prometheus and KEDA

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add kedacore https://kedacore.github.io/charts
helm repo update

kubectl create ns monitoring --dry-run=client -o yaml | kubectl apply -f -
kubectl create ns keda --dry-run=client -o yaml | kubectl apply -f -

helm upgrade --install prom prometheus-community/prometheus -n monitoring
helm upgrade --install keda kedacore/keda -n keda
```

### 2) Build and deploy gateway

```bash
minikube image build -t external-metrics-gateway:dev .
kubectl apply -k manifests/
```

### 3) Point APIService to this gateway

If `v1beta1.external.metrics.k8s.io` is currently owned by KEDA, patch it:

```bash
kubectl patch apiservice v1beta1.external.metrics.k8s.io --type='merge' -p '{
  "spec":{
    "service":{"namespace":"external-metrics","name":"external-metrics-gateway","port":443}
  }
}'
```

### 4) Verify discovery

```bash
kubectl get apiservice v1beta1.external.metrics.k8s.io
kubectl get --raw /apis/external.metrics.k8s.io/v1beta1 | jq .
```

Expected:

- APIService `AVAILABLE=True`
- discovery returns `APIResourceList`

### 5) Verify end-to-end metric query

```bash
kubectl get --raw '/apis/external.metrics.k8s.io/v1beta1/namespaces/kube-system/prom_up' | jq .
```

Expected: non-empty `items` list.

## HPA smoke test (validated)

Use a metric available in `default` namespace, for example `prom_container_memory_usage_bytes`.

```bash
cat <<'YAML' | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hpa-demo
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hpa-demo
  template:
    metadata:
      labels:
        app: hpa-demo
    spec:
      containers:
      - name: app
        image: nginx:1.27-alpine
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: hpa-demo-external
  namespace: default
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: hpa-demo
  minReplicas: 1
  maxReplicas: 3
  metrics:
  - type: External
    external:
      metric:
        name: prom_container_memory_usage_bytes
      target:
        type: AverageValue
        averageValue: "500Mi"
YAML

kubectl -n default describe hpa hpa-demo-external
kubectl -n default get deploy hpa-demo
```

Expected:

- `ScalingActive=True`
- `SuccessfulRescale` event
- deployment scales above 1 replica

Cleanup:

```bash
kubectl -n default delete hpa hpa-demo-external --ignore-not-found
kubectl -n default delete deploy hpa-demo --ignore-not-found
```

## Troubleshooting

### APIService is not Available

- Check owner and backend service:

```bash
kubectl get apiservice v1beta1.external.metrics.k8s.io -o yaml
```

- Check gateway pod and logs:

```bash
kubectl -n external-metrics get pods -o wide
kubectl -n external-metrics logs deploy/external-metrics-gateway --tail=200
```

### `http: server gave HTTP response to HTTPS client`

Cause: API aggregation talks HTTPS, but gateway endpoint is serving HTTP.

Fix: run gateway with TLS and use HTTPS probes.

### Pod is restarting after TLS enablement

Cause: probes still use `HTTP` scheme.

Fix: set readiness/liveness probe scheme to `HTTPS`.

### External metric returns `items: []`

Expected if there are no series matching namespace + selector.

Test with a namespace known to have data (`kube-system` worked in validation) or with a different metric.

### Prometheus DNS errors from gateway

Example:

`lookup prometheus-server.monitoring.svc.cluster.local: no such host`

Fix: point config to your actual service name (for Helm release `prom`):

`http://prom-prometheus-server.monitoring.svc.cluster.local`

## Limitations (MVP)

- Label selector parser supports only simple `k=v,k2=v2`.
- No auth/retry policy yet.
- KEDA backend is a direct pass-through endpoint.
