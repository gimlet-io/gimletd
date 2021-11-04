module github.com/gimlet-io/gimletd

go 1.16

require (
	github.com/Masterminds/sprig/v3 v3.1.0
	github.com/cenkalti/backoff/v4 v4.1.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fluxcd/pkg/runtime v0.3.1
	github.com/gimlet-io/gimlet-cli v0.4.0
	github.com/go-chi/chi v1.5.1
	github.com/go-chi/cors v1.1.1
	github.com/go-git/go-billy/v5 v5.3.1
	github.com/go-git/go-git/v5 v5.3.0
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gobwas/glob v0.2.3
	github.com/google/go-github/v37 v37.0.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/securecookie v1.1.1
	github.com/joho/godotenv v1.3.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lib/pq v1.8.0
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/otiai10/copy v1.6.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/russross/meddler v1.0.0
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.7.0
	github.com/whilp/git-urls v1.0.0
	golang.org/x/oauth2 v0.0.0-20191202225959-858c2ad4c8b6
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	helm.sh/helm/v3 v3.4.1
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	sigs.k8s.io/yaml v1.2.0
)

//replace github.com/go-git/go-git/v5 => github.com/juliens/go-git/v5 v5.4.3-0.20210820144752-1cb831023bcc
replace github.com/go-git/go-git/v5 => github.com/gimlet-io/go-git/v5 v5.2.1-0.20210917081253-a2ab483ba818
