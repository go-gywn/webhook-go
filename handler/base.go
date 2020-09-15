package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-gywn/webhook-go/common"
	"github.com/go-gywn/webhook-go/model"
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

// ErrorIf return boolean if error
func ErrorIf(c *gin.Context, err error) bool {
	if err != nil {
		c.JSON(http.StatusExpectationFailed, gin.H{
			"status": "fail",
			"result": err.Error(),
		})
		c.Abort()
		return true
	}
	return false
}

// Success normal message if success
func Success(c *gin.Context, result interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"result": result,
	})
	c.Abort()
}
func hookSender(chanHook chan t.Alert) {
	var targets = common.Cfg.Webhook.Targets
	hookDefaultTemplate, _ = template.New("default_template").Parse(defaultTemplate)

	go func() {
		for {

			alert := <-chanHook
			target := targets[alert.Labels[labelLevel]]
			api := target.API
			apiParams := target.Params

			// ============================================
			// Generate fingerprint if fingerprint is empty
			// ============================================
			k := fmt.Sprintf("%d", alert.StartsAt.Unix())
			k += alert.Labels[labelAlertname]
			k += alert.Labels[labelInstance]
			k += alert.Labels[labelJob]
			k += alert.Labels[labelLevel]
			hookID := common.MD5(k)

			// ============================================
			// Generate template variables
			// ============================================
			var vars = map[string]interface{}{}
			for _, v := range common.Cfg.Webhook.LabelMapper {
				vars[v] = alert.Labels[v]
			}
			for _, v := range common.Cfg.Webhook.AnnotationMapper {
				vars[v] = alert.Annotations[v]
			}
			startsAt := alert.StartsAt.In(common.Location)
			endsAt := alert.EndsAt.In(common.Location)
			vars["startsAt"] = startsAt
			vars["endsAt"] = endsAt
			vars["status"] = alert.Status
			logger.Debug("==>", vars)

			// ============================================
			// apply template, if error set default
			// ============================================
			var messageBuffer bytes.Buffer
			if err := hookTemplate.Execute(&messageBuffer, vars); err != nil {
				hookDefaultTemplate.Execute(&messageBuffer, vars)
			}

			jsonMarshal, _ := json.Marshal(alert)
			reqJSON := string(jsonMarshal)
			message := messageBuffer.String()
			hook := &model.Hook{
				HookID:    hookID,
				AlertName: alert.Labels["alertname"],
				Instance:  alert.Labels[labelInstance],
				Job:       alert.Labels[labelJob],
				Level:     alert.Labels[labelLevel],
				Status:    alert.Status,
				StartsAt:  &startsAt,
				EndsAt:    &endsAt,
				HookDetails: []model.HookDetail{
					{
						HookID:  hookID,
						Status:  alert.Status,
						ReqJSON: string(reqJSON),
						Message: messageBuffer.String(),
					},
				},
			}
			jsonMarshal, _ = json.Marshal(hook)

			// ============================================
			// Check ignore hook
			// ============================================
			hookIgnore := &model.HookIgnore{
				Instance:  hook.Instance,
				AlertName: hook.AlertName,
				Status:    hook.Status,
			}

			logger.Debug("IsTarget", hookIgnore.IsTarget())
			if hookIgnore.IsTarget() {
				continue
			}

			// ============================================
			// Send alarm
			// ============================================
			httpClient := &http.Client{Timeout: 3 * time.Second}
			urlencodedParams := strings.Replace(apiParams, "[[message]]", url.QueryEscape(message), -1)
			switch strings.ToUpper(target.Method) {
			case "POST":
				resp, err := httpClient.Post(api, "application/x-www-form-urlencoded", strings.NewReader(urlencodedParams))
				if err != nil {
					logger.Error("API -", err.Error(), string(jsonMarshal))
					continue
				}
				if resp.StatusCode != 200 {
					b, _ := ioutil.ReadAll(resp.Body)
					logger.Error("API code -", resp.StatusCode, "-", string(b), string(jsonMarshal))
					continue
				}
				defer resp.Body.Close()
			case "GET":
				resp, err := httpClient.Get(api + "?" + urlencodedParams)
				if err != nil {
					logger.Error("API -", err.Error(), string(jsonMarshal))
					continue
				}
				if resp.StatusCode != 200 {
					b, _ := ioutil.ReadAll(resp.Body)
					logger.Error("API code -", resp.StatusCode, "-", string(b), string(jsonMarshal))
					continue
				}
				defer resp.Body.Close()
			default:
				logger.Error("unsupport method -", target.Method, string(jsonMarshal))
				continue
			}

			// ============================================
			// Save database
			// ============================================
			switch strings.ToLower(hook.Status) {
			case "firing":
				if hook.Job == "noti" {
					hook.Status = "resolved"
				} else {
					hook.EndsAt = nil
				}
				if err := hook.Upsert("hook_id"); err != nil {
					logger.Error("DB -", err.Error(), string(jsonMarshal))
				}
			case "resolved":
				if err := hook.Upsert(); err != nil {
					logger.Error("DB -", err.Error(), string(jsonMarshal))
				}
			default:
				logger.Info("DB - skip status", hook.Status, string(jsonMarshal))
			}
		}
	}()
}

func loadTemplate(tmpl ...string) error {
	// default template
	if len(tmpl) == 0 {
		logger.Info("Load default template")
		hookTemplate, _ = template.New("template").Parse(defaultTemplate)
		return nil
	}

	logger.Info("open template file", common.Cfg.Webhook.Template)
	tmplData, err := ioutil.ReadFile(tmpl[0])
	if err != nil {
		if tmplData, err = ioutil.ReadFile(common.ABS + "/" + tmpl[0]); err != nil {
			return err
		}
	}

	tempTemplate, err := template.New("template").Parse(string(tmplData))
	if err != nil {
		return err
	}

	hookTemplate = tempTemplate
	return err
}
