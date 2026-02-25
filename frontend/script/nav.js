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