{{/*
Expand the name of the chart.
*/}}
{{- define "nginx-fileserver.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Full name: release + chart name, truncated to 63 chars.
*/}}
{{- define "nginx-fileserver.fullname" -}}
{{- printf "%s-%s" .Release.Name (include "nginx-fileserver.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "nginx-fileserver.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
app.kubernetes.io/name: {{ include "nginx-fileserver.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "nginx-fileserver.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nginx-fileserver.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
