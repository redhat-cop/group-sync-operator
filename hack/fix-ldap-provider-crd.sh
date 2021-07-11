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
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required[] | select(. == "derefAliases"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required[] | select(. == "filter"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required[] | select(. == "scope"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required[] | select(. == "timeout"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.activeDirectory.properties.usersQuery.required[] | select(. == "pageSize"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml

${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required[] | select(. == "derefAliases"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required[] | select(. == "filter"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required[] | select(. == "scope"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required[] | select(. == "timeout"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.groupsQuery.required[] | select(. == "pageSize"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required[] | select(. == "derefAliases"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required[] | select(. == "filter"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required[] | select(. == "scope"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required[] | select(. == "timeout"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.augmentedActiveDirectory.properties.usersQuery.required[] | select(. == "pageSize"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml

${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required[] | select(. == "derefAliases"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required[] | select(. == "filter"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required[] | select(. == "scope"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required[] | select(. == "timeout"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.groupsQuery.required[] | select(. == "pageSize"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required[] | select(. == "derefAliases"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required[] | select(. == "filter"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required[] | select(. == "scope"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required[] | select(. == "timeout"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.properties.usersQuery.required[] | select(. == "pageSize"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.required[] | select(. == "tolerateMemberNotFoundErrors"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml
${YQ} e -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.providers.items.properties.ldap.properties.rfc2307.required[] | select(. == "tolerateMemberOutOfScopeErrors"))' ${DIR}/../config/crd/bases/redhatcop.redhat.io_groupsyncs.yaml

