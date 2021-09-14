package main

import (
	"encoding/base32"
	"fmt"
	"github.com/gimlet-io/gimletd/cmd/config"
	"github.com/gimlet-io/gimletd/git/customScm"
	"github.com/gimlet-io/gimletd/git/customScm/customGithub"
	"github.com/gimlet-io/gimletd/git/nativeGit"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/notifications"
	"github.com/gimlet-io/gimletd/server"
	"github.com/gimlet-io/gimletd/server/token"
	"github.com/gimlet-io/gimletd/store"
	"github.com/gimlet-io/gimletd/worker"
	"github.com/go-chi/chi"
	"github.com/gorilla/securecookie"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
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

	var tokenManager customScm.NonImpersonatedTokenManager
	if config.Github.AppID != "" {
		tokenManager, err = customGithub.NewGithubOrgTokenManager(config)
		if err != nil {
			panic(err)
		}
	}

	notificationsManager := notifications.NewManager()
	notificationsManager = addSlackNotificationProvider(config, notificationsManager)
	if config.Github.AppID != "" {
		notificationsManager.AddProvider(notifications.NewGithubProvider(tokenManager))
	}
	go notificationsManager.Run()

	stopCh := make(chan struct{})
	defer close(stopCh)

	repoCache, err := nativeGit.NewGitopsRepoCache(config.GitopsRepo, config.GitopsRepoDeployKeyPath, stopCh)
	if err != nil {
		panic(err)
	}
	go repoCache.Run()
	logrus.Info("repo cache initialized")

	if config.GitopsRepo != "" &&
		config.GitopsRepoDeployKeyPath != "" {
		gitopsWorker := worker.NewGitopsWorker(
			store,
			config.GitopsRepo,
			config.GitopsRepoDeployKeyPath,
			tokenManager,
			notificationsManager,
			eventsProcessed,
			repoCache,
		)
		go gitopsWorker.Run()
		logrus.Info("Gitops worker started")
	} else {
		logrus.Warn("Not starting GitOps worker. GITOPS_REPO and GITOPS_REPO_DEPLOY_KEY_PATH must be set to start GitOps worker")
	}

	releaseStateWorker := &worker.ReleaseStateWorker{
		GitopsRepo: config.GitopsRepo,
		RepoCache:  repoCache,
		Releases:   releases,
		Perf:       perf,
	}
	go releaseStateWorker.Run()

	branchDeleteEventWorker := worker.NewBranchDeleteEventWorker(
		tokenManager,
		config.RepoCachePath,
		store,
	)
	go branchDeleteEventWorker.Run()

	metricsRouter := chi.NewRouter()
	metricsRouter.Get("/metrics", promhttp.Handler().ServeHTTP)
	go http.ListenAndServe(":8889", metricsRouter)

	r := server.SetupRouter(config, store, notificationsManager, repoCache, perf)
	err = http.ListenAndServe(":8888", r)
	if err != nil {
		panic(err)
	}
}

func addSlackNotificationProvider(config *config.Config, notificationsManager *notifications.ManagerImpl) *notifications.ManagerImpl {
	channelMap := map[string]string{}
	if config.Notifications.ChannelMapping != "" {
		pairs := strings.Split(config.Notifications.ChannelMapping, ",")
		for _, p := range pairs {
			keyValue := strings.Split(p, "=")
			channelMap[keyValue[0]] = keyValue[1]
		}
	}
	notificationsManager.AddProvider(&notifications.SlackProvider{
		Token:          config.Notifications.Token,
		ChannelMapping: channelMap,
		DefaultChannel: config.Notifications.DefaultChannel,
	})

	return notificationsManager
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
