package handler

import (
	"html/template"

	"github.com/gin-gonic/gin"
	"github.com/go-gywn/goutil"
	"github.com/go-gywn/webhook-go/common"
	t "github.com/prometheus/alertmanager/template"
)

var routerGroup *gin.RouterGroup
var logger = goutil.GetLogger("handler")
var crypt = goutil.GetCrypto(common.CONF.Key)
var fileUtil = goutil.GetFileUtil()

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

var labelAlertname = common.CONF.Webhook.LabelMapper["alertname"]
var labelInstance = common.CONF.Webhook.LabelMapper["instance"]
var labelLevel = common.CONF.Webhook.LabelMapper["level"]
var labelJob = common.CONF.Webhook.LabelMapper["job"]
var labelSummary = common.CONF.Webhook.AnnotationMapper["summary"]
var labelDescription = common.CONF.Webhook.AnnotationMapper["description"]

func init() {
	gin.SetMode(goutil.GetLogMode())
}

// StartHandler start API server
func StartHandler() error {
	router := gin.Default()

	// Start webhook API
	startHook(router.Group(common.CONF.Base))

	// startHookThread(make(chan t.Alert, 100))
	return router.Run(common.CONF.Port)
}
