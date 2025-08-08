package apiserver

import (
	"fmt"
	"net/http"

	"github.com/eu-sovereign-cloud/ecp/pkg/logger"
)

// Start starts the backend HTTP server on the given address.
func Start(addr string) {
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	logger.Log.Infow("Starting API server on", "addr", addr)
	logger.Log.Fatal(http.ListenAndServe(addr, nil))
}
