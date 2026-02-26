const searchInput = document.getElementById("nav-search-input");

document.addEventListener("keydown", (e) => {
    // Check for Ctrl (or Cmd on Mac) + K
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        searchInput.focus();
    }
});

searchInput.addEventListener("keydown", async (e) => {
    if (e.key === "Enter") {
        const query = searchInput.value.trim();
        if (!query || query === "") return;

        const params = new URLSearchParams({ "query": query });
        const url = `/api/search?${params.toString()}`;

        try {
            console.log(`Fetching: ${url}`);
            const response = await fetch(url, {
                method: "GET",
                headers: {
                    "Accept": "application/json"
                }
            });

            if (response.ok) {
                const data = await response.json();
                console.log("Results received:", data);
            }
        } catch (err) {
            console.error("API connection error:", err);
        }
    }
});

(async () => {
    try {
        const response = await fetch("/api/anime-timetable", {
            method: "GET",
            headers: {
                "Accept": "application/json"
            }
        });

        if (!response.ok) {
            console.error("Failed to fetch timetable:", response.statusText);
            return;
        }

        const data = await response.json();
        console.log("Timetable received data:", data);

        if (!data || typeof data !== 'object') {
            console.error("Invalid data received from API:", data);
            return;
        }

        const animeData = data.anime || [];
        console.log(`Processing ${animeData.length} anime entries.`);

        // Clear existing timetable if it exists
        const existingContainer = document.getElementById("timetable-container");
        if (existingContainer) {
            existingContainer.remove();
        }

        const container = document.createElement("div");
        container.id = "timetable-container";
        container.style.padding = "20px";

        const title = document.createElement("h1");
        title.textContent = "Anime Timetable";
        container.appendChild(title);

        const dayNames = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];
        const currentDayName = new Date().toLocaleDateString('en-US', { weekday: 'long' });

        // Group anime by day
        const groupedAnime = {};
        dayNames.forEach(day => groupedAnime[day] = []);

        animeData.forEach(anime => {
            if (!anime || !anime.air_time) return;
            const date = new Date(anime.air_time * 1000);
            const dayName = date.toLocaleDateString('en-US', { weekday: 'long' });
            if (groupedAnime[dayName]) {
                groupedAnime[dayName].push(anime);
            }
        });

        dayNames.forEach(dayName => {
            const daySection = document.createElement("section");
            daySection.style.marginBottom = "20px";
            daySection.style.padding = "10px";
            
            if (dayName === currentDayName) {
                daySection.style.border = "2px solid #ff4500";
                daySection.style.backgroundColor = "rgba(255, 69, 0, 0.1)";
                daySection.style.borderRadius = "8px";
            }

            const dayTitle = document.createElement("h2");
            dayTitle.textContent = dayName + (dayName === currentDayName ? " (Today)" : "");
            daySection.appendChild(dayTitle);

            const animeList = groupedAnime[dayName];
            if (!animeList || animeList.length === 0) {
                const noAnime = document.createElement("p");
                noAnime.textContent = "No anime scheduled for this day.";
                daySection.appendChild(noAnime);
            } else {
                // Sort by time
                animeList.sort((a, b) => a.air_time - b.air_time);

                const ul = document.createElement("ul");
                animeList.forEach(anime => {
                    const item = document.createElement("li");
                    const time = new Date(anime.air_time * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
                    item.innerHTML = `<strong>${time}</strong> - ${anime.title} (Episode ${anime.episode})`;
                    ul.appendChild(item);
                });
                daySection.appendChild(ul);
            }

            container.appendChild(daySection);
        });

        document.body.appendChild(container);
    } catch (err) {
        console.error("Error fetching or rendering timetable:", err);
    }
})();