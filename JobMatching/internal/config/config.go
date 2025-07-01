package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	LinkedInEmail    string
	LinkedInPassword string
	JobKeywords      string
	Location         string
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &Config{
		LinkedInEmail:    viper.GetString("linkedin_email"),
		LinkedInPassword: viper.GetString("linkedin_password"),
		JobKeywords:      viper.GetString("job_keywords"),
		Location:         viper.GetString("location"),
	}
	return cfg, nil
}
