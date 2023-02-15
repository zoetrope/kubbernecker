[![GitHub release](https://img.shields.io/github/release/zoetrope/kubbernecker.svg?maxAge=60)](https://github.com/zoetrope/kubbernecker/releases)
[![CI](https://github.com/zoetrope/kubbernecker/actions/workflows/ci.yaml/badge.svg)](https://github.com/zoetrope/kubbernecker/actions/workflows/ci.yaml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/zoetrope/kubbernecker?tab=overview)](https://pkg.go.dev/github.com/zoetrope/kubbernecker?tab=overview)

# Kubbernecker

**Project Status**: Alpha

Kubbernecker is tools that helps to check the number of changes made to Kubernetes resources.
It provides two tools: `kubbernecker-metrics` and `kubectl-kubbernecker`.

`kubbernecker-metrics` is an exporter that exposes the number of changes made to Kubernetes resources as Prometheus
metrics.
It helps to monitor the changes made to resources within a Kubernetes cluster.

`kubectl-kubbernecker` is a kubectl plugin that shows the number of changes made to Kubernetes resources and the manager
who made the changes.
It helps to quickly check the changes made to resources within a Kubernetes cluster.

The name of Kubbernecker comes from rubbernecker.
It is like staring at a fight between Kubernetes controllers.

## Motivation

In a Kubernetes cluster, different controllers may continuously edit the same resource, leading to a race condition.
It can cause increased loads on kube-apiserver and performance issues.
Kubbernecker helps to solve these problems by checking the number of changes made to Kubernetes resources.

## Installation

### kubbernecker-metrics

You need to add this repository to your Helm repositories:

```console
$ helm repo add kubbernecker https://zoetrope.github.io/kubbernecker/
$ helm repo update
```

To install the chart with the release name `kubbernecker` using a dedicated namespace(recommended):

```
$ helm install --create-namespace --namespace kubbernecker kubbernecker kubbernecker/kubbernecker
```

Specify parameters using `--set key=value[,key=value]` argument to `helm install`.
Alternatively a YAML file that specifies the values for the parameters can be provided like this:

```console
$ helm install --create-namespace --namespace kubbernecker kubbernecker -f values.yaml kubbernecker/kubbernecker
```

Values:

| Key                           | Type   | Default                                       | Description                                                                                                                                             |
|-------------------------------|--------|-----------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------|
| image.repository              | string | `"ghcr.io/zoetrope/kubbernecker"`             | Kubbernecker image repository to use.                                                                                                                   |
| image.tag                     | string | `{{ .Chart.AppVersion }}`                     | Kubbernecker image tag to use.                                                                                                                          |
| image.imagePullPolicy         | string | `IfNotPresent`                                | imagePullPolicy applied to Kubbernecker image.                                                                                                          |
| resources                     | object | `{"requests":{"cpu":"100m","memory":"20Mi"}}` | Specify resources.                                                                                                                                      |
| config.targetResources        | list   | `[]` (See [values.yaml])                      | Target Resources. If this is empty, all resources will be the target.                                                                                   |
| config.namespaceSelector      | list   | `{}` (See [values.yaml])                      | Selector of the namespace to which the target resource belongs. If this is empty, all namespaces will be the target.                                    |
| config.enableClusterResources | bool   | `false`                                       | If `targetResources` is empty, whether to include cluster-scope resources in the target. If `targetResources` is not empty, this field will be ignored. |

### kubectl-kubbernecker

Download the binary and put it in a directory of your `PATH`.
The following is an example to install the plugin in `/usr/local/bin`.

```console
$ OS=$(go env GOOS)
$ ARCH=$(go env GOARCH)
$ curl -L -sS https://github.com/zoetrope/kubbernecker/releases/latest/download/kubectl-kubbernecker_${OS}-${ARCH}.tar.gz \
  | tar xz -C /usr/local/bin kubectl-kubbernecker
```

NOTE: In the future, this tool will be able to be installed by [krew](https://krew.sigs.k8s.io).

## Usage

### kubbernecker-metrics

| Name                                  | Type    | Description                                                 | Labels                                                                                                                   |
|---------------------------------------|---------|-------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------|
| `kubbernecker_resource_events_total`  | counter | Total number of Kubernetes events by resource type.         | `resource_type`: resource type </br> `namespace`: namespace </br> `event_type`: event type ("add", "update" or "delete") |
| `kubbernecker_resource_updates_total` | counter | Total number of updates for a Kubernetes resource instance. | `resource_type`: resource type </br> `namespace`: namespace </br> `resource_name`: resource name                         |

### kubectl-kubbernecker

- watch sub-command

```console
$ kubectl kubbernecker watch --all-resources --all-namespaces
```

- blame sub-command

```console
$ kubectl kubbernecker blame -n default pod nginx
```

## Development

```console
$ make start-dev
$ tilt up
```

[values.yaml]: ./charts/kubbernecker/values.yaml
