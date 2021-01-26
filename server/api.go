package server

import (
	"encoding/json"
	"net/http"
)

func saveArtifact(w http.ResponseWriter, r *http.Request) {
	var params map[string]interface{}
	json.NewDecoder(r.Body).Decode(&params)

	//ctx := r.Context()
	//user := ctx.Value("user").(*model.User)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("{}"))
}