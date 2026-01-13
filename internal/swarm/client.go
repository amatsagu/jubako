package swarm

import (
	"io"
	"jubako/internal/config"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/amatsagu/lumo"
	alog "github.com/anacrolix/log"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
)

type SwarmClient struct {
	client          *torrent.Client
	activeDownloads int

	// Registry tracks where files are located after they are "Dropped" from the client
	// Key: InfoHash (HexString), Value: Absolute Path on Disk
	readyFiles map[string]string
	mu         sync.RWMutex // Protects the map
}

type DownloadDetails struct {
	InfoHash           string
	Path               string
	PercentageProgress float64
	ActivePeers        int
}

func NewSwarmClient() *SwarmClient {
	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = filepath.Join(config.APP_FILES_PATH, "downloads")
	cfg.Debug = false
	cfg.DisableIPv6 = true // Fix "network unreachable" spam
	cfg.Logger = cfg.Logger.WithFilterLevel(alog.Disabled)

	c, err := torrent.NewClient(cfg)
	if err != nil {
		werr := lumo.WrapError(err)
		lumo.Panic("Failed to create SwarmClient: %v", werr)
	}

	return &SwarmClient{
		client:     c,
		readyFiles: make(map[string]string),
	}
}

func (s *SwarmClient) AddMagnet(magnet string, identifier string, callback func(data *DownloadDetails, err error)) {
	if identifier == "" {
		identifier = magnet
	}

	lumo.Debug("Added \"%s\" magnet to swarm queue.", identifier)
	t, err := s.client.AddMagnet(magnet)
	if err != nil {
		werr := lumo.WrapError(err)
		if identifier != magnet {
			werr.Include("identifier", identifier)
		}

		werr.Include("magnet", magnet)
		callback(nil, werr)
		return
	}

	go func() {
		select {
		case <-t.GotInfo():
			// lumo.Debug("Successfully obtained metadata for \"%s\" magnet.", identifier)

			s.mu.Lock()
			s.activeDownloads++
			s.mu.Unlock()
		case <-time.After(60 * time.Second):
			t.Drop()

			s.mu.Lock()
			s.activeDownloads--
			s.mu.Unlock()

			werr := lumo.WrapString("reached timeout for fetching metadata")
			if identifier != magnet {
				werr.Include("identifier", identifier)
			}

			werr.Include("magnet", magnet)
			callback(nil, werr)
			return
		}

		var target *torrent.File
		for _, f := range t.Files() {
			name := strings.ToLower(f.Path())
			if strings.HasSuffix(name, ".mkv") || strings.HasSuffix(name, ".mp4") {
				if target == nil || f.Length() > target.Length() {
					target = f
				}
			}
		}

		if target == nil {
			t.Drop()

			s.mu.Lock()
			s.activeDownloads--
			s.mu.Unlock()

			werr := lumo.WrapString("used magnet torrent points to no valid video files (.mkv or .mp4)")
			if identifier != magnet {
				werr.Include("identifier", identifier)
			}

			werr.Include("magnet", magnet)
			callback(nil, werr)
			return
		}

		lumo.Debug("Started downloading \"%s\" magnet to: %s", identifier, target.DisplayPath())
		target.Download()

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				got := target.BytesCompleted()
				total := target.Length()

				if total == 0 {
					continue
				}

				stats := t.Stats()
				details := DownloadDetails{
					InfoHash:           t.InfoHash().String(),
					Path:               target.DisplayPath(),
					PercentageProgress: (float64(got) / float64(total)) * 100,
					ActivePeers:        stats.ActivePeers,
				}

				callback(&details, nil)

				if got >= total {
					lumo.Debug("Successfully finished downloading \"%s\" magnet.", identifier)

					s.mu.Lock()
					s.activeDownloads--
					s.readyFiles[t.InfoHash().String()] = target.Path()
					s.mu.Unlock()
					details.PercentageProgress = 100
					callback(&details, nil)
					return
				}

			case <-t.Closed():
				s.mu.Lock()
				s.activeDownloads--
				s.mu.Unlock()

				werr := lumo.WrapString("torrent connection closed unexpectedly")
				if identifier != magnet {
					werr.Include("identifier", identifier)
				}

				werr.Include("magnet", magnet)
				callback(nil, werr)
				return
			}
		}
	}()
}

func (s *SwarmClient) CancelMagnet(magnet string) error {
	m, err := metainfo.ParseMagnetV2Uri(magnet)
	if err != nil {
		werr := lumo.WrapError(err)
		werr.Include("magnet", magnet)
		return werr
	}

	t, ok := s.client.Torrent(m.InfoHash.Value)
	if !ok {
		lumo.Debug("Attempted to cancel \"%s\" magnet, but it was not found in active magnets. Ignored.", magnet)
		return nil
	}

	lumo.Debug("Requested to cancel \"%s\" magnet.", t.Name())
	t.Drop()
	return nil
}

// StreamHandler smart-routes requests:
// - If torrent is active -> Streams from RAM/Network
// - If torrent is dropped -> Streams from Disk
func (s *SwarmClient) StreamHandler(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash") // We pass ?hash=... in URL
	if hash == "" {
		http.Error(w, "Missing hash param", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "video/x-matroska")

	s.mu.RLock()
	diskPath, exists := s.readyFiles[hash]
	s.mu.RUnlock()

	if exists {
		// Serve directly from disk (OS efficient)
		http.ServeFile(w, r, diskPath)
		return
	}

	hashInfo := metainfo.NewHashFromHex(hash)
	t, ok := s.client.Torrent(hashInfo)
	if !ok {
		http.Error(w, "Torrent not found (maybe queued but no metadata yet?)", 404)
		return
	}

	// Naive way to search for file, need change later.
	var target *torrent.File
	for _, f := range t.Files() {
		if strings.HasSuffix(f.Path(), ".mkv") || strings.HasSuffix(f.Path(), ".mp4") {
			target = f
			break
		}
	}

	if target != nil {
		reader := target.NewReader()
		reader.SetReadahead(20 * 1024 * 1024)
		defer reader.Close()
		io.Copy(w, reader)
	}
}
