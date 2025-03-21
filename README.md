Group Sync Operator
===================

![build status](https://github.com/redhat-cop/group-sync-operator/workflows/push/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/redhat-cop/group-sync-operator)](https://goreportcard.com/report/github.com/redhat-cop/group-sync-operator)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/redhat-cop/group-sync-operator)
[![CRD Docs](https://img.shields.io/badge/CRD-Docs-brightgreen)](https://doc.crds.dev/github.com/redhat-cop/group-sync-operator)

Synchronizes groups from external providers into OpenShift

## Overview

The OpenShift Container Platform contains functionality to synchronize groups found in external identity providers into the platform. Currently, the functionality that is included in OpenShift is limited to synchronizing LDAP only. This operator is designed to integrate with external providers in order to provide new solutions.

Group Synchronization is facilitated by creating a `GroupSync` resource. The following describes the high level schema for this resource:

```yaml
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: example-groupsync
spec:
  providers:
    - <One or more providers to synchronize>
```

## Deploying the Operator

Use the following steps to deploy the operator to an OpenShift cluster

1. Clone the project locally and changed into the project
.
.

```shell
git clone https://github.com/redhat-cop/group-sync-operator.git
cd group-sync-operator
```

2. Deploy the Operator

```shell
make deploy IMG=quay.io/redhat-cop/group-sync-operator:latest
```

_Note:_ The `make deploy` command will execute the `manifests` target that will require additional build tools to be made available. This target can be skipped by including the `-o manifests` in the command above.

## Authentication

In most cases, authentication details must be provided in order to communicate with providers. Authentication details are provider specific with regards to the required values. In supported providers, the secret can be referenced in the `credentialsSecret` by name and namespace where it has been created as shown below:

```yaml
credentialsSecret:
  name: <secret_name>
  namespace: <secret_namespace>
```

## Providers

Integration with external systems is made possible through a set of pluggable external providers. The following providers are currently supported:

* [Azure](https://azure.microsoft.com/)
* [GitHub](https://github.com)
* [GitLab](https://gitlab.com)
* [LDAP](https://en.wikipedia.org/wiki/Lightweight_Directory_Access_Protocol)
* [Keycloak](https://www.keycloak.org/)/[Red Hat Single Sign On](https://access.redhat.com/products/red-hat-single-sign-on)
* [Okta](https://www.okta.com/)
* [IBM Security Verify](https://docs.verify.ibm.com/verify)

The following sections describe the configuration options available for each provider

### Azure

Groups contained within Azure Active Directory can be synchronized into OpenShift. The following table describes the set of configuration options for the Azure provider:

| Name | Description | Defaults | Required |
| ----- | ---------- | -------- | ----- |
| `authorityHost` | Azure Active Directory Endpoint | `https://login.microsoftonline.com` | No |
| `baseGroups` | List of groups to start searching from instead of listing all groups in the directory | | No |
| `credentialsSecret` | Name of the secret containing authentication details (See below) | | Yes |
| `filter` | Graph API filter | | No |
| `groups` | List of groups to filter against | | No |
| `userNameAttributes` | Fields on a user record to use as the User Name | `userPrincipalName` | No |
| `prune` | Prune Whether to prune groups that are no longer in Azure | `false` | No |

The following is an example of a minimal configuration that can be applied to integrate with a Azure provider:

```yaml
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: azure-groupsync
spec:
  providers:
  - name: azure
    azure:
      credentialsSecret:
        name: azure-group-sync
        namespace: group-sync-operator
```

#### Authenticating to Azure

Authentication to Azure can be performed using Application Registration with access to query group information in Azure Active Directory.

The App Registration must be granted access to the following Microsoft Graph API's:

* Group.Read.All
* GroupMember.Read.All
* User.Read.All

A secret must be created in the same namespace that contains the `GroupSync` resource:

The following keys must be defined in the secret

* `AZURE_TENANT_ID` - Tenant ID
* `AZURE_CLIENT_ID` - Client ID
* `AZURE_CLIENT_SECRET` - Client Secret

The secret can be created by executing the following command:

```shell
oc create secret generic azure-group-sync --from-literal=AZURE_TENANT_ID=<AZURE_TENANT_ID> --from-literal=AZURE_CLIENT_ID=<AZURE_CLIENT_ID> --from-literal=AZURE_CLIENT_SECRET=<AZURE_CLIENT_SECRET>
```

### GitHub

Teams stored within a GitHub organization can be synchronized into OpenShift. The following table describes the set of configuration options for the GitHub provider:

| Name | Description | Defaults | Required |
| ----- | ---------- | -------- | ----- |
| `ca` | Reference to a resource containing a SSL certificate to use for communication (See below) | | No |
| `caSecret` | **DEPRECATED** Reference to a secret containing a SSL certificate to use for communication (See below) | | No |
| `credentialsSecret` | Reference to a secret containing authentication details (See below) | | Yes |
| `insecure` | Ignore SSL verification | `false` | No |
| `organization` | Organization to synchronize against | | Yes |
| `teams` | List of teams to filter against | | No |
| `url` | Base URL for the GitHub or GitHub Enterprise host (Must contain a trailing slash) | | No |
| `prune` | Prune Whether to prune groups that are no longer in GitHub | `false` | No |

The following is an example of a minimal configuration that can be applied to integrate with a GitHub provider:

```yaml
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: github-groupsync
spec:
  providers:
  - name: github
    github:
      organization: ocp
      credentialsSecret:
        name: github-group-sync
        namespace: group-sync-operator
```

#### Authenticating to GitHub

Authentication to GitHub can be performed using an [OAuth Personal Access Token](https://docs.github.com/en/github/authenticating-to-github/keeping-your-account-and-data-secure/creating-a-personal-access-token) or as an [GitHub App](https://docs.github.com/en/developers/apps/getting-started-with-apps/about-apps#about-github-apps), using a secret key and appId.
The OAuth Personal Access Token needs to scope of `admin:org/read:org`.
A secret must be created in the same namespace that contains the `GroupSync` resource:

##### OAuth

When using an OAuth token, the following key is required:

* `token` - OAuth token

The secret can be created by executing the following command:

```shell
oc create secret generic github-group-sync --from-literal=token=<token>
```

##### As a GitHub app

When authenticating as a Github App, the following keys are required:

* `privateKey` and `appId`

###### First create a GitHub app

In GitHub, go to developer-settings -> github apps.

* Create a new app, it does not need webhook callbacks.
* Generate a private-key and download it
* Under "permissions and events", the app will need read-only access to the "Members" permission in the "Organization" section. NOTE: If you enable `mapByScimId`, this permissions needs to be _Read & Write_, though the operator only does read-only operations. The reason for this is the use of the v4 graphql api-endpoint.
* Take note of the "App ID" as you need it for later.
* Install the app to your organization.

###### Create the secret

The secret can be created by executing the following command:

```shell
oc create secret generic github-group-sync --from-literal=appId=<theAppId> --from-file=privateKey=</path/to/thefile>
```

### GitLab

Groups stored within a GitLab can be synchronized into OpenShift. The following table describes the set of configuration options for the GitLab provider:

| Name | Description | Defaults | Required |
| ----- | ---------- | -------- | ----- |
| `ca` | Reference to a resource containing a SSL certificate to use for communication (See below) | | No |
| `caSecret` | **DEPRECATED** Reference to a secret containing a SSL certificate to use for communication (See below) | | No |
| `credentialsSecret` | Reference to a secret containing authentication details (See below) | | Yes |
| `insecure` | Ignore SSL verification | `false` | No |
| `groups` | List of groups to filter against | | No |
| `prune` | Prune Whether to prune groups that are no longer in GitLab | `false` | No |
| `scope` | Scope for group synchronization. Options are `one` for one level or `sub` to include subgroups | `sub` | No |
| `url` | Base URL for the GitLab instance | `https://gitlab.com` | No |

The following is an example of a minimal configuration that can be applied to integrate with a GitHub provider:

```yaml
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: gitlab-groupsync
spec:
  providers:
  - name: gitlab
    gitlab:
      credentialsSecret:
        name: gitlab-group-sync
        namespace: group-sync-operator
```

#### Authenticating to GitLab

Authentication to GitLab can be performed using a [Token](https://docs.gitlab.com/ee/security/token_overview.html) or a Username and Password (Note: 2FA not supported). A secret must be created in the same namespace that contains the `GroupSync` resource:

When using an OAuth token, the following token types are supported:

* Personal Access Token
* OAuth Token
* Job Token

 the following key is required:

* `token` - OAuth token

Optionally, the `tokenType` key can be specified to indicate the type of token being provided from the following values:

* OAuth - `oauth`
* Personal Access Token - `personal`
* Job Token - `job`

If no `tokenType` is provided, `oauth` is used by default

The secret can be created by executing the following command:

```shell
oc create secret generic gitlab-group-sync --from-literal=token=<token>
```

To specify a token type, such as a Personal Access Token, the following command can be executed:

```shell
oc create secret generic gitlab-group-sync --from-literal=token=<token> --from-literal=tokenType=personal
``` 

The following keys are required for username and password:

* `username` - Username for authenticating with GitLab
* `password` - Password for authenticating with GitLab

The secret can be created by executing the following command:

```shell
oc create secret generic gitlab-group-sync --from-literal=username=<username> --from-literal=password=<password>
```

### LDAP

Groups stored within an [LDAP](https://en.wikipedia.org/wiki/Lightweight_Directory_Access_Protocol) server can be synchronized into OpenShift. The LDAP provider implements the included features of the [Syncing LDAP groups](https://docs.openshift.com/container-platform/latest/authentication/ldap-syncing.html) feature and makes use of the libraries from the [OpenShift Command Line](https://docs.openshift.com/container-platform/latest/cli_reference/openshift_cli/getting-started-cli.html) tool to streamline the migration to this operator based implementation.

The configurations of the three primary schemas (`rfc2307`, `activeDirectory` and `augmentedActiveDirectory`) can be directly migrated as is without any modification.

| Name | Description | Defaults | Required |
| ----- | ---------- | -------- | ----- |
| `ca` | Reference to a resource containing a SSL certificate to use for communication (See below) | | No |
| `caSecret` | **DEPRECATED** Reference to a secret containing a SSL certificate to use for communication (See below) | | No |
| `credentialsSecret` | Reference to a secret containing authentication details (See below) | | No |
| `insecure` | Ignore SSL verification | `false` | No |
| `groupUIDNameMapping` | User defined name mapping | | No |
| `rfc2307` | Configuration using the [rfc2307](https://docs.openshift.com/container-platform/latest/authentication/ldap-syncing.html#ldap-syncing-rfc2307_ldap-syncing-groups) schema | | No |
| `activeDirectory` | Configuration using the [activeDirectory](https://docs.openshift.com/container-platform/4.5/authentication/ldap-syncing.html#ldap-syncing-activedir_ldap-syncing-groups) schema | | No |
| `augmentedActiveDirectory` | Configuration using the [activeDirectory](https://docs.openshift.com/container-platform/4.5/authentication/ldap-syncing.html#ldap-syncing-augmented-activedir_ldap-syncing-groups) schema | | No |
| `url` | Connection URL for the LDAP server | `ldap://ldapserver:389` | No |
| `whitelist` | Explicit list of groups to synchronize |  | No |
| `blacklist` | Explicit list of groups to not synchronize |  | No |
| `prune` | Prune Whether to prune groups that are no longer in LDAP | `false` | No |

The following is an example using the `rfc2307` schema:

```yaml
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: ldap-groupsync
spec:
  providers:
  - ldap:
      credentialsSecret:
        name: ldap-group-sync
        namespace: group-sync-operator
      insecure: true
      rfc2307:
        groupMembershipAttributes:
        - member
        groupNameAttributes:
        - cn
        groupUIDAttribute: dn
        groupsQuery:
          baseDN: ou=Groups,dc=example,dc=com
          derefAliases: never
          filter: (objectClass=groupofnames)
          scope: sub
        tolerateMemberNotFoundErrors: true
        tolerateMemberOutOfScopeErrors: true
        userNameAttributes:
        - cn
        userUIDAttribute: dn
        usersQuery:
          baseDN: ou=Users,dc=example,dc=com
          derefAliases: never
          scope: sub
      url: ldap://ldapserver:389
    name: ldap
```

The examples provided in the OpenShift documented referenced previously can be used to construct the schemas for the other LDAP synchronization types.

#### Authenticating to LDAP

If authentication is required in order to communicate with the LDAP server, a secret should be created in the same namespace that contains the `GroupSync` resource. The following keys can be defined:

* `username` - Username (Bind DN) for authenticating with the LDAP server
* `password` - Password for authenticating with the LDAP server

The secret can be created by executing the following command:

```shell
oc create secret generic ldap-group-sync --from-literal=username=<username> --from-literal=password=<password>
```

#### Whitelists and Blacklists

Groups can be explicitly whitelisted or blacklisted in order to control the groups that are eligible to be synchronized into OpenShift. When running LDAP group synchronization using the command line, this configuration is referenced via separate files, but these are instead specified in the `blacklist` and `whitelist` properties as shown below:

```yaml
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: ldap-groupsync
spec:
  providers:
  - ldap:
...
      whitelist:
      - cn=Online Corporate Banking,ou=Groups,dc=example,dc=com
...
    name: ldap
```

```yaml
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: ldap-groupsync
spec:
  providers:
  - ldap:
...
      blacklist:
      - cn=Finance,ou=Groups,dc=example,dc=com
...
    name: ldap
```

### Keycloak

Groups stored within Keycloak can be synchronized into OpenShift. The following table describes the set of configuration options for the Keycloak provider:

| Name | Description | Defaults | Required |
| ----- | ---------- | -------- | ----- |
| `ca` | Reference to a resource containing a SSL certificate to use for communication (See below) | | No |
| `caSecret` | **DEPRECATED** Reference to a secret containing a SSL certificate to use for communication (See below) | | No |
| `credentialsSecret` | Reference to a secret containing authentication details (See below) | | Yes |
| `groups` | List of groups to filter against | | No |
| `insecure` | Ignore SSL verification | `false` | No |
| `loginRealm` | Realm to authenticate against | `master` | No |
| `realm` | Realm to synchronize | | Yes |
| `scope` | Scope for group synchronization. Options are `one` for one level or `sub` to include subgroups | `sub` | No |
| `url` | Base URL for the Keycloak server. Older versions (<17.0.0) including Red Hat SSO should include the context path `/auth` appended to the hostname  | | Yes |
| `prune` | Prune Whether to prune groups that are no longer in Keycloak | `false` | No |

The following is an example of a minimal configuration that can be applied to integrate with a Keycloak provider:

```yaml
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: keycloak-groupsync
spec:
  providers:
  - name: keycloak
    keycloak:
      realm: ocp
      credentialsSecret:
        name: keycloak-group-sync
        namespace: group-sync-operator
      url: https://keycloak-keycloak-operator.apps.openshift.com/auth
```

#### Authenticating to Keycloak

A user with rights to query for Keycloak groups must be available. The following permissions must be associated to the user:

* Password must be set (Temporary option unselected) on the _Credentials_ tab
* On the _Role Mappings_ tab, select _master-realm_ or _realm-management_ next to the _Client Roles_ dropdown and then select **query-groups**, **query-users**, and **view-users**.

A secret must be created in the same namespace that contains the `GroupSync` resource. It must contain the following keys for the user previously created:

* `username` - Username for authenticating with Keycloak
* `password` - Password for authenticating with Keycloak

The secret can be created by executing the following command:

```shell
oc create secret generic keycloak-group-sync --from-literal=username=<username> --from-literal=password=<password>
```

### Okta

[Okta Groups](https://help.okta.com/en/prod/Content/Topics/users-groups-profiles/usgp-main.htm) assigned to [Okta Applications](https://help.okta.com/en/prod/Content/Topics/Apps/Apps_Apps.htm) can be synchronized into OpenShift. The developer docs for the Okta API that the Okta Syncer uses can be found [here](https://developer.okta.com/docs/reference/api/apps/#list-groups-assigned-to-application).
The following table describes the set of configuration options for the Okta provider:

| Name | Description | Defaults | Required |
| ----- | ---------- | -------- | ----- |
| `credentialsSecret` | Reference to a secret containing authentication details (See below) | `''`  | Yes |
| `groups` | List of groups to filter against | `nil`  | No |
| `url` | Okta URL which can be found under the "Okta Domain" in your application settings (must contain the scheme and a trailing slash) | `''`  | Yes |
| `appId` | Okta Application (Client) ID that is attached to the application groups you wish to sync | `''`  | Yes |
| `extractLoginUsername` | Bool to determine if you should extract username from okta login | `false`  | No |
| `profileKey` | Attribute field on Okta User Profile you would like to use as identity | `'login'` | No |
| `groupLimit` | Integer to set the maximum number of groups to retrieve from OKTA per request. | `1000` | No |
| `prune` | Prune Whether to prune groups that are no longer in OKTA | `false` | No |

The following is an example of a minimal configuration that can be applied to integrate with an Okta provider:

```yaml
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: okta-sync
spec:
  providers:
    - name: okta
      okta:
        credentialsSecret:
          name: okta-api-token
          namespace: group-sync-operator
        url: "https://example.okta.com/"
        appId: xxxxxxxxxxxxxxxxxxxx
```

#### Authenticating to Okta

A secret must be created in the same namespace as the group-sync-operator pod. It must contain the following key:

* `okta-api-token` - Okta API Token for interacting with Okta

The secret can be created by executing the following command:

```shell
oc create secret generic okta-api-token --from-literal=okta-api-token=<OKTA_API_TOKEN> -n group-sync-operator
```

### IBM Security Verify

Groups defined in [IBM Security Verify](https://help.okta.com/en/prod/Content/Topics/users-groups-profiles/usgp-main.htm) (ISV) can be synchronized into OpenShift. Currently only the `userName` field from ISV will be synchronized. The developer docs for the ISV API can be found [here](https://docs.verify.ibm.com/verify/page/api-documentation).
The following table describes the set of configuration options for the provider:

| Name | Description | Defaults | Required |
| ----- | ---------- | -------- | ----- |
| `credentialsSecret` | Reference to a secret containing authentication details (see below) | `''`  | Yes |
| `groups` | List of groups to synchronize (see below) | `nil`  | Yes |
| `tenantUrl` | The ISV tenant URL, for example `https://my-isv.verify.ibm.com`) | `''`  | Yes |

The following is an example of a minimal configuration that can be applied to integrate with an Okta provider:

```yaml
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: ibmsecurityverify-sync
spec:
    providers:
    - name: ibmsecurityverify
      ibmsecurityverify:
        credentialsSecret:
          name: isv-group-sync
          namespace: group-sync-operator
        tenantUrl: https://my-isv.verify.ibm.com
        groups:
          - name: 'application owners'
            id: 645001V3V9
          - name: developer
            id: 645001V3VA
```

#### Group Objects
Each group object in the `groups` array must contain an `id` field. The group ID can be retrieved by pulling the group information from the ISV API. Optionally, the object may also contain a `name` which corresponds to the group's display name. When defined, the operator will confirm that the name defined in the YAML matches that received from the API when synchronization occurs; as the group IDs are not human-friendly, using the name can confirm the correct groups are configured. If the names do not match an error will be logged.

#### Group Names
The name of each groups created in OpenShift will match the group name in ISV. Any whitespace in the ISV group name will be replaced with a hyphen.

#### Authenticating to IBM Security Verify

A secret must be created in the same namespace as the group-sync-operator pod. It must contain the following keys:

* `clientId` - The API client ID.
* `clientSecret`- The API client secret.

See the IBM Security Verify [API documentation](https://docs.verify.ibm.com/verify/docs/api-access) for setting up authentication.

### Support for Additional Metadata (Beta)

Additional metadata based on Keycloak group are also added to the OpenShift groups as Annotations including:

* Parent/child relationship between groups and their subgroups
* Group attributes

## CA Certificates

Several providers allow for certificates to be provided in either a _ConfigMap_ or _Secret_ to communicate securely to the target host through the use of a property called `ca`.

The certificate can be added to a Secret called _keycloak-certs_ using the key `ca.crt` representing the certificate using the following command.

```
oc create secret generic keycloak-certs --from-file=ca.crt=<file>
```

An example of how the CA certificate can be added to the Keycloak provider is shown below:

```yaml
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: keycloak-groupsync
spec:
  providers:
  - name: keycloak
    keycloak:
      realm: ocp
      credentialsSecret:
        name: keycloak-group-sync
        namespace: group-sync-operator
      ca:
        kind: Secret
        name: keycloak-certs
        namespace: group-sync-operator
        key: ca.crt
      url: https://keycloak-keycloak-operator.apps.openshift.com
```

Alteratively, a _ConfigMap_ can be used instead instead of a _Secret_. This is useful when using the [Certificate injection using Operators](https://docs.openshift.com/container-platform/latest/networking/configuring-a-custom-pki.html#certificate-injection-using-operators_configuring-a-custom-pki) feature.

The following command can be used to create a _ConfigMap_ containing the certificate:

```shell
oc create configmap keycloak-certs --from-file=ca.crt=<file>
```

An example of how the CA certificate can be added to the Keycloak provider is shown below:

```yaml
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: keycloak-groupsync
spec:
  providers:
  - name: keycloak
    keycloak:
      realm: ocp
      credentialsSecret:
        name: keycloak-group-sync
        namespace: group-sync-operator
      ca:
        kind: ConfigMap
        name: keycloak-certs
        namespace: group-sync-operator
        key: ca.crt
      url: https://keycloak-keycloak-operator.apps.openshift.com
```

## Scheduled Execution

A cron style expression can be specified for which a synchronization event will occur. The following specifies that a synchronization should occur nightly at 3AM

```yaml
apiVersion: redhatcop.redhat.io/v1alpha1
kind: GroupSync
metadata:
  name: keycloak-groupsync
spec:
  schedule: "0 3 * * *"
  providers:
  - ...
```

If a schedule is not provided, synchronization will occur only when the object is reconciled by the platform.

## Accessing Secrets and ConfigMaps in Other Namespaces

By default, the operator monitors resources in the namespace that it has been deployed within. This is defined by setting the `WATCH_NAMESPACE` environment variable. Support is available for accessing ConfigMaps and Secrets in other namespaces so that existing resources may be utilized as desired.

To enable the operator to access resources across multiple, set the environment variable with a comma separate list of namespaces that include the namespace the operator is deployed within and any additional namespaces that are desired.

To make use of this feature when deploying through the Operator Lifecycle Manager, set the following configuration on the `Subscription` resource:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: group-sync-operator
  namespace: group-sync-operator
spec:
  channel: alpha
  installPlanApproval: Automatic
  name: group-sync-operator
  source: community-operators
  sourceNamespace: openshift-marketplace
  config:
    env:
      - name: WATCH_NAMESPACE
        value: "<comma separated list of namespaces>"
```

## Deploying the Operator

This is a namespace level operator that you can deploy in any namespace. However, `group-sync-operator` is recommended.

It is recommended to deploy this operator via [`OperatorHub`](https://operatorhub.io/), but you can also deploy it using [`Helm`](https://helm.sh/).

### Deploying from OperatorHub

If you want to utilize the Operator Lifecycle Manager (OLM) to install this operator, you can do so in two ways: from the UI or the CLI.

### Multiarch Support

| Arch  | Support  |
|:-:|:-:|
| amd64  | ✅ |
| arm64  | ✅  |
| ppc64le  | ✅  |
| s390x  | ✅  |

#### Deploying from OperatorHub UI

* If you would like to launch this operator from the UI, you'll need to navigate to the OperatorHub tab in the console.
* Search for this operator by name: `group sync operator`. This will then return an item for our operator and you can select it to get started. Once you've arrived here, you'll be presented with an option to install, which will begin the process.
* After clicking the install button, you are presented with the namespace that the operator will be installed in. A suggested name of `group-sync-operator` is presented and can be created automatically at installation time.
* Select the installation strategy you would like to proceed with (`Automatic` or `Manual`).
* Once you've made your selection, you can select `Subscribe` and the installation will begin. After a few moments you can go ahead and check your namespace and you should see the operator running.

#### Deploying from OperatorHub using CLI

If you'd like to launch this operator from the command line, you can use the manifests contained in this repository by running the following:

```shell
oc new-project group-sync-operator
oc apply -f config/operatorhub -n group-sync-operator
```

This will create the appropriate OperatorGroup and Subscription and will trigger OLM to launch the operator in the specified namespace.

### Deploying with Helm

Here are the instructions to install the latest release with Helm.

```shell
oc new-project group-sync-operator
helm repo add group-sync-operator https://redhat-cop.github.io/group-sync-operator
helm repo update
helm install group-sync-operator group-sync-operator/group-sync-operator
```

This can later be updated with the following commands:

```shell
helm repo update
helm upgrade group-sync-operator group-sync-operator/group-sync-operator
```

## Metrics

Prometheus compatible metrics are exposed by the Operator and can be integrated into OpenShift's default cluster monitoring. To enable OpenShift cluster monitoring, label the namespace the operator is deployed in with the label `openshift.io/cluster-monitoring="true"`.

```shell
oc label namespace <namespace> openshift.io/cluster-monitoring="true"
```

### Test metrics

```sh
export operatorNamespace=group-sync-operator-local # or group-sync-operator
oc label namespace ${operatorNamespace} openshift.io/cluster-monitoring="true"
oc rsh -n openshift-monitoring -c prometheus prometheus-k8s-0 /bin/bash
export operatorNamespace=group-sync-operator-local # or group-sync-operator
curl -v -s -k -H "Authorization: Bearer $(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" https://group-sync-operator-controller-manager-metrics-service.${operatorNamespace}.svc.cluster.local:8443/metrics
exit
```

## Development

### Running the operator locally

```shell
make install
export repo=redhatcopuser #replace with yours
docker login quay.io/$repo/group-sync-operator
make docker-build IMG=quay.io/$repo/group-sync-operator:latest
make docker-push IMG=quay.io/$repo/group-sync-operator:latest
oc new-project group-sync-operator-local
kustomize build ./config/local-development | oc apply -f - -n group-sync-operator-local
export token=$(oc serviceaccounts get-token 'group-sync-operator-controller-manager' -n group-sync-operator-local)
oc login --token ${token}
make run ENABLE_WEBHOOKS=false
```

### Test helm chart locally

Define an image and tag. For example...

```shell
export imageRepository="quay.io/redhat-cop/group-sync-operator"
export imageTag="$(git -c 'versionsort.suffix=-' ls-remote --exit-code --refs --sort='version:refname' --tags https://github.com/redhat-cop/group-sync-operator.git '*.*.*' | tail --lines=1 | cut --delimiter='/' --fields=3)"
```

Deploy chart...

```shell
make helmchart IMG=${imageRepository} VERSION=${imageTag}
helm upgrade -i group-sync-operator-local charts/group-sync-operator -n group-sync-operator-local --create-namespace
```

Delete...

```shell
helm delete group-sync-operator-local -n group-sync-operator-local
kubectl delete -f charts/group-sync-operator/crds/crds.yaml
```

### Building/Pushing the operator image

```shell
export repo=redhatcopuser #replace with yours
docker login quay.io/$repo/group-sync-operator
make docker-build IMG=quay.io/$repo/group-sync-operator:latest
make docker-push IMG=quay.io/$repo/group-sync-operator:latest
```

### Deploy to OLM via bundle

```shell
make manifests
make bundle IMG=quay.io/$repo/group-sync-operator:latest
operator-sdk bundle validate ./bundle --select-optional name=operatorhub
make bundle-build BUNDLE_IMG=quay.io/$repo/group-sync-operator-bundle:latest
docker login quay.io/$repo/group-sync-operator-bundle
docker push quay.io/$repo/group-sync-operator-bundle:latest
operator-sdk bundle validate quay.io/$repo/group-sync-operator-bundle:latest --select-optional name=operatorhub
oc new-project group-sync-operator
oc label namespace group-sync-operator openshift.io/cluster-monitoring="true"
operator-sdk cleanup group-sync-operator -n group-sync-operator
operator-sdk run bundle -n group-sync-operator quay.io/$repo/group-sync-operator-bundle:latest
```

### Releasing

```shell
git tag -a "<tagname>" -m "<commit message>"
git push upstream <tagname>
```

If you need to remove a release:

```shell
git tag -d <tagname>
git push upstream --delete <tagname>
```

If you need to "move" a release to the current master

```shell
git tag -f <tagname>
git push upstream -f <tagname>
```

### Cleaning up

```shell
operator-sdk cleanup group-sync-operator -n group-sync-operator
oc delete operatorgroup operator-sdk-og
oc delete catalogsource group-sync-operator-catalog
```
