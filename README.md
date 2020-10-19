# gke-exporter

<https://hub.docker.com/r/kunzese/gke-exporter>

![CI](https://github.com/kunzese/gke-exporter/workflows/CI/badge.svg)
![Release](https://github.com/kunzese/gke-exporter/workflows/Release/badge.svg)
![Docker Image Version (latest semver)](https://img.shields.io/docker/v/kunzese/gke-exporter?sort=semver)
![Docker Pulls](https://img.shields.io/docker/pulls/kunzese/gke-exporter)

## Description

This exporter provides two metrics, `gke_master_version_count` and `gke_master_unsupported_versions_count`.

### gke_master_version_count

Number of GKE clusters, partitioned by the version of their master node.

```text
# HELP gke_master_version_count Number of GKE clusters, partitioned by the version of their master node.
# TYPE gke_master_version_count counter
gke_master_version_count{version="1.14.10-gke.42"} 1
gke_master_version_count{version="1.14.10-gke.46"} 2
gke_master_version_count{version="1.14.10-gke.50"} 10
gke_master_version_count{version="1.15.12-gke.2"} 5
gke_master_version_count{version="1.15.12-gke.20"} 1
gke_master_version_count{version="1.16.11-gke.5"} 1
gke_master_version_count{version="1.16.13-gke.1"} 10
gke_master_version_count{version="1.16.13-gke.401"} 100
gke_master_version_count{version="1.16.15-gke.500"} 78
gke_master_version_count{version="1.17.9-gke.1500"} 23
gke_master_version_count{version="1.17.9-gke.1504"} 9
```

### gke_master_unsupported_versions_count

Number of GKE clusters with unsupported master versions, partitioned by the location, project and version of their master node.

```text
# HELP gke_master_unsupported_versions_count Number of GKE clusters with unsupported master versions, partitioned by the location, project and version of their master node.
# TYPE gke_master_unsupported_versions_count counter
gke_master_unsupported_versions_count{location="europe-west4",name="cluster1",project_id="demo-project-1",project_name="Demo Project 1",version="1.16.11-gke.5"} 1
gke_master_unsupported_versions_count{location="europe-west4",name="cluster2",project_id="demo-project-2",project_name="Demo Project 2",version="1.16.11-gke.5"} 1
gke_master_unsupported_versions_count{location="europe-west4",name="cluster3",project_id="demo-project-3",project_name="Demo Project 3",version="1.14.10-gke.42"} 1
gke_master_unsupported_versions_count{location="europe-west4",name="cluster4",project_id="demo-project-4",project_name="Demo Project 4",version="1.16.13-gke.1"} 10
gke_master_unsupported_versions_count{location="europe-west4-a",name="cluster5",project_id="demo-project-5",project_name="Demo Project 5",version="1.15.12-gke.2"} 5
```

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
