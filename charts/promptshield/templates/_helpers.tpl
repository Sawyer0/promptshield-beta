{{/*
Common labels*/}}
{{- define "promptshield.labels" -}}
helm.sh/chart: {{ include "promptshield.chart" . }}
app.kubernetes.io/name: {{ include "promptshield.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "promptshield.selectorLabels" -}}
app.kubernetes.io/name: {{ include "promptshield.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "promptshield.name" -}}
{{ .Chart.Name }}
{{- end -}}

{{- define "promptshield.chart" -}}
{{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}

{{- define "promptshield.fullname" -}}
{{ .Release.Name }}-{{ .Chart.Name }}
{{- end -}}

{{- define "promptshield.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{ tpl (default (include "promptshield.fullname" .) .Values.serviceAccount.name) . }}
{{- else }}
{{ tpl .Values.serviceAccount.name . }}
{{- end -}}
{{- end -}}
