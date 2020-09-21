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
	Base     string   `yaml:"base"`
	Port     string   `yaml:"port"`
	Timezone string   `yaml:"timezone"`
	Key      string   `yaml:"key"`
	Database Database `yaml:"database"`
	Webhook  Webhook  `yaml:"webhook"`
}

// Database Database
type Database struct {
	Host   string `yaml:"host"`
	User   string `yaml:"user"`
	Pass   string `yaml:"pass"`
	Schema string `yaml:"schema"`
}

// Webhook Webhook
type Webhook struct {
	CacheSyncSec     int                      `yaml:"cacheSyncSec"`
	Template         string                   `yaml:"template"`
	LabelMapper      map[string]string        `yaml:"labelMapper"`
	AnnotationMapper map[string]string        `yaml:"annotationMapper"`
	Targets          map[string]WebhookTarget `yaml:"targets"`
}

// WebhookTarget webhook target
type WebhookTarget struct {
	API    string
	Params string
	Method string
}

// CONF default config (overwritten by configure.yml)
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
	// Load default configuration
	// ==========================
	if err = yaml.Unmarshal([]byte(defaultConfigure), &CONF); err != nil {
		logger.Fatal(err)
	}

	// ==========================
	// Read config file and load
	// ==========================
	var b []byte
	if b, err = ioutil.ReadFile(config); err != nil {
		logger.Fatal(err)
	}

	if err = yaml.Unmarshal(b, &CONF); err != nil {
		logger.Fatal(err)
	}

	// ==========================
	// encrypt password to use
	// ==========================
	if password != "" {
		crypto := goutil.GetCrypto(CONF.Key)
		fmt.Printf("<Encrypted>\n%s\n", crypto.EncryptAES(password))
		os.Exit(0)
	}

	// ==========================
	// Config check
	// ==========================

	// Timezone setting
	if location, err = time.LoadLocation(CONF.Timezone); err != nil {
		logger.Fatal("Load timezone '", CONF.Timezone, "' failed")
	}
	os.Setenv("TZ", CONF.Timezone)

	// Label & Annotation mapper check
	if len(CONF.Webhook.LabelMapper) == 0 || len(CONF.Webhook.AnnotationMapper) == 0 {
		logger.Fatal("Mapper has no entry, exit")
	}

	logger.Println("start with", CONF)
}

// GetLocation GetLocation
func GetLocation() *time.Location {
	return location
}

var defaultConfigure = `
base: "webhook"
port: ":52802"
timezone: "Asia/Seoul"
key: "03a73f3e7c9a7b38d196cd34c072567e"

database:
  host: "127.0.0.1:3306"
  user: "dbadmin"
  pass: "l-6ILJ3Y6yahD7ibKwNe-t12rt1ahMUU6mI="
  schema: "dbadmin"

webhook:
  cacheSyncSec: 60
  template: "tempalte.tpl"
  labelMapper:
    alertname: "alertname"
    instance: "instance"
    level: "level"
    job: "job"
  annotationMapper:
    description: "description"
    summary: "summary"
  targets:
    critical:
      api: "http://127.0.0.1:52802/webhook/hook/test"
      params: "id=12345&message=[[message]]"
      method: "POST"
    warning:
      api: "http://127.0.0.1:52802/webhook/hook/test"
      params: "id=54321&message=[[message]]"
      method: "POST"
`
