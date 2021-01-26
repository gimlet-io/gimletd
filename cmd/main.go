package main

import (
	"fmt"
	"github.com/gimlet-io/gimletd/cmd/config"
	"github.com/gimlet-io/gimletd/server"
	"github.com/gimlet-io/gimletd/store"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"net/http"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		logrus.Warnf("could not load .env file, relying on env vars")
	}

	config, err := config.Environ()
	if err != nil {
		logger := logrus.WithError(err)
		logger.Fatalln("main: invalid configuration")
	}

	initLogging(config)

	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		fmt.Println(config.String())
	}

	store := store.New(config.Database.Driver, config.Database.Config)

	r := server.SetupRouter(config, store)
	http.ListenAndServe(":8888", r)
}

// helper function configures the logging.
func initLogging(c *config.Config) {
	if c.Logging.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	if c.Logging.Trace {
		logrus.SetLevel(logrus.TraceLevel)
	}
	if c.Logging.Text {
		logrus.SetFormatter(&logrus.TextFormatter{
			ForceColors:   c.Logging.Color,
			DisableColors: !c.Logging.Color,
		})
	} else {
		logrus.SetFormatter(&logrus.JSONFormatter{
			PrettyPrint: c.Logging.Pretty,
		})
	}
}
