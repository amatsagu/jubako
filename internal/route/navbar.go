package route

import (
	"database/sql"
	"fmt"
	"net/http"
)

func NewNavSearchHandler(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")

		if query == "" {
			fmt.Println("Received empty search request")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error": "no query provided"}`)
			return
		}

		fmt.Printf("🔍 Search received: %s\n", query)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status": "success", "received": "%s"}`, query)
	}
}
