# group-sync-operator Helm Chart


## Introduction

This chart installs the Group Sync Operator to an OpenShift environment.

## Installing the Chart

To install the chart with the release name `group-sync-operator`:

```console
$ helm repo add redhat-cop https://redhat-cop.github.io/helm-charts
$ helm install group-sync-operator redhat-cop/group-sync-operator
```

## Uninstalling the Chart

To uninstall/delete the `group-sync-operator` deployment:

```console
$ helm delete group-sync-operator
```

## Parameters

The following tables lists the configurable parameters of the Group Sync Operator chart and their default values.

| Parameter                                      | Description                                                                                                                                                                                                                    | Default                                                      |
|------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------|
| `nameOverride`                                 | String to partially override group-sync-operator.fullname template with a string (will prepend the release name)                                                                                                                           | `nil`                                                        |
| `fullnameOverride`                             | String to fully override the group-sync-operator.fullname template with a string                                                                                                                                                               | `nil`                                                        |
| `image.registry`                         | Container image registry                                                                                                                                                                                                   | `quay.io`                                                        |
| `image.repository`                         | Container image repository                                                                                                                                                                                                   | `redhat-cop/group-sync-operator`                                                        |
| `image.tag`                         | Container image tag                                                                                                                                                                                                   | `v<Chart App Version>`                                                        |
| `image.pullPolicy`                             | Container image pull policy                                                                                                                                                                                                      | `Always`                                               |
| `imagePullSecrets`                      | Container registry secret names as an array                                                                                                                                                                                | `[]` (does not add image pull secrets to deployed pods)      |                                      
| `securityContext`                      | Container security context                                                                                                                                                                                                        | `{}` |
| `resources`                      | Container resources                                                                                                                                                                                                        | `{}` || `nodeSelector`                                 | Node labels for pod assignment                                                                                                                                                                                                 | `{}`                                                         |
| `tolerations`                                  | Toleration labels for pod assignment                                                                                                                                                                                           | `[]`                                                         |
| `affinity`                                     | Map of node/pod affinities                                                                                                                                                                                                     | `{}`                                                         |
Specify each parameter using the `--set key=value[,key=value]` argument to `helm install` or `helm upgrade`. For example,

```console
$ helm install group-sync-operator \
               --set image.registry=myregistry.example.com \
               --set image.repository=helm/group-sync-operator \
               redhat-cop/group-sync-operator
```

The above command sets the location of the Group Sync Operator image.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```console
$ helm install group-sync-operator -f values.yaml redhat-cop/group-sync-operator
```

Once deployed, a `GroupSync` Custom Resource can be created by following the instructions and parameters in the project [repository](https://github.com/redhat-cop/group-sync-operator/)