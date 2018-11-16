package config

import (
	"github.com/kelseyhightower/envconfig"
)

// Config struct used for configuration of app with env variables
type Config struct {
	SlackToken         string `envconfig:"SLACK_TOKEN" required:"true"`
	DatabaseURL        string `envconfig:"DATABASE" required:"true" default:"comedian:comedian@/comedian?parseTime=true"`
	HTTPBindAddr       string `envconfig:"HTTP_BIND_ADDR" required:"true" default:"0.0.0.0:8080"`
	NotifierInterval   int    `envconfig:"REMINDER_INTERVAL" required:"true" default:"2"`
	ManagerSlackUserID string `envconfig:"SUPER_ADMIN_ID" required:"true"`
	ReportingChannel   string `envconfig:"REPORT_CHANNEL" required:"true"`
	ReportTime         string `envconfig:"REPORT_TIME" required:"true" default:"13:05"`
	Language           string `envconfig:"LANGUAGE" required:"true" default:"en_US"`
	ReminderRepeatsMax int    `envconfig:"MAX_REMINDERS" required:"true" default:"5"`
	ReminderTime       int64  `envconfig:"WARNING_TIME" required:"true" default:"5"`
	Translate          Translate
}

// Get method processes env variables and fills Config struct
func Get() (Config, error) {
	var c Config
	err := envconfig.Process("comedian", &c)
	if err != nil {
		return c, err
	}
	t, err := GetTranslation(c.Language)
	if err != nil {
		return c, err
	}
	c.Translate = t
	return c, nil
}
