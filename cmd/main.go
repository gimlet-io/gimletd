package main

import (
	"encoding/base32"
	"fmt"
	"github.com/gimlet-io/gimletd/cmd/config"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/notifications"
	"github.com/gimlet-io/gimletd/server"
	"github.com/gimlet-io/gimletd/server/token"
	"github.com/gimlet-io/gimletd/store"
	"github.com/gimlet-io/gimletd/worker"
	"github.com/gorilla/securecookie"
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

	err = setupAdminUser(store)
	if err != nil {
		panic(err)
	}

	notificationsManager := notifications.NewManager()
	notificationsManager.AddProvider(
		config.Notifications.Provider,
		config.Notifications.Token,
		config.Notifications.DefaultChannel,
		config.Notifications.ChannelMapping,
	)
	if config.GithubStatusToken != "" {
		notificationsManager.AddProvider("github", config.GithubStatusToken, "", "")
	}
	go notificationsManager.Run()

	if config.GitopsRepo != "" &&
		config.GitopsRepoDeployKeyPath != "" {
		gitopsWorker := worker.NewGitopsWorker(
			store,
			config.GitopsRepo,
			config.GitopsRepoDeployKeyPath,
			config.GithubChartAccessDeployKeyPath,
			notificationsManager,
			eventsProcessed,
		)
		go gitopsWorker.Run()
		logrus.Info("Gitops worker started")
	} else {
		logrus.Warn("Not starting GitOps worker. GITOPS_REPO and GITOPS_REPO_DEPLOY_KEY_PATH must be set to start GitOps worker")
	}

	releaseStateWorker := &worker.ReleaseStateWorker{
		GitopsRepo:              config.GitopsRepo,
		GitopsRepoDeployKeyPath: config.GitopsRepoDeployKeyPath,
		Releases:                releases,
	}
	go releaseStateWorker.Run()

	r := server.SetupRouter(config, store, notificationsManager)
	err = http.ListenAndServe(":8888", r)
	if err != nil {
		panic(err)
	}
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

// Creates an admin user and prints her access token, in case there are no users in the database
func setupAdminUser(store *store.Store) error {
	users, err := store.Users()
	if err != nil {
		return fmt.Errorf("couldn't list users to create admin user %s", err)
	}

	if len(users) == 0 {
		user := &model.User{
			Login: "admin",
			Secret: base32.StdEncoding.EncodeToString(
				securecookie.GenerateRandomKey(32),
			),
			Admin: true,
		}
		err = store.CreateUser(user)
		if err != nil {
			return fmt.Errorf("couldn't create user admin user %s", err)
		}

		token := token.New(token.UserToken, user.Login)
		tokenStr, err := token.Sign(user.Secret)
		if err != nil {
			return fmt.Errorf("couldn't create admin token %s", err)
		}
		logrus.Infof("Admin token created: %s", tokenStr)
	} else {
		logrus.Info("Admin token is already created")
	}

	return nil
}
