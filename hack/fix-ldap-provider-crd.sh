#!/bin/bash

trap popd >> /dev/null 2>&1 

YQ="${YQ:-yq}"

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

pushd "${DIR}/../" >> /dev/null 2>&1 

set +e

if ! command -v ${YQ} > /dev/null 2>&1; then
  echo yq is not available
  exit 1
fi

set -e

# Remove Required Properties from LDAP
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required(.==derefAliases)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required(.==filter)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required(.==scope)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required(.==timeout)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required(.==pageSize)'

${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required(.==derefAliases)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required(.==filter)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required(.==scope)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required(.==timeout)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required(.==pageSize)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required(.==derefAliases)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required(.==filter)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required(.==scope)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required(.==timeout)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required(.==pageSize)'

${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required(.==derefAliases)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required(.==filter)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required(.==scope)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required(.==timeout)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required(.==pageSize)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required(.==derefAliases)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required(.==filter)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required(.==scope)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required(.==timeout)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required(.==pageSize)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.required(.==tolerateMemberNotFoundErrors)'
${YQ} d -i ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml 'spec.validation.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.required(.==tolerateMemberOutOfScopeErrors)'

