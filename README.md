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

`kubbernecker-metrics` exposes the following metrics:

| Name                                 | Type    | Description                                      | Labels                                                                                                                                                                                   |
|--------------------------------------|---------|--------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `kubbernecker_resource_events_total` | counter | Total number of events for Kubernetes resources. | `group`: group </br> `version`: version </br> `kind`: kind </br>`namespace`: namespace </br> `event_type`: event type ("add", "update" or "delete") </br> `resource_name`: resource name |

### kubectl-kubbernecker

`kubectl-kubbernecker` has two subcommands:

`watch` sub-command prints the number of times a resource is updated.

```console
$ kubectl kubbernecker watch -n default configmap
{
  "gvk": {
    "group": "",
    "version": "v1",
    "kind": "ConfigMap"
  },
  "namespaces": {
    "default": {
      "resources": {
        "test-cm": {
          "add": 0,
          "delete": 0,
          "update": 9
        }
      }
    }
  }
}
```

`blame` sub-command prints the name of managers that updated the given resource.

```console
$ kubectl kubbernecker blame -n default configmap test-cm
{
  "managers": {
    "manager1": {
      "update": 4
    },
    "manager2": {
      "update": 4
    }
  },
  "lastUpdate": "2023-02-17T22:25:20+09:00"
}
```

## Development

Tools for developing kubbernecker are managed by aqua.
Please install aqua as described in the following page:

https://aquaproj.github.io/docs/reference/install

Then install the tools.

```console
$ cd /path/to/kubbernecker
$ aqua i -l
```

You can start development with tilt.

```console
$ make start-dev
$ tilt up
```

[values.yaml]: ./charts/kubbernecker/values.yaml
