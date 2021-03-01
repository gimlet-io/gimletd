package server

import (
	"encoding/json"
	"github.com/gimlet-io/gimletd/dx"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"time"
)

func getReleases(w http.ResponseWriter, r *http.Request) {
	var limit, offset int
	var since, until *time.Time

	var app, env string

	params := r.URL.Query()
	if val, ok := params["limit"]; ok {
		l, err := strconv.Atoi(val[0])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest)+" - "+err.Error(), http.StatusBadRequest)
			return
		}
		limit = l
	}
	if val, ok := params["offset"]; ok {
		o, err := strconv.Atoi(val[0])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest)+" - "+err.Error(), http.StatusBadRequest)
			return
		}
		offset = o
	}

	if val, ok := params["since"]; ok {
		t, err := time.Parse(time.RFC3339, val[0])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest)+" - "+err.Error(), http.StatusBadRequest)
			return
		}
		since = &t
	}
	if val, ok := params["until"]; ok {
		t, err := time.Parse(time.RFC3339, val[0])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest)+" - "+err.Error(), http.StatusBadRequest)
			return
		}
		until = &t
	}

	if val, ok := params["app"]; ok {
		app = val[0]
	}
	if val, ok := params["env"]; ok {
		env = val[0]
	}

	// TODO getting releases from gitops repo
	logrus.Println(limit)
	logrus.Println(offset)
	logrus.Println(since)
	logrus.Println(until)
	logrus.Println(app)
	logrus.Println(env)
	
	releases := []*dx.Release{}

	releasesStr, err := json.Marshal(releases)
	if err != nil {
		logrus.Errorf("cannot serialize artifacts: %s", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(releasesStr)
}
