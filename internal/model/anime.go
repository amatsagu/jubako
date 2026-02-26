package model

import "time"

// AniList response structures
type AniListResponse struct {
	Data struct {
		Page struct {
			PageInfo struct {
				Total       int  `json:"total"`
				PerPage     int  `json:"perPage"`
				CurrentPage int  `json:"currentPage"`
				LastPage    int  `json:"lastPage"`
				HasNextPage bool `json:"hasNextPage"`
			} `json:"pageInfo"`
			AiringSchedules []AiringSchedule `json:"airingSchedules"`
		} `json:"Page"`
	} `json:"data"`
}

type AiringSchedule struct {
	ID       int   `json:"id"`
	AiringAt int64 `json:"airingAt"`
	Episode  int   `json:"episode"`
	Media    Media `json:"media"`
}

type Media struct {
	ID    int `json:"id"`
	IDMal int `json:"idMal"`
	Title struct {
		Romaji  string `json:"romaji"`
		English string `json:"english"`
		Native  string `json:"native"`
	} `json:"title"`
	CoverImage struct {
		Large string `json:"large"`
		Color string `json:"color"`
	} `json:"coverImage"`
	Description  string   `json:"description"`
	Genres       []string `json:"genres"`
	AverageScore int      `json:"averageScore"`
}

// Application-level structures
type Timetable struct {
	UpdatedAt time.Time `json:"updated_at"`
	Anime     []Anime   `json:"anime"`
}

type Anime struct {
	ID           int      `json:"id"`
	IDMal        int      `json:"id_mal"`
	Title        string   `json:"title"`
	Image        string   `json:"image"`
	Color        string   `json:"color"`
	AirTime      int64    `json:"air_time"`
	Episode      int      `json:"episode"`
	Genres       []string `json:"genres"`
	AverageScore int      `json:"average_score"`
	Description  string   `json:"description"`
}
