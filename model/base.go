package model

import (
	"os"
	"reflect"
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

	syncTable()
	startHookIgnoreMapThread()
}

// syncTable sync table and data
func syncTable() {
	var err error
	err = db.AutoMigrate(&Hook{})
	common.PanicIf(err)
	err = db.AutoMigrate(&HookDetail{})
	common.PanicIf(err)
	err = db.AutoMigrate(&HookIgnore{})
	common.PanicIf(err)
}

// UpdateHookIgnoreMap update ignore map
func UpdateHookIgnoreMap() {
	var hookIgnores []HookIgnore

	logger.Info("Update hook ignore map start")
	// Get all hook ignores from database
	if result := db.Find(&hookIgnores); result.Error != nil {
		logger.Error(result.Error)
		return
	}

	// Cache update
	tmpHookIgnoreMap := make(map[string]HookIgnore)
	for _, hookIgnore := range hookIgnores {
		k := hookIgnore.GetKey()
		tmpHookIgnoreMap[k] = hookIgnore
	}
	hookIgnoreMtx.Lock()
	hookIgnoreMap = tmpHookIgnoreMap
	hookIgnoreMtx.Unlock()
	logger.Debug("hookIgnoreMap", hookIgnoreMap)
	logger.Info("Update hook ignore map end")
}

func startHookIgnoreMapThread() {
	go func() {
		for {
			UpdateHookIgnoreMap()
			time.Sleep(5 * time.Minute)
		}
	}()
}

// GetUpsertAllColumns return all column names
func GetUpsertAllColumns(value interface{}) []string {
	tx := db.Model(value)
	el := reflect.ValueOf(value).Elem()
	s := []string{}
	for i := 0; i < el.NumField(); i++ {
		t := el.Type().Field(i)
		f := tx.Statement.Schema.ParseField(t)
		if !f.Updatable {
			continue
		}
		if f.DBName == "" {
			f.DBName = tx.NamingStrategy.ColumnName("", f.Name)
		}
		s = append(s, f.DBName)
	}
	return s
}

// GetUpsertAppendColumns return all column names
func GetUpsertAppendColumns(value interface{}, columns []string) []string {
	tx := db.Model(value)
	el := reflect.ValueOf(value).Elem()
	for i := 0; i < el.NumField(); i++ {
		t := el.Type().Field(i)
		f := tx.Statement.Schema.ParseField(t)

		if _, ok := f.TagSettings["AUTOUPDATETIME"]; ok || (f.Name == "UpdatedAt" && (f.DataType == "time" || f.DataType == "int" || f.DataType == "uint")) {
			if f.DBName == "" {
				f.DBName = tx.NamingStrategy.ColumnName("", f.Name)
			}
			columns = append(columns, f.DBName)
			continue
		}
	}

	return columns
}
