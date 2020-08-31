#!/bin/bash

trap popd >> /dev/null 2>&1 

OPERATOR_SDK="${OPERATOR_SDK:-operator-sdk}"
YQ="${YQ:-yq}"
OPENAPI_GEN="${OPENAPI_GEN:-openapi-gen}"

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

pushd "${DIR}/../" >> /dev/null 2>&1 


set +e

if ! command -v ${OPERATOR_SDK} > /dev/null 2>&1; then
  echo operator-sdk is not available
  exit 1
fi

if ! command -v ${YQ} > /dev/null 2>&1; then
  echo yq is not available
  exit 1
fi

if ! command -v ${OPENAPI_GEN} > /dev/null 2>&1; then
  echo openapi-gen is not available
  exit 1
fi

set -e

${OPERATOR_SDK} generate crds
${OPERATOR_SDK} generate k8s
${OPENAPI_GEN} --logtostderr=true -o "" -i ./pkg/apis/redhatcop/v1alpha1/ -O zz_generated.openapi -p ./pkg/apis/redhatcop/v1alpha1/ -h ./hack/boilerplate.go.txt -r "-"

# Remove Required Properties from LDAP
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required(.==derefAliases)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required(.==filter)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required(.==scope)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required(.==timeout)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required(.==pageSize)'

yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required(.==derefAliases)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required(.==filter)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required(.==scope)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required(.==timeout)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required(.==pageSize)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required(.==derefAliases)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required(.==filter)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required(.==scope)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required(.==timeout)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required(.==pageSize)'

yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required(.==derefAliases)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required(.==filter)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required(.==scope)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required(.==timeout)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required(.==pageSize)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required(.==derefAliases)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required(.==filter)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required(.==scope)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required(.==timeout)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required(.==pageSize)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.required(.==tolerateMemberNotFoundErrors)'
yq d -i ${DIR}/../deploy/crds/redhatcop.redhat.io_groupsyncs_crd.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.required(.==tolerateMemberOutOfScopeErrors)'


