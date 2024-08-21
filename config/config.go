package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)


func init() {
	localDbConnString := "mongodb://db-01.hydra.local:27017,db-02.hydra.local:27027/hydra_api?replicaSet=rs0"
	viper.AutomaticEnv()
	envReplacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(envReplacer)
	viper.SetDefault("port", 8888)
	viper.SetDefault("log.level", "debug")
	viper.SetDefault("env", "dev")
	viper.SetDefault("dbaas.mongodb.endpoint", localDbConnString)
	viper.SetDefault("database.name", "hydra_api_dev")
	// set max pool size for mgo
	viper.SetDefault("database.maxPoolSize", 4000)
	// set max idle time for mgo
	maxIdleTimeout := time.Hour * 12
	viper.SetDefault("database.maxConnIdleTime", int(maxIdleTimeout.Milliseconds()))

	viper.SetDefault("external.azps", []string{"cmaas-externo-client-qa"})
	viper.SetDefault("client.id", "CqgMYSIm/9cbA2u8pd6DYQ==")
	viper.SetDefault("sentry.timeout", 3)
	viper.SetDefault("sentry.dsn", "")
}
