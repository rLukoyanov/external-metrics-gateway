# external-metrics-gateway (MVP)

Single `external.metrics.k8s.io` provider for Kubernetes, with internal routing to multiple backends.

## What this MVP does

- Exposes External Metrics API endpoints:
  - `GET /apis/external.metrics.k8s.io/v1beta1`
  - `GET /apis/external.metrics.k8s.io/v1beta1/namespaces/{namespace}/{metric}`
- Routes metric requests by metric prefix.
- Supports two backends:
  - Prometheus (instant query)
  - KEDA external metrics API proxy

## Routing example

- `prom_http_requests_total` -> Prometheus (`http_requests_total`)
- `keda_queueLength` -> KEDA (`queueLength`)

## Config

Copy `config.example.yaml` to `config.yaml` and adjust URLs.

## Run locally

```bash
go mod tidy
go run ./cmd/gateway -config ./config.yaml
```

## Kubernetes APIService note

In the cluster there must be only one owner of `external.metrics.k8s.io`. This service is intended to be that single owner.

## Kubernetes manifests

Manifests are in `manifests/` and include:

- Namespace
- ConfigMap with routing config
- cert-manager Issuer/Certificate
- ServiceAccount
- Deployment
- Service
- APIService (`v1beta1.external.metrics.k8s.io`)

Apply:

```bash
kubectl apply -k manifests/
```

Prerequisites:

- cert-manager must be installed (for serving cert + CA injection)
- image in `manifests/40-deployment.yaml` must point to your built gateway image
- there must not be another active owner for `external.metrics.k8s.io/v1beta1`

Check APIService status:

```bash
kubectl get apiservice v1beta1.external.metrics.k8s.io -o yaml
```

## Prometheus query behavior in MVP

MVP builds query as:

`<metric>{namespace="<ns>",...labelSelector}`

Example:

`/apis/external.metrics.k8s.io/v1beta1/namespaces/prod/prom_http_requests_total?labelSelector=app=my-api`

becomes:

`http_requests_total{namespace="prod",app="my-api"}`

## Limitations (MVP)

- Label selector parser supports only simple `k=v,k2=v2`.
- No auth/TLS/retry policy yet.
- KEDA backend is a direct pass-through to KEDA external metric endpoint.
