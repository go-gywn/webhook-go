package model

import (
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/go-gywn/goutil"
	"github.com/go-gywn/webhook-go/common"
	_ "github.com/go-sql-driver/mysql" // for gorm
)

var db *gorm.DB
var logger = goutil.GetLogger()
var cryptor = goutil.GetCrypto(common.CONF.Key)

// ignore hook map
var hookIgnoreMtx = &sync.Mutex{}
var hookIgnoreMap map[string]HookIgnore

// OpenDatabase new database connection
func OpenDatabase() {
	var err error

	// ===================================
	// Open database
	// ===================================
	host := common.CONF.Database.Host
	user := common.CONF.Database.User
	schema := common.CONF.Database.Schema
	pass := cryptor.DecryptAES(common.CONF.Database.Pass)

	dsn := user + ":" + pass + "@tcp(" + host + ")/" + schema + "?charset=utf8&parseTime=True&loc=Local"
	cfg := &gorm.Config{
		SkipDefaultTransaction: true,
	}
	if db, err = gorm.Open(mysql.Open(dsn), cfg); err != nil {
		logger.Fatal("mysql.Open failed - ", err)
		logger.Fatal(err)
	}

	sqlDB, err := db.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(1 * time.Hour)

	// ===================================
	// sync table
	// ===================================
	var syncTargets = []interface{}{
		&Hook{},
		&HookDetail{},
		&HookIgnore{},
	}
	if err = db.AutoMigrate(syncTargets...); err != nil {
		logger.Fatal("db.AutoMigrate failed - ", err)
	}

	// ===================================
	// start cache batch
	// ===================================
	startHookIgnoreMapThread()
}
