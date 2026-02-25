# 🍱 Jubako (重箱)

**The portable, offline-first anime theater.**

Jubako is a self-contained anime streaming and downloading client written in Go. It packages a torrent engine, a metadata scraper, a database, and a modern web interface into a **single binary file**.

Use on any Windows/Linux powered device, take it to a cabin without internet, and watch your entire library instantly.

### ✨ Key Features

* **📦 Single Binary:** No installer. No dependencies. Everything (HTML/JS/CSS) is baked in.
* **🔌 Offline First:** Downloads episodes to your local disk. Once downloaded, the internet is optional.
* **🧠 Smart Resolve:** Automatically selects the best torrent (1080p, trusted groups like *SubsPlease* or *Erai-raws*) based on metadata.
* **🚅 Hybrid Playback:**
    * **Web Mode:** Streams directly to the browser with smart remuxing (no CPU-killing transcoding).
    * **Theater Mode:** One-click launch to MPV or VLC for perfect native playback.
* **🔎 Unified Search:** Fetches metadata from AniList and correlates it with Nyaa torrents automatically.

### ⚠️ Disclaimer & Technologies

> THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.

This software is an experimental project designed to explore:
* **Peer-to-Peer Networking:** Utilizing the BitTorrent protocol via `anacrolix/torrent`.
* **Web Scraping & Aggregation:** Interfacing with public metadata APIs (AniList) and search indexes.
* **Local Media Streaming:** Real-time video remuxing and browser-based playback.

**Important Notes:**
1.  **Content Agnostic Utility:** Jubako is a client-side utility, similar to a web browser or a standard BitTorrent client. The software itself does not host, index, or store any copyrighted content.
2.  **Protocol Behavior:** Usage of the BitTorrent protocol involves the simultaneous downloading and uploading of data chunks. By using this software, your device participates in a peer-to-peer swarm, which may involve distributing data to other users.
3.  **Third-Party Sources:** Metadata is provided by AniList. Search results are aggregated from public sources. The availability and legality of content depend on your local jurisdiction and the specific sources you choose to access.
4.  **User Responsibility:** The developer of Jubako is not responsible for the content accessed or the data transmitted by the user. You are solely responsible for ensuring your usage complies with local laws and copyright regulations.

*Use this tool responsibly and support the creators of the media you love.*

### 🗺️ Development Roadmap

Use this as your checklist during development.

#### **Phase 1: The Foundation (Backend)**
- [ ] **Project Skeleton:** Set up Go module and `main.go`.
- [ ] **Database Layer:** Initialize `modernc.org/sqlite` and design the schema (Series, Episodes, DownloadState).
- [ ] **Metadata Agent:** Implement AniList GraphQL client to fetch "Currently Airing" and search by Romaji title.
- [ ] **Torrent Engine:** Integrate `anacrolix/torrent`.
    - [ ] Ability to add Magnet URI.
    - [ ] Ability to prioritize the first 5% of a file (for instant streaming).
    - [ ] Save/Resume state handling.

#### **Phase 2: The Collector (Logic)**
- [ ] **Nyaa Scraper:** Build the search logic.
    - [ ] Regex filter for Trusted Groups (SubsPlease, etc.).
    - [ ] Sort by Seeders.
    - [ ] "Smart Pick" function that returns the single best Magnet URI.
- [ ] **Library Manager:** Logic to scan the download directory and sync it with the SQLite DB (detects if files were moved/deleted).

#### **Phase 3: The View (Frontend)**
- [ ] **Web Server:** Set up Go `http.FileServer` with `//go:embed`.
- [ ] **Search UI:** React/Vue page to search AniList and display results.
- [ ] **Library UI:** A grid view of downloaded anime (reading from local DB, not API).
- [ ] **Download Manager:** A simple progress bar list using WebSocket/SSE updates from Go.

#### **Phase 4: The Cinema (Playback)**
- [ ] **Stream Handler:** Create the HTTP endpoint that pipes the torrent reader to the response.
- [ ] **Remux Engine:** Implement the `ffmpeg -c copy` wrapper to convert MKV → MP4 on the fly.
- [ ] **Web Player:** Integrate Video.js + SubtitlesOctopus for browser playback.
- [ ] **Native Launcher:** Implement `os/exec` to detect and launch VLC/MPV with the local file path.

### 🛠️ Architecture

**Jubako** uses a "Monolith" architecture to ensure portability:

1.  **Core:** Go (Golang) handles all networking and file IO.
2.  **Data:** SQLite (Embedded) stores watch history and library index.
3.  **UI:** React/Vue compiled to static HTML/CSS, embedded into the binary via `go:embed`.
4.  **IO:** FFMpeg (for browser compat) or Direct File Access (for VLC/MPV).

### 🚀 Quick Start (Dev)

```bash
git clone https://github.com/amatsagu/jubako]
cd frontend && npm install && npm run build
cd .. && go run main.go