package handler

import (
	"html/template"

	"github.com/gin-gonic/gin"
	"github.com/go-gywn/webhook-go/common"
	t "github.com/prometheus/alertmanager/template"
)

var routerGroup *gin.RouterGroup
var logger = common.NewLogger("handler")

var chanHook = make(chan t.Alert, 100)
var hookTemplate *template.Template
var hookDefaultTemplate *template.Template
var defaultTemplate = `[{{ .status }}] {{ .summary }}
> Instance: {{ .instance }}
> Level: {{ .level }}{{ if eq .status "firing" }}
> Start: {{ .startsAt.Format "01/02 15:04:05 MST" }}{{ else }}
> Start: {{ .endsAt.Format "01/02 15:04:05 MST" }}
> End: {{ .endsAt.Format "01/02 15:04:05 MST" }}{{ end }}
> Description: {{ .description }}`

var labelAlertname = common.Cfg.Webhook.LabelMapper["alertname"]
var labelInstance = common.Cfg.Webhook.LabelMapper["instance"]
var labelLevel = common.Cfg.Webhook.LabelMapper["level"]
var labelJob = common.Cfg.Webhook.LabelMapper["job"]
var labelSummary = common.Cfg.Webhook.AnnotationMapper["summary"]
var labelDescription = common.Cfg.Webhook.AnnotationMapper["description"]

// StartHandler start API server
func StartHandler() error {
	router := gin.Default()

	// Start webhook API
	startHook(router.Group(common.Cfg.Base))

	// startHookThread(make(chan t.Alert, 100))
	return router.Run(common.Cfg.Port)
}
