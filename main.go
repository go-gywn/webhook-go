package main

import (
	"github.com/go-gywn/goutil"
	"github.com/go-gywn/webhook-go/handler"
	"github.com/go-gywn/webhook-go/model"
	"golang.org/x/sync/errgroup"
)

var g errgroup.Group
var logger = goutil.GetLogger()

func init() {
	logger.Debug("[WARNING] Running in \"debug\" mode. Switch to \"release\" mode in production.")
	logger.Debug("- using env:   export LOGGING=release\n")
	model.OpenDatabase()
}

func main() {

	g.Go(func() error {
		return handler.StartHandler()
	})

	if err := g.Wait(); err != nil {
		logger.Panic("Startup failed", err)
	}

}
