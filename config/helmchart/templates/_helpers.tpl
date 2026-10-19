{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "group-sync-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "group-sync-operator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "group-sync-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "group-sync-operator.labels" -}}
helm.sh/chart: {{ include "group-sync-operator.chart" . }}
{{ include "group-sync-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if .Values.commonLabels }}
{{ toYaml .Values.commonLabels }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "group-sync-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "group-sync-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}


{{/*
Service Acount Name
*/}}
{{- define "group-sync-operator.serviceAccountName" -}}
{{- printf "%s-%s" (include "group-sync-operator.name" .) "controller-manager" }}
{{- end }}

{{/*
Return the proper image name.
Usage:
  {{ include "group-sync-operator.image" ( dict "imageRoot" .Values.image "global" .Values.global "Chart" .Chart "context" $ ) }}
*/}}
{{- define "group-sync-operator.image" -}}
{{- $registryName := "" -}}
{{- $repositoryName := .imageRoot.repository | default "" -}}
{{- $tag := .imageRoot.tag | default .imageRoot.version | default .Chart.AppVersion | toString -}}
{{- if .imageRoot.registry }}
    {{- $registryName = tpl .imageRoot.registry .context -}}
{{- end -}}
{{- if .global }}
    {{- if .global.imageRegistry }}
        {{- $registryName = tpl .global.imageRegistry .context -}}
    {{- end -}}
{{- end -}}
{{- if eq (substr 0 7 $tag) "sha256:" -}}
{{- if $registryName }}
    {{- printf "%s/%s@%s" $registryName $repositoryName $tag -}}
{{- else }}
    {{- printf "%s@%s" $repositoryName $tag -}}
{{- end -}}
{{- else }}
{{- if $registryName }}
    {{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else }}
    {{- printf "%s:%s" $repositoryName $tag -}}
{{- end -}}
{{- end -}}
{{- end -}}
