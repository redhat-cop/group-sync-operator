Group Sync Operator
===================

Synchronizes groups from external providers into OpenShift

## Overview

The OpenShift Container Platform contains functionality to synchronize groups found in external identity providers into the platform. Currently, this functionality is limited to LDAP only. This operator is designed to integrate with external providers in order to provide new solutions.

Group Synchronization is facilitated by creating a `GroupSync` resource within a namespace. The following describes the high level schema for this resource:

```shell
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: example-groupsync
  namespace: group-sync-operator
spec:
  providers:
    - <One or more providers to synchronize>
```

## Deploying the Operator

Use the following steps to deploy the operator to an OpenShift cluster

1. Assuming that you are authenticated to the cluster, create a new project called `group-sync-operator`
.
```shell
oc new-project group-sync-operator
```
.
2. Clone the project locally and changed into the project
.
.
```shell
git clone https://github.com/redhat-cop/group-sync-operator.git
cd group-sync-operator
oc -n group-sync-operator apply -f deploy
```

## Providers

Integration with external systems is made possible through a set of plugable external providers. The following providers are currently supported:

* [Keycloak](https://www.keycloak.org/)/[Red Hat Single Sign On](https://access.redhat.com/products/red-hat-single-sign-on)

The following sections describe the configuration options provided by each provider

### Keycloak

Groups stored within Keycloak can be synchronized into OpenShift. The following table describes the set of configuration options for the Keycloak provider:

| Name | Description | Defaults | Required | 
| ----- | ---------- | -------- | ----- |
| `url` | URL Location for Keycloak | | Yes |
| `loginRealm` | Realm to authenticate against | `master` | No |
| `realm` | Realm to synchronize | | Yes |
| `secretName` | Name of the secret containing authentication details (See below) | | Yes |
| `insecure` | Ignore SSL verification | 'false' | No |
| `scope` | Scope for group synchronization. Options are `one` for one level or `sub` to include subgroups | `sub` | No |

The following is an example of a minimal configuration that can be applied to integrate with a Keycloak provider:

```shell
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: keycloak-groupsync
  namespace: group-sync-operator
spec:
  providers:
  - keycloak:
      realm: ocp
      secretName: keycloak-group-sync
      url: https://keycloak-keycloak-operator.apps.openshift.com
```

#### Authenticating to Keycloak

A secret must be created in the same namespace that contains the `GroupSync` resource. It must contain the following keys:

* `username` - Username for authenticating with Keycloak
* `password` - Password for authenticating with Keycloak

To specify the TLS certificates that should be used to communicate with Keycloak, add the certificates to `ca.crt` key 

## Sync Period

To specify the period for which synchronization should occur on a regular basis, the `syncPeriodMinutes` field can be set as described below



## Local Development

Execute the following steps to develop the functionality locally. It is recommended that development be done using a cluster with `cluster-admin` permissions.

```shell
go mod download
```

optionally:

```shell
go mod vendor
```

Using the [operator-sdk](https://github.com/operator-framework/operator-sdk), run the operator locally:

```shell
oc apply -f deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml
OPERATOR_NAME='group-sync-operator' operator-sdk run --local --watch-namespace ""
