{{- define "eratemanager.name" -}}
eratemanager
{{- end }}

{{- define "eratemanager.fullname" -}}
eratemanager
{{- end }}

{{- define "eratemanager.db.dsn" -}}
{{- if .Values.db.dsn -}}
{{- .Values.db.dsn -}}
{{- else if .Values.postgresql.enabled -}}
postgres://{{ .Values.postgresql.auth.username }}:{{ .Values.postgresql.auth.password }}@{{ include "eratemanager.fullname" . }}-postgresql:5432/{{ .Values.postgresql.auth.database }}?sslmode=disable
{{- end -}}
{{- end -}}
