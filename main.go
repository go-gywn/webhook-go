package main

import (
	"github.com/go-gywn/goutil"
	"github.com/go-gywn/webhook-go/handler"
	"github.com/go-gywn/webhook-go/model"
	"golang.org/x/sync/errgroup"
)

var g errgroup.Group
var logger = goutil.GetLogger("main")

func init() {
	// common.LoadConfigure()
	model.OpenDatabase()
}

func main() {
	g.Go(func() error {
		return handler.StartHandler()
	})

	if err := g.Wait(); err != nil {
		panic("startup failed")
	}

}
