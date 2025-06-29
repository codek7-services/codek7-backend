package api

import (
	"fmt"
	"net/http"
	"time"
)

func (a API) HealthCheck(w http.ResponseWriter,r *http.Request) {
  w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`) 
}
