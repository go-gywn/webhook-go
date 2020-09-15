package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-gywn/webhook-go/common"
)

var routerGroup *gin.RouterGroup
var logger = common.NewLogger("handler")

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
