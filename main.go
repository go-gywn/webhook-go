package main

import (
	"github.com/go-gywn/webhook-go/common"
	"github.com/go-gywn/webhook-go/handler"
	"github.com/go-gywn/webhook-go/model"
	"golang.org/x/sync/errgroup"
)

var g errgroup.Group
var logger = common.NewLogger("main")

func init() {
	// common.LoadConfigure()
	model.InitDatabase()
}

func main() {
	g.Go(func() error {
		return handler.StartHandler()
	})

	if err := g.Wait(); err != nil {
		panic("startup failed")
	}

}
