package route

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"jubako/internal/model"
	"net/http"
	"sync"
	"time"

	"github.com/amatsagu/lumo"
)

const aniListAPI = "https://graphql.anilist.co"

const timetableQuery = `
query ($airingAt_greater: Int, $airingAt_lesser: Int, $page: Int) {
  Page(page: $page, perPage: 50) {
    pageInfo {
      total
      perPage
      currentPage
      lastPage
      hasNextPage
    }
    airingSchedules(airingAt_greater: $airingAt_greater, airingAt_lesser: $airingAt_lesser, sort: TIME) {
      id
      airingAt
      episode
      media {
        id
        idMal
        title {
          romaji
          english
          native
        }
        coverImage {
          large
          color
        }
        description
        genres
        averageScore
        isAdult
      }
    }
  }
}
`

var (
	isRefreshing     bool
	refreshMutex     sync.Mutex
	lastRefreshTried time.Time
)

const maxRetries = 3

// NewAnimeTimetableHandler fetches information & returns a json that contains a list of anime series that are airing this week.
func NewAnimeTimetableHandler(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	// Initialize cache table
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS anime_timetable (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		data TEXT,
		updated_at DATETIME
	)`)
	if err != nil {
		lumo.Error("Failed to create anime_timetable table: %v", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var cachedData string
		var updatedAt time.Time
		err := db.QueryRow("SELECT data, updated_at FROM anime_timetable WHERE id = 1").Scan(&cachedData, &updatedAt)

		if err == nil {
			// If cache is still fresh (< 10 mins), serve it
			if time.Since(updatedAt) < 10*time.Minute {
				lumo.Debug("Serving fresh anime timetable from cache.")
				w.Write([]byte(cachedData))
				return
			}

			// If cache is stale (> 10 mins), serve it but trigger background refresh
			lumo.Info("Serving stale anime timetable, triggering background refresh...")
			w.Write([]byte(cachedData))

			go triggerBackgroundRefresh(db)
			return
		}

		// No cache at all - must wait for fresh data
		lumo.Info("No cache found, fetching fresh anime timetable from AniList...")
		timetable, err := fetchTimetableFromAniList()
		if err != nil {
			lumo.Error("Failed to fetch timetable from AniList: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error": "failed to fetch timetable: %v"}`, err)
			return
		}

		jsonData, err := json.Marshal(timetable)
		if err != nil {
			lumo.Error("Failed to marshal timetable: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Update cache
		_, err = db.Exec("INSERT OR REPLACE INTO anime_timetable (id, data, updated_at) VALUES (1, ?, ?)", string(jsonData), time.Now())
		if err != nil {
			lumo.Error("Failed to update anime_timetable cache: %v", err)
		}

		w.Write(jsonData)
	}
}

func triggerBackgroundRefresh(db *sql.DB) {
	refreshMutex.Lock()
	if isRefreshing || time.Since(lastRefreshTried) < 5*time.Minute {
		refreshMutex.Unlock()
		return
	}
	isRefreshing = true
	lastRefreshTried = time.Now()
	refreshMutex.Unlock()

	defer func() {
		refreshMutex.Lock()
		isRefreshing = false
		refreshMutex.Unlock()
	}()

	timetable, err := fetchTimetableFromAniList()
	if err != nil {
		lumo.Error("Background refresh failed: %v", err)
		return
	}

	jsonData, err := json.Marshal(timetable)
	if err != nil {
		lumo.Error("Failed to marshal timetable in background: %v", err)
		return
	}

	_, err = db.Exec("INSERT OR REPLACE INTO anime_timetable (id, data, updated_at) VALUES (1, ?, ?)", string(jsonData), time.Now())
	if err != nil {
		lumo.Error("Failed to update cache in background: %v", err)
	} else {
		lumo.Info("Background refresh of anime timetable successful.")
	}
}

func fetchTimetableFromAniList() (*model.Timetable, error) {
	now := time.Now().UTC()
	daysSinceMonday := int(now.Weekday()) - 1
	if daysSinceMonday < 0 {
		daysSinceMonday = 6 // Sunday
	}

	startOfWeek := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -daysSinceMonday)
	start := startOfWeek.Unix()
	end := start + (7 * 24 * 60 * 60)

	allAnime := make([]model.Anime, 0)
	page := 1

	for {
		lumo.Debug("Fetching page %d...", page)
		resp, err := fetchPageWithRetry(int(start), int(end), page)
		if err != nil {
			return nil, err
		}

		schedules := resp.Data.Page.AiringSchedules
		allAnime = append(allAnime, processSchedules(schedules)...)

		if !resp.Data.Page.PageInfo.HasNextPage || page >= 10 { // Safety cap at 10 pages
			break
		}
		
		page++
		// Sequential delay to be super safe with rate limits
		time.Sleep(500 * time.Millisecond)
	}

	lumo.Info("Successfully fetched %d total anime episodes for the week.", len(allAnime))

	return &model.Timetable{
		UpdatedAt: time.Now(),
		Anime:     allAnime,
	}, nil
}

func fetchPageWithRetry(start, end, page int) (*model.AniListResponse, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<attempt) * time.Second
			lumo.Debug("Retrying page %d (attempt %d/%d) in %v...", page, attempt+1, maxRetries, backoff)
			time.Sleep(backoff)
		}

		resp, err := fetchPage(start, end, page)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		if !isRateLimitError(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("failed after %d attempts: %v", maxRetries, lastErr)
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	return bytes.Contains([]byte(err.Error()), []byte("429"))
}

func fetchPage(start, end, page int) (*model.AniListResponse, error) {
	variables := map[string]interface{}{
		"airingAt_greater": start,
		"airingAt_lesser":  end,
		"page":             page,
	}

	payload := map[string]interface{}{
		"query":     timetableQuery,
		"variables": variables,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(aniListAPI, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AniList API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var aniResp model.AniListResponse
	if err := json.NewDecoder(resp.Body).Decode(&aniResp); err != nil {
		return nil, err
	}

	return &aniResp, nil
}

func processSchedules(schedules []model.AiringSchedule) []model.Anime {
	animeList := make([]model.Anime, 0, len(schedules))
	for _, s := range schedules {
		title := s.Media.Title.English
		if title == "" {
			title = s.Media.Title.Romaji
		}

		animeList = append(animeList, model.Anime{
			ID:           s.Media.ID,
			IDMal:        s.Media.IDMal,
			Title:        title,
			Image:        s.Media.CoverImage.Large,
			Color:        s.Media.CoverImage.Color,
			AirTime:      s.AiringAt,
			Episode:      s.Episode,
			Genres:       s.Media.Genres,
			AverageScore: s.Media.AverageScore,
			Description:  s.Media.Description,
		})
	}
	return animeList
}
