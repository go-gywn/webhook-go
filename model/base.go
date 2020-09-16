package model

import (
	"os"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/go-gywn/webhook-go/common"
	_ "github.com/go-sql-driver/mysql" // for xorm
)

// var orm *xorm.Engine
var db *gorm.DB
var logger = common.NewLogger("model")

// ignore hook map
// var hookIgnoreMap = make(map[string]HookIgnore)
var hookIgnoreMtx = &sync.Mutex{}
var hookIgnoreMap map[string]HookIgnore

// InitDatabase new database connection
func InitDatabase() {
	var err error
	host := common.Cfg.Database.Host
	user := common.Cfg.Database.User
	schema := common.Cfg.Database.Schema
	pass, err := common.Decrypt(common.Cfg.Database.Pass)
	if err != nil {
		logger.Error("Invalid password, exit")
		os.Exit(1)
	}

	dsn := user + ":" + pass + "@tcp(" + host + ")/" + schema + "?charset=utf8&parseTime=True&loc=Local"
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	common.PanicIf(err)

	sqlDB, err := db.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(1 * time.Hour)

	err = db.AutoMigrate(&Hook{})
	common.PanicIf(err)
	err = db.AutoMigrate(&HookDetail{})
	common.PanicIf(err)
	err = db.AutoMigrate(&HookIgnore{})
	common.PanicIf(err)

	startHookIgnoreMapThread()
}
