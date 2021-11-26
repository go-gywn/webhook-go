package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"text/template"
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
	for i := 0; i < 5; i++ {
		hookSender(chanHook)
	}

	// =======================
	// load template
	// =======================
	if err := loadTemplate(common.CONF.Webhook.Template); err != nil {
		fmt.Println(err)
		loadTemplate()
	}

	r.POST("/hook/send", func(c *gin.Context) {
		var err error
		var params t.Data

		// bind template json data
		err = c.BindJSON(&params)
		if ErrorIf(c, err) {
			logger.Error(err)
			return
		}

		for _, alert := range params.Alerts {
			chanHook <- alert
		}

		Success(c, "ok")
	})

	r.GET("/hook/shoot", func(c *gin.Context) {
		var err error
		var params model.Notification

		err = c.Bind(&params)
		if ErrorIf(c, err) {
			logger.Error(err)
			return
		}

		err = params.CheckForm()
		if ErrorIf(c, err) {
			logger.Error(err)
			return
		}

		chanHook <- toPromAlert(params)
		Success(c, "ok")
	})

	r.POST("/hook/shoot", func(c *gin.Context) {
		var err error
		var params model.Notification

		err = c.Bind(&params)
		if ErrorIf(c, err) {
			logger.Error(err)
			return
		}

		err = params.CheckForm()
		if ErrorIf(c, err) {
			logger.Error(err)
			return
		}

		chanHook <- toPromAlert(params)
		Success(c, "ok")
	})

	r.POST("/hook/ignore", func(c *gin.Context) {
		var err error
		var params model.HookIgnore

		// bind template json data
		err = c.Bind(&params)
		if ErrorIf(c, err) {
			logger.Error(err)
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
		err := loadTemplate(common.CONF.Webhook.Template)
		if ErrorIf(c, err) {
			logger.Error(err)
			return
		}
		Success(c, "ok")
	})

	r.GET("/hook/template", func(c *gin.Context) {
		content := readTemplate(common.CONF.Webhook.Template)
		Success(c, content)
	})

	r.POST("/hook/template", func(c *gin.Context) {
		var err error
		content, b := c.GetPostForm("content")
		if !b {
			err = fmt.Errorf("content is empty")
		}

		if ErrorIf(c, err) {
			logger.Error(err)
			return
		}

		writeTemplate(common.CONF.Webhook.Template, content)
		Success(c, "OK")
	})

	r.POST("/hook/template/check", func(c *gin.Context) {
		var err error
		content, b := c.GetPostForm("content")
		if !b {
			err = fmt.Errorf("content is empty")
		}

		if ErrorIf(c, err) {
			logger.Error(err)
			return
		}

		_, err = fileUtil.GetTemplate("template", content)
		if ErrorIf(c, err) {
			logger.Error(err)
			return
		}

		Success(c, "OK")
	})

	r.POST("/hook/test", func(c *gin.Context) {
		logger.Info("POST request")
		logger.Info(c.GetPostForm("message"))
		Success(c, "ok")
	})

	r.GET("/hook/test", func(c *gin.Context) {
		logger.Info("GET request")
		logger.Info(c.GetQuery("message"))
		Success(c, "ok")
	})

	r.GET("/hook/ignores", func(c *gin.Context) {
		var err error
		var params model.HookIgnore
		var lists []model.HookIgnore

		lists, err = params.GetList()
		if ErrorIf(c, err) {
			logger.Error(err)
			return
		}
		Success(c, lists)
	})

	r.GET("/hook/alerts", func(c *gin.Context) {
		var err error
		var params model.Hook
		var lists []model.Hook

		// bind template json data
		err = c.Bind(&params)
		if ErrorIf(c, err) {
			logger.Error(err)
			return
		}

		rowsValue, _ := c.GetQuery("rows")
		limit := common.ParseInt(rowsValue)
		if limit == 0 {
			limit = 100
		}

		lists, err = params.GetList(limit)
		if ErrorIf(c, err) {
			logger.Error(err)
			return
		}
		Success(c, lists)
	})

}

func toPromAlert(o model.Notification) t.Alert {
	tmpAlert := t.Alert{
		Status: "firing",
		Labels: t.KV{
			labelAlertname: o.Alertname,
			labelInstance:  o.Instance,
			labelLevel:     o.Level,
			labelJob:       "noti",
		},
		Annotations: t.KV{
			labelSummary:     o.Summary,
			labelDescription: o.Message,
		},
		StartsAt: time.Now().UTC(),
		EndsAt:   time.Now().UTC(),
	}
	return tmpAlert
}

func hookSender(chanHook chan t.Alert) {
	var targets = common.CONF.Webhook.Targets
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
			hookID := crypt.MD5(k)

			// ============================================
			// Generate template variables
			// ============================================
			var vars = map[string]interface{}{}
			for _, v := range common.CONF.Webhook.LabelMapper {
				vars[v] = alert.Labels[v]
			}
			for _, v := range common.CONF.Webhook.AnnotationMapper {
				vars[v] = alert.Annotations[v]
			}
			startsAt := alert.StartsAt.In(common.GetLocation())
			endsAt := alert.EndsAt.In(common.GetLocation())
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
				Ignored:   "N",
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
			logger.Debug("Check ignore hook")
			hookIgnore := &model.HookIgnore{
				Instance:  hook.Instance,
				AlertName: hook.AlertName,
				Status:    hook.Status,
			}

			if hookIgnore.IsTarget() {
				hook.Ignored = "Y"
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
				continue
			}

			if hook.Ignored == "Y" {
				continue
			}

			// ============================================
			// Send alarm
			// ============================================
			httpClient := &http.Client{Timeout: 3 * time.Second}
			urlencodedParams := strings.Replace(apiParams, "[[message]]", url.QueryEscape(strings.TrimSpace(message)), -1)
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
			}
		}
	}()
}

func loadTemplate(path ...string) (err error) {
	if len(path) == 0 {
		hookTemplate, _ = fileUtil.GetTemplate("template", defaultTemplate)
	}
	logger.Info("open template file ", path)
	hookTemplate, err = fileUtil.GetTemplate("template", fileUtil.ReadFile(path[0]))
	return
}

func readTemplate(path ...string) (content string) {
	if len(path) == 0 {
		logger.Debug("get default template ")
		content = defaultTemplate
	} else {
		logger.Debug("read template file ", path)
		content = fileUtil.ReadFile(path[0])
	}
	return
}

func writeTemplate(path string, content string) (err error) {
	err = ioutil.WriteFile(path, []byte(content), 0644)
	return
}
