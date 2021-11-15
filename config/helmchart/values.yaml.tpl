# Default values for helm-try.
# This is a YAML-formatted file.
# Declare variables to be passed into your templates.

replicaCount: 1

image:
  repository: ${image_repo}
  pullPolicy: IfNotPresent
  # Overrides the image tag whose default is the chart appVersion.
  version: ${version}

imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""
env: []
podAnnotations: {}

resources:
  requests:
    cpu: 300m
    memory: 200Mi

nodeSelector: {}

tolerations: []

affinity: {}

env: 
  # - name: VAR_NAME
  #   value: var-value

kube_rbac_proxy:
  image:
    repository: quay.io/redhat-cop/kube-rbac-proxy
    pullPolicy: IfNotPresent
    version: v0.11.0
  resources:
    requests:
      cpu: 100m
      memory: 20Mi

enableMonitoring: true      
