package handler

import (
	"fmt"
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
	if err := loadTemplate(common.Cfg.Webhook.Template); err != nil {
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
		err := loadTemplate(common.Cfg.Webhook.Template)
		if ErrorIf(c, err) {
			logger.Error(err)
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
