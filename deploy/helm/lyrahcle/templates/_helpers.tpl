{{- define "lyrahcle.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "lyrahcle.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name (include "lyrahcle.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{- define "lyrahcle.labels" -}}
app.kubernetes.io/name: {{ include "lyrahcle.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end }}

{{- define "lyrahcle.selectorLabels" -}}
app.kubernetes.io/name: {{ include "lyrahcle.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "lyrahcle.backendServiceAccountName" -}}
{{- if .Values.serviceAccounts.backend.create }}
{{- default (printf "%s-backend" (include "lyrahcle.fullname" .)) .Values.serviceAccounts.backend.name }}
{{- else }}
{{- required "serviceAccounts.backend.name is required when create=false" .Values.serviceAccounts.backend.name }}
{{- end }}
{{- end }}

{{- define "lyrahcle.frontendServiceAccountName" -}}
{{- if .Values.serviceAccounts.frontend.create }}
{{- default (printf "%s-frontend" (include "lyrahcle.fullname" .)) .Values.serviceAccounts.frontend.name }}
{{- else }}
{{- required "serviceAccounts.frontend.name is required when create=false" .Values.serviceAccounts.frontend.name }}
{{- end }}
{{- end }}

{{- define "lyrahcle.cveSyncServiceAccountName" -}}
{{- if .Values.serviceAccounts.cveSync.create }}
{{- default (printf "%s-cve-sync" (include "lyrahcle.fullname" .)) .Values.serviceAccounts.cveSync.name }}
{{- else }}
{{- required "serviceAccounts.cveSync.name is required when create=false" .Values.serviceAccounts.cveSync.name }}
{{- end }}
{{- end }}

{{- define "lyrahcle.image" -}}
{{- if .image.digest -}}
{{ printf "%s@%s" .image.repository .image.digest }}
{{- else -}}
{{ printf "%s:%s" .image.repository .image.tag }}
{{- end -}}
{{- end }}
