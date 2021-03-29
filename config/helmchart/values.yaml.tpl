# Default values for helm-try.
# This is a YAML-formatted file.
# Declare variables to be passed into your templates.

replicaCount: 1

image:
  repository: ${image_repo}
  pullPolicy: IfNotPresent
  # Overrides the image tag whose default is the chart appVersion.
  tag: ${version}

imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""

podAnnotations: {}

resources:
  requests:
    cpu: 100m
    memory: 20Mi

nodeSelector: {}

tolerations: []

affinity: {}

env: 
  # - name: VAR_NAME
  #   value: var-value
