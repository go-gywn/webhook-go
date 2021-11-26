[{{ .status }}] {{ .summary }}
** Instance: {{ .instance }}
** Level: {{ .level }}{{ if eq .status "firing" }}
** Start: {{ .startsAt.Format "01/02 15:04:05 MST" }}{{ else }}
** Start: {{ .endsAt.Format "01/02 15:04:05 MST" }}
** End: {{ .endsAt.Format "01/02 15:04:05 MST" }}{{ end }}
** Desc: {{ .description }}
