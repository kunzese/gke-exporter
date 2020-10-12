# gke-exporter

![CI](https://github.com/kunzese/gke-exporter/workflows/CI/badge.svg)
![Release](https://github.com/kunzese/gke-exporter/workflows/Release/badge.svg)

## Docker Hub

<https://hub.docker.com/r/kunzese/gke-exporter>

![Docker Image Version (latest semver)](https://img.shields.io/docker/v/kunzese/gke-exporter?sort=semver)
![Docker Pulls](https://img.shields.io/docker/pulls/kunzese/gke-exporter)

## Instructions

If you plan to use this exporter in a Google Cloud Platform GKE environment you will need a service account with at least Kubernetes Engine Cluster Viewer (`roles/container.clusterViewer`) permission.

### Create new service account

```shell
$ gcloud iam service-accounts create gke-exporter --display-name "gke-exporter service account"
Created service account [gke-exporter].
```

### Add `roles/container.clusterViewer` permission on project level

```shell
$ gcloud projects add-iam-policy-binding <PROJECT_ID> \
    --member='serviceAccount:gke-exporter@<PROJECT_ID>.iam.gserviceaccount.com' \
    --role='roles/container.clusterViewer'
Updated IAM policy for project [<PROJECT_ID>].
bindings:
...
- members:
  - serviceAccount:gke-exporter@<PROJECT_ID>.iam.gserviceaccount.com
  role: roles/container.clusterViewer
...
etag: BwWxeBHam4F=
version: 1
```

### Create and export private key for service account

```shell
$ gcloud iam service-accounts keys create ~/key.json \
    --iam-account gke-exporter@$<PROJECT_ID>.iam.gserviceaccount.com
created key [e31ada6e2657fde3296e51c4199e6c90158d01a0] of type [json] as [/home/demo/key.json] for [gke-exporter@<PROJECT_ID>.iam.gserviceaccount.com
```

### Add private key as Kubernetes Secret

```shell
$ kubectl create secret generic gke-exporter --from-file=key.json=/home/demo/key.json
secret/gke-exporter created
```

### Kubernetes Deployment + Service

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: gke-exporter
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      run: gke-exporter
  template:
    metadata:
      labels:
        run: gke-exporter
    spec:
      securityContext:
        fsGroup: 2000
        runAsUser: 1000
        runAsNonRoot: true
      containers:
        - name: gke-exporter
          image: kunzese/gke-exporter:1.0.1
          ports:
            - containerPort: 8080
              protocol: TCP
          volumeMounts:
            - name: gke-exporter
              readOnly: true
              mountPath: /gke-exporter
          env:
            - name: GOOGLE_APPLICATION_CREDENTIALS
              value: /gke-exporter/key.json
      volumes:
        - name: gke-exporter
          secret:
            secretName: gke-exporter
---
apiVersion: v1
kind: Service
metadata:
  name: gke-exporter
  namespace: monitoring
spec:
  ports:
    - port: 8080
      protocol: TCP
      targetPort: 8080
  selector:
    run: gke-exporter
  type: NodePort
```

### Prometheus.yaml

```yaml
global:
  scrape_interval: 5s
  evaluation_interval: 5s

scrape_configs:
  - job_name: 'gke-exporter'
    honor_labels: true
    honor_timestamps: true
    scrape_interval: 5m
    scrape_timeout: 30s
    metrics_path: /metrics
    scheme: http
    static_configs:
      - targets:
        - gke-exporter.monitoring.svc:8080
```
