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
oc -n group-sync-operator --recursive=true -f deploy
```

## Providers

Integration with external systems is made possible through a set of plugable external providers. The following providers are currently supported:

* [Azure](https://azure.microsoft.com/)
* [GitHub](https://github.com)
* [GitLab](https://gitlab.com)
* [Keycloak](https://www.keycloak.org/)/[Red Hat Single Sign On](https://access.redhat.com/products/red-hat-single-sign-on)

The following sections describe the configuration options available for each provider


### Azure

Groups contained within Azure Active Directory can be synchronized into OpenShift. The following table describes the set of configuration options for the Azure provider:

| Name | Description | Defaults | Required | 
| ----- | ---------- | -------- | ----- |
| `credentialsSecretName` | Name of the secret containing authentication details (See below) | | Yes |
| `groups` | List of groups to filter against | | No |

The following is an example of a minimal configuration that can be applied to integrate with a Github provider:

```shell
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: azure-groupsync
  namespace: group-sync-operator
spec:
  providers:
  - name: azure
    azure:
      credentialsSecretName: azure-group-sync
```

#### Authenticating to Azure

Authentication to Azure can be performed using Service Principal with access to query group information in Azure Active Directory. A secret must be created in the same namespace that contains the `GroupSync` resource:

The following keys must be defined in the secret

* `AZURE_SUBSCRIPTION_ID` - Subscription ID
* `AZURE_TENANT_ID` - Tenant ID
* `AZURE_CLIENT_ID` - Client ID
* `AZURE_CLIENT_SECRET` - Client Secret

The secret can be created by executing the following command:

```shell
oc create secret generic azure-group-sync --from-literal=AZURE_SUBSCRIPTION_ID=<AZURE_SUBSCRIPTION_ID> --from-literal=AZURE_TENANT_ID=<AZURE_TENANT_ID> --from-literal=AZURE_CLIENT_ID=<AZURE_CLIENT_ID> --from-literal=AZURE_CLIENT_SECRET=<AZURE_CLIENT_SECRET>
```

### GitHub

Teams stored within a GitHub organization can be synchronized into OpenShift. The following table describes the set of configuration options for the GitHub provider:

| Name | Description | Defaults | Required | 
| ----- | ---------- | -------- | ----- |
| `caSecretRef` | Reference  to a secret containing a SSL certificate to use for communication (See below) | | No |
| `credentialsSecretName` | Name of the secret containing authentication details (See below) | | Yes |
| `insecure` | Ignore SSL verification | `false` | No |
| `organization` | Organization to synchronize against | | Yes |
| `teams` | List of teams to filter against | | No |
| `url` | Base URL for the GitHub or GitHub Enterprise host (Must contain a trailing slash) | | No |


The following is an example of a minimal configuration that can be applied to integrate with a Github provider:

```shell
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: github-groupsync
  namespace: group-sync-operator
spec:
  providers:
  - name: github
    github:
      organization: ocp
      credentialsSecretName: github-group-sync
```

#### Authenticating to GitHub

Authentication to GitHub can be performed using an OAuth Personal Access Token or a Username and Password (Note: 2FA not supported). A secret must be created in the same namespace that contains the `GroupSync` resource:

When using an OAuth token, the following key is required:

* `token` - OAuth token

The secret can be created by executing the following command:

```shell
oc create secret generic github-group-sync --from-literal=token=<token>
```


The following keys are required for username and password:

* `username` - Username for authenticating with GitHub
* `password` - Password for authenticating with GitHub

The secret can be created by executing the following command:

```shell
oc create secret generic github-group-sync --from-literal=username=<username> --from-literal=password=<password>
```

### GitLab

Groups stored within a GitLab can be synchronized into OpenShift. The following table describes the set of configuration options for the GitLab provider:

| Name | Description | Defaults | Required | 
| ----- | ---------- | -------- | ----- |
| `caSecretRef` | Reference  to a secret containing a SSL certificate to use for communication (See below) | | No |
| `credentialsSecretName` | Name of the secret containing authentication details (See below) | | Yes |
| `insecure` | Ignore SSL verification | 'false' | No |
| `groups` | List of groups to filter against | | No |
| `url` | Base URL for the GitLab instance | `https://gitlab.com` | No |


The following is an example of a minimal configuration that can be applied to integrate with a Github provider:

```shell
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: gitlab-groupsync
  namespace: group-sync-operator
spec:
  providers:
  - name: gitlab
    gitlab:
      credentialsSecretName: gitlab-group-sync
```

#### Authenticating to GitLab

Authentication to GitLab can be performed using an OAuth Personal Access Token or a Username and Password (Note: 2FA not supported). A secret must be created in the same namespace that contains the `GroupSync` resource:

When using an OAuth token, the following key is required:

* `token` - OAuth token

The secret can be created by executing the following command:

```shell
oc create secret generic gitlab-group-sync --from-literal=token=<token>
```


The following keys are required for username and password:

* `username` - Username for authenticating with GitLab
* `password` - Password for authenticating with GitLab

The secret can be created by executing the following command:

```shell
oc create secret generic gitlab-group-sync --from-literal=username=<username> --from-literal=password=<password>
```

### Keycloak

Groups stored within Keycloak can be synchronized into OpenShift. The following table describes the set of configuration options for the Keycloak provider:

| Name | Description | Defaults | Required | 
| ----- | ---------- | -------- | ----- |
| `caSecretRef` | Reference  to a secret containing a SSL certificate to use for communication (See below) | | No |
| `credentialsSecretName` | Name of the secret containing authentication details (See below) | | Yes |
| `groups` | List of groups to filter against | | No |
| `insecure` | Ignore SSL verification | 'false' | No |
| `loginRealm` | Realm to authenticate against | `master` | No |
| `realm` | Realm to synchronize | | Yes |
| `scope` | Scope for group synchronization. Options are `one` for one level or `sub` to include subgroups | `sub` | No |
| `url` | URL Location for Keycloak | | Yes |


The following is an example of a minimal configuration that can be applied to integrate with a Keycloak provider:

```shell
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: keycloak-groupsync
  namespace: group-sync-operator
spec:
  providers:
  - name: keycloak
    keycloak:
      realm: ocp
      credentialsSecretName: keycloak-group-sync
      url: https://keycloak-keycloak-operator.apps.openshift.com
```

#### Authenticating to Keycloak

A secret must be created in the same namespace that contains the `GroupSync` resource. It must contain the following keys:

* `username` - Username for authenticating with Keycloak
* `password` - Password for authenticating with Keycloak

## CA Certificates

Each provider allows for certificates to be provided in a secret to communicate to the target host. The secret must be placed in the same namespace as the `GroupSync`. An example of how a CA certificate for the Keycloak provider can be found below:


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
      credentialsSecretName: keycloak-group-sync
      url: https://keycloak-keycloak-operator.apps.openshift.com
      caSecretRef:
        name: keycloak-certs
        key: tls.crt
```


## Scheduled Execution

A cron style expression can be specified for which a synchronization event will occur. The following specifies that a synchronization should occur nightly at 3AM


```shell
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: keycloak-groupsync
  namespace: group-sync-operator
spec:
  schedule: "0 3 * * *"
  providers:
  - ...
```

If a schedule is not provided, synchronization will occur only when the object is reconciled by the platform.


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
```