package common

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

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

// Cfg config
var Cfg Config

// Location location
var Location *time.Location

// logger
var logger Logger

// ABS absolute path
var ABS string

var err error

// LoadConfigure load config
func init() {
	if ABS, err = filepath.Abs(filepath.Dir(os.Args[0])); err != nil {
		panic("set ABS path failed")
	}

	// Default configure file
	defaultConfigFile := "configure.yml"
	_, err := os.Stat(defaultConfigFile)
	if os.IsNotExist(err) {
		defaultConfigFile = ABS + "/" + defaultConfigFile
	}

	var config, password string
	flag.StringVar(&config, "config", defaultConfigFile, "configuration")
	flag.StringVar(&password, "password", "", "password")
	flag.Parse()

	confContent, err := ioutil.ReadFile(config)
	if err != nil {
		panic(err)
	}

	if err := yaml.Unmarshal(confContent, &Cfg); err != nil {
		panic(err)
	}

	logger = NewLogger("common")

	if Cfg.Key == "" {
		Cfg.Key = "03a73f3e7c9a7b38d196cd34c072567e"
	}

	if password != "" {
		enc, _ := Encrypt(password)
		fmt.Printf("<Encrypted>\n%s\n", enc)
		os.Exit(0)
	}

	// Timezone setting
	if Location, err = time.LoadLocation(Cfg.Timezone); err != nil {
		logger.Info("set timezone failed, set to UTC")
		Location, _ = time.LoadLocation("UTC")
		os.Setenv("TZ", "UTC")
	}

	// Default label setting
	dafaultLabels := []string{"alertname", "instance", "level", "job"}
	if Cfg.Webhook.LabelMapper == nil {
		Cfg.Webhook.LabelMapper = map[string]string{}
	}
	for _, key := range dafaultLabels {
		if val, ok := Cfg.Webhook.LabelMapper[key]; !ok || val == "" {
			Cfg.Webhook.LabelMapper[key] = key
		}
	}
	// Default annotation label setting
	dafaultAnnotations := []string{"description", "summary"}
	if Cfg.Webhook.AnnotationMapper == nil {
		Cfg.Webhook.AnnotationMapper = map[string]string{}
	}
	for _, key := range dafaultAnnotations {
		if val, ok := Cfg.Webhook.AnnotationMapper[key]; !ok || val == "" {
			Cfg.Webhook.AnnotationMapper[key] = key
		}
	}
	logger.Debug("Config", Cfg)
	os.Setenv("TZ", Cfg.Timezone)
}
