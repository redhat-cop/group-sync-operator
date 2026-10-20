# Default values for group-sync-operator.
# This is a YAML-formatted file.
# Declare variables to be passed into your templates.

# Global parameters
global:
  # global.imageRegistry -- Global Docker image registry (overrides image.registry)
  imageRegistry: ""

replicaCount: 1

image:
  # image.registry -- Image registry prefix (overridden by global.imageRegistry if set).
  # Leave empty to use image.repository as the full image path.
  registry: ""
  # image.repository -- Image repository
  repository: ${image_repo}
  pullPolicy: IfNotPresent
  # Overrides the image tag whose default is the chart appVersion.
  tag: ${version}
  # DEPRECATED: use image.tag instead. Kept for backwards compatibility.
  version: ${version}

imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""
env: []
args: []
podAnnotations: {}

podLabels: {}

commonLabels: {}

podSecurityContext:
  runAsNonRoot: true

securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - "ALL"

resources:
  requests:
    cpu: 300m
    memory: 200Mi

nodeSelector: {}

tolerations: []

affinity: {}

volumes: []

volumeMounts: []

# Set to true to create the metrics Service and ServiceMonitor resources (the ServiceMonitor requires Prometheus Operator CRDs).
enableMonitoring: true
# Set to false to expose the metrics endpoint over plain HTTP.
metricsSecure: true

# Set to true to create a cert-manager Certificate for the metrics service (requires cert-manager).
# Recommended on Kubernetes clusters where OpenShift serving certificates are not available.
# When false, the metrics Service is annotated so OpenShift creates the serving certificate automatically.
enableCertManager: false

certificate:
  issuerRef:
    name: selfsigned-issuer
    kind: Issuer
    # group: cert-manager.io
