package handler

import (
	"audyn/config"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

type SearchHandler struct {
	Config config.Config
}

func xmlEscape(s string) string {
	var buf strings.Builder
	xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

func (h *SearchHandler) Serve(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	artist := r.URL.Query().Get("artist")
	album := r.URL.Query().Get("album")

	query := q
	if artist != "" {
		query += " " + artist
	}
	if album != "" {
		query += " " + album
	}
	if query == "" {
		query = "top"
	}

	slog.Info("Search requested", "query", query)

	albums, err := SearchDeezer(query)
	if err == nil {
		var wg sync.WaitGroup
		for i := range albums {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				detail, err := GetAlbumDetail(albums[i].ID)
				if err == nil {
					albums[i].ReleaseDate = detail.ReleaseDate
					albums[i].NbTracks = detail.NbTracks
				}
			}(i)
		}
		wg.Wait()
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")

	if err != nil {
		slog.Error("Deezer search failed", "err", err)
		fmt.Fprint(w, emptyResponse())
		return
	}

	slog.Info("Deezer results", "count", len(albums))

	items := ""
	for _, a := range albums {
		year := "0000"
		if len(a.ReleaseDate) >= 4 {
			year = a.ReleaseDate[:4]
		}

		// TODO: adjust depending on user profile & adjust when estimating size
		quality := "MP3 128kbps"

		title := fmt.Sprintf("%s - %s (%s) [%s]", xmlEscape(a.Artist.Name), xmlEscape(a.Title), year, quality)
		guid := fmt.Sprintf("audyn-deezer-%d", a.ID)
		downloadURL := fmt.Sprintf("http://%s/download/%d", r.Host, a.ID)
		estimatedSize := int64(a.NbTracks) * 210 * 128000 / 8 // TODO: change 128000 depending on quality

		items += fmt.Sprintf(`
		<item>
			<title>%s</title>
			<guid>%s</guid>
			<pubDate>%s</pubDate>
			<enclosure url="%s" length="150000000" type="application/x-nzb"/>
			<newznab:attr name="category" value="3010"/>
			<newznab:attr name="size" value="%d"/>
		</item>`, title, guid, time.Now().Format(time.RFC1123Z), downloadURL, estimatedSize)
	}

	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
	<rss version="2.0" xmlns:newznab="http://www.newznab.com/DTD/2010/feeds/attributes/">
	<channel>
		<title>Audyn</title>
		%s
	</channel>
	</rss>`, items)
}

func emptyResponse() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
	<rss version="2.0" xmlns:newznab="http://www.newznab.com/DTD/2010/feeds/attributes/">
	<channel><title>Audyn</title></channel>
	</rss>`
}
