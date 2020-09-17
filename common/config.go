package common

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/go-gywn/goutil"
	"gopkg.in/yaml.v2"
)

// Config config
type Config struct {
	Base     string
	Port     string
	Timezone string
	Key      string
	LogLevel string
	Database Database
	Webhook  Webhook
}

// Database Database
type Database struct {
	Host   string
	User   string
	Pass   string
	Schema string
}

// Webhook Webhook
type Webhook struct {
	SyncSec          int
	Template         string
	LabelMapper      map[string]string
	AnnotationMapper map[string]string
	Targets          map[string]WebhookTarget
}

// WebhookTarget webhook target
type WebhookTarget struct {
	API    string
	Params string
	Method string
}

// CONF config
var CONF Config

var location *time.Location
var logger = goutil.GetLogger()

// LoadConfigure load config
func init() {
	var err error

	// ==========================
	// get os parameters
	// ==========================
	var config, password string
	flag.StringVar(&config, "config", "configure.yml", "configuration")
	flag.StringVar(&password, "password", "", "password")
	flag.Parse()

	// ==========================
	// Read config file
	// ==========================
	var b []byte
	if b, err = ioutil.ReadFile(config); err != nil {
		logger.Error(err)
		return
		//panic(err)
	}

	if err = yaml.Unmarshal(b, &CONF); err != nil {
		panic(err)
	}

	// ==========================
	// encrypt password to use
	// ==========================
	if CONF.Key == "" {
		CONF.Key = "03a73f3e7c9a7b38d196cd34c072567e"
	}

	if password != "" {
		crypto := goutil.GetCrypto(CONF.Key)
		fmt.Printf("<Encrypted>\n%s\n", crypto.EncryptAES(password))
		os.Exit(0)
	}

	// ==========================
	// Config parameter check
	// ==========================

	// Timezone setting
	if location, err = time.LoadLocation(CONF.Timezone); err != nil {
		logger.Info("set timezone failed, set to UTC")
		location, _ = time.LoadLocation("UTC")
		os.Setenv("TZ", CONF.Timezone)
	}

	// Load default label setting
	dafaultLabels := []string{"alertname", "instance", "level", "job"}
	if CONF.Webhook.LabelMapper == nil {
		CONF.Webhook.LabelMapper = map[string]string{}
	}
	for _, key := range dafaultLabels {
		if val, ok := CONF.Webhook.LabelMapper[key]; !ok || val == "" {
			CONF.Webhook.LabelMapper[key] = key
		}
	}
	// Load default annotation label setting
	dafaultAnnotations := []string{"description", "summary"}
	if CONF.Webhook.AnnotationMapper == nil {
		CONF.Webhook.AnnotationMapper = map[string]string{}
	}
	for _, key := range dafaultAnnotations {
		if val, ok := CONF.Webhook.AnnotationMapper[key]; !ok || val == "" {
			CONF.Webhook.AnnotationMapper[key] = key
		}
	}

	// Default cache load sec
	if CONF.Webhook.SyncSec == 0 {
		CONF.Webhook.SyncSec = 60
	}
	logger.Println("start with", CONF)
}

// GetLocation GetLocation
func GetLocation() *time.Location {
	return location
}
