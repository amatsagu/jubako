package main

import (
	"fmt"
	"jubako/internal/app"
	"jubako/internal/config"
	"jubako/internal/swarm"
	"os"
	"os/exec"
	"time"

	"github.com/amatsagu/lumo"
)

func main() {
	lumo.EnableDebug()
	lumo.EnableStackOnWarns()

	app := app.NewApplication()
	go func() {
		lumo.Warn("Sleeping for 5s...")
		time.Sleep(time.Second * 5)

		launchedPlayer := false
		app.SwarmClient.AddMagnet(
			"magnet:?xt=urn:btih:d0612af2d527b880702a06df5801adf0b4fd5f02&dn=%5BToonsHub%5D%20There%20Was%20a%20Cute%20Girl%20in%20the%20Heros%20Party%20So%20I%20Tried%20Confessing%20to%20Her%20S01E02%201080p%20CR%20WEB-DL%20AAC2.0%20H.264%20%28Yuusha%20Party%20ni%20Kawaii%20Ko%20ga%20Ita%20no%20de%2C%20Kokuhaku%20Shitemita.%2C%20Multi-Subs%29&tr=http%3A%2F%2Fnyaa.tracker.wf%3A7777%2Fannounce&tr=udp%3A%2F%2Fopen.stealth.si%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337%2Fannounce&tr=udp%3A%2F%2Fexodus.desync.com%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.torrent.eu.org%3A451%2Fannounce",
			"",
			func(data *swarm.DownloadDetails, err error) {
				if err != nil {
					lumo.Panic("%v", err)
				}

				lumo.Info("Downloading... %.1f%% (%d peers)", data.PercentageProgress, data.ActivePeers)

				if !launchedPlayer && data.PercentageProgress >= 5 {
					launchedPlayer = true
					go launchPlayer(data.InfoHash)
				}
			},
		)

	}()

	app.Run()
	lumo.Close()
}

func launchPlayer(videoInfoHash string) {
	url := fmt.Sprintf("http://localhost:%s/stream?hash=%s", config.HTTP_PORT, videoInfoHash)
	players := []struct {
		Name string
		Cmd  string
		Args []string
	}{
		{
			"MPV", "mpv",
			[]string{
				"--fs",
				"--force-window=immediate",
				url, // Passing URL instead of file path
			},
		},
		{
			"Haruna", "haruna",
			[]string{url},
		},
		{
			"VLC", "vlc",
			[]string{"--fullscreen", url},
		},
	}

	for _, p := range players {
		_, err := exec.LookPath(p.Cmd)
		if err == nil {
			lumo.Info("Found %s as available video player.", p.Name)
			cmd := exec.Command(p.Cmd, p.Args...)
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				lumo.Warn("Detected that %s was closed.", p.Name)
			}
			return
		}
	}
}
