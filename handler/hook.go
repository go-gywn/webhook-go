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

// StartHostAPI start host API
func startHook(r *gin.RouterGroup) {
	// =======================
	// start message thread
	// =======================
	chanHook := make(chan t.Alert, 100)

	// load template
	if err := loadTemplate(common.Cfg.Webhook.Template); err != nil {
		fmt.Println(err)
		loadTemplate()
	}

	for i := 0; i < 5; i++ {
		createHookShooter(chanHook)
	}

	r.POST("/hook/send", func(c *gin.Context) {
		var err error
		var params t.Data

		// bind template json data
		err = c.BindJSON(&params)
		if ErrorIf(c, err) {
			fmt.Println(err)
			return
		}

		for _, alert := range params.Alerts {
			chanHook <- alert
		}

		Success(c, "ok")
	})

	r.POST("/hook/ignore", func(c *gin.Context) {
		var err error
		var params model.HookIgnore

		// bind template json data
		err = c.Bind(&params)
		if ErrorIf(c, err) {
			fmt.Println(err)
			return
		}

		// upsert hook ignore
		params.Upsert()

		Success(c, "ok")
	})

	r.DELETE("/hook/ignore", func(c *gin.Context) {
		var err error
		var params model.HookIgnore

		// bind template json data
		err = c.Bind(&params)
		if ErrorIf(c, err) {
			logger.Error(err)
			return
		}

		// upsert hook ignore
		params.Delete()

		Success(c, "ok")
	})

	r.POST("/hook/template/reload", func(c *gin.Context) {
		err := loadTemplate(common.Cfg.Webhook.Template)
		if ErrorIf(c, err) {
			return
		}
		Success(c, "ok")
	})

	r.POST("/hook/test", func(c *gin.Context) {
		logger.Info("GET request")
		logger.Info(c.GetPostForm("message"))
		Success(c, "ok")
	})

	r.GET("/hook/test", func(c *gin.Context) {
		logger.Info("GET request")
		logger.Info(c.GetPostForm("message"))
		Success(c, "ok")
	})

	r.GET("/hook/test/shoot", func(c *gin.Context) {
		resp1, err := http.Post("http://127.0.0.1"+common.Cfg.Port+"/webhook/hook/send", "application/json", bytes.NewBufferString(msg01))
		if err != nil {
			logger.Error(err)
		}
		defer resp1.Body.Close()
		resp2, err := http.Post("http://127.0.0.1"+common.Cfg.Port+"/webhook/hook/send", "application/json", bytes.NewBufferString(msg02))
		if err != nil {
			logger.Error(err)
		}
		defer resp2.Body.Close()
		resp3, err := http.Post("http://127.0.0.1"+common.Cfg.Port+"/webhook/hook/send", "application/json", bytes.NewBufferString(msg03))
		if err != nil {
			logger.Error(err)
		}
		defer resp3.Body.Close()
		resp4, err := http.Post("http://127.0.0.1"+common.Cfg.Port+"/webhook/hook/send", "application/json", bytes.NewBufferString(msg04))
		if err != nil {
			logger.Error(err)
		}
		defer resp4.Body.Close()
		Success(c, "ok")
	})

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

func createHookShooter(chanHook chan t.Alert) {
	alertname := common.Cfg.Webhook.LabelMapper["alertname"]
	labelInstance := common.Cfg.Webhook.LabelMapper["instance"]
	labelLevel := common.Cfg.Webhook.LabelMapper["level"]
	labelJob := common.Cfg.Webhook.LabelMapper["job"]
	targets := common.Cfg.Webhook.Targets
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
			k += alert.Labels[alertname]
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
				hook.EndsAt = nil
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

var hookTemplate *template.Template
var hookDefaultTemplate *template.Template
var defaultTemplate = `[{{ .status }}] {{ .summary }}
> Instance: {{ .instance }}
> Level: {{ .level }}{{ if eq .status "firing" }}
> Start: {{ .startsAt.Format "01/02 15:04:05 MST" }}{{ else }}
> Start: {{ .endsAt.Format "01/02 15:04:05 MST" }}
> End: {{ .endsAt.Format "01/02 15:04:05 MST" }}{{ end }}
> Description: {{ .description }}`

var msg01 = `{
	"receiver": "web\\.hook",
	"status": "firing",
	"alerts": [
		{
		"status": "firing",
		"labels": {
			"alertname": "node_cpu_usage",
			"instance": "pmm-server",
			"instance0": "pmm-server",
			"job": "linux",
			"level": "critical"
		},
		"annotations": {
			"description": "CPU usage excessive, current 2.95 %",
			"summary": "CPU usage excessive"
		},
		"startsAt": "2019-02-13T13:43:41.825374766Z",
		"endsAt": "0001-01-01T00:00:00Z",
		"generatorURL": "http://localhost:9090/prometheus/graph?g0.expr=round%28raw%3Anode_cpu_usage%2C+0.01%29+%3E+30\u0026g0.tab=1",
		"fingerprint":"56a752262c346328"
		}
	],
	"groupLabels": {
		"alertname": "node_cpu_usage",
		"instance": "pmm-server",
		"level": "critical"
	},
	"commonLabels": {
		"alertname": "node_cpu_usage",
		"instance": "pmm-server",
		"instance0": "pmm-server",
		"job": "linux",
		"level": "critical"
	},
	"commonAnnotations": {
		"description": "CPU usage excessive, current 2.95 %",
		"summary": "CPU usage excessive"
	},
	"externalURL": "http://26ac5ef8f438:19000",
	"version": "4",
	"groupKey": "{}:{alertname=\"node_cpu_usage\", instance=\"pmm-server\", level=\"critical\"}"
}`

var msg02 = `{
	"receiver": "web\\.hook",
	"status": "resolved",
	"alerts": [
		{
			"status": "resolved",
			"labels": {
				"alertname": "node_cpu_usage",
				"instance": "pmm-server",
				"instance0": "pmm-server",
				"job": "linux",
				"level": "critical"
			},
			"annotations": {
				"description": "CPU usage excessive, current 2.95 %",
				"summary": "CPU usage excessive"
			},
			"startsAt": "2019-02-13T13:43:41.825374766Z",
			"endsAt": "2019-02-13T13:45:56.825374766Z",
			"generatorURL": "http://localhost:9090/prometheus/graph?g0.expr=round%28raw%3Anode_cpu_usage%2C+0.01%29+%3E+30\u0026g0.tab=1",
			"fingerprint":"56a752262c346328"
		}
	],
	"groupLabels": {
		"alertname": "node_cpu_usage",
		"instance": "pmm-server",
		"level": "critical"
	},
	"commonLabels": {
		"alertname": "node_cpu_usage",
		"instance": "pmm-server",
		"instance0": "pmm-server",
		"job": "linux",
		"level": "critical"
	},
	"commonAnnotations": {
		"description": "CPU usage excessive, current 2.95 %",
		"summary": "CPU usage excessive"
	},
	"externalURL": "http://26ac5ef8f438:19000",
	"version": "4",
	"groupKey": "{}:{alertname=\"node_cpu_usage\", instance=\"pmm-server\", level=\"critical\"}"
}`

var msg03 = `{
	"receiver": "web\\.hook",
	"status": "firing",
	"alerts": [
		{
		"status": "firing",
		"labels": {
			"alertname": "node_cpu_usage",
			"instance": "pmm-server",
			"instance0": "pmm-server",
			"job": "linux",
			"level": "warning"
		},
		"annotations": {
			"description": "CPU usage excessive, current 2.95 %",
			"summary": "CPU usage excessive"
		},
		"startsAt": "2019-02-13T13:43:41.825374766Z",
		"endsAt": "0001-01-01T00:00:00Z",
		"generatorURL": "http://localhost:9090/prometheus/graph?g0.expr=round%28raw%3Anode_cpu_usage%2C+0.01%29+%3E+30\u0026g0.tab=1",
		"fingerprint":"f0ad52edec7bfa22"
		}
	],
	"groupLabels": {
		"alertname": "node_cpu_usage",
		"instance": "pmm-server",
		"level": "warning"
	},
	"commonLabels": {
		"alertname": "node_cpu_usage",
		"instance": "pmm-server",
		"instance0": "pmm-server",
		"job": "linux",
		"level": "warning"
	},
	"commonAnnotations": {
		"description": "CPU usage excessive, current 2.95 %",
		"summary": "CPU usage excessive"
	},
	"externalURL": "http://26ac5ef8f438:19000",
	"version": "4",
	"groupKey": "{}:{alertname=\"node_cpu_usage\", instance=\"pmm-server\", level=\"warning\"}"
}`

var msg04 = `{
	"receiver": "web\\.hook",
	"status": "resolved",
	"alerts": [
		{
		"status": "resolved",
		"labels": {
			"alertname": "node_cpu_usage",
			"instance": "pmm-server",
			"instance0": "pmm-server",
			"job": "linux",
			"level": "warning"
		},
		"annotations": {
			"description": "CPU usage excessive, current 2.95 %",
			"summary": "CPU usage excessive"
		},
		"startsAt": "2019-02-13T13:43:41.825374766Z",
		"endsAt": "2019-02-13T13:45:56.825374766Z",
		"generatorURL": "http://localhost:9090/prometheus/graph?g0.expr=round%28raw%3Anode_cpu_usage%2C+0.01%29+%3E+30\u0026g0.tab=1",
		"fingerprint":"f0ad52edec7bfa22"
		}
	],
	"groupLabels": {
		"alertname": "node_cpu_usage",
		"instance": "pmm-server",
		"level": "warning"
	},
	"commonLabels": {
		"alertname": "node_cpu_usage",
		"instance": "pmm-server",
		"instance0": "pmm-server",
		"job": "linux",
		"level": "warning"
	},
	"commonAnnotations": {
		"description": "CPU usage excessive, current 2.95 %",
		"summary": "CPU usage excessive"
	},
	"externalURL": "http://26ac5ef8f438:19000",
	"version": "4",
	"groupKey": "{}:{alertname=\"node_cpu_usage\", instance=\"pmm-server\", level=\"warning\"}"
}`
