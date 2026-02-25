package route

import (
	"database/sql"
	"net/http"
)

// Fetches information & returns a json that contains a list of anime series that are airing this week.
// It should also return server time & week day information which allows correct filtering.
//
// This endpoint is always hit by user when starting app,
// in home section to load up to date information about ongoing anime season.
//
// /api/anime-timetable
func NewAnimeTimetableHandler(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// db variable is a pointer to sqlite3 local file that we can use for quick cache.
	}
}
