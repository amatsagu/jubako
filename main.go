package main

import (
	"embed"
	"fmt"
	"jubako/internal/app"
	"jubako/internal/config"
	"os"
	"os/exec"

	"github.com/amatsagu/lumo"
)

//go:embed frontend
var embeddedFrontend embed.FS

func main() {
	lumo.EnableDebug()
	lumo.EnableStackOnWarns()

	app.NewApplication(embeddedFrontend).Run()
	lumo.Close()
}

// launchPlayer is a helper to launch external video players.
func launchPlayer(videoInfoHash string) {
	url := fmt.Sprintf("http://127.0.0.1:%s/stream?hash=%s", config.HTTP_PORT, videoInfoHash)
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
