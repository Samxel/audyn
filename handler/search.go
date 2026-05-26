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

		qualityLabel, qualityCategory, qualityAudioFmt, bytesPerTrack := qualityForLevel(h.Config.DeezerQuality)

		title := fmt.Sprintf("%s - %s (%s) [%s]", xmlEscape(a.Artist.Name), xmlEscape(a.Title), year, qualityLabel)
		guid := fmt.Sprintf("audyn-deezer-%d", a.ID)
		downloadURL := fmt.Sprintf("http://%s/download/%d", r.Host, a.ID)
		estimatedSize := int64(a.NbTracks) * bytesPerTrack

		items += fmt.Sprintf(`
		<item>
			<title>%s</title>
			<guid>%s</guid>
			<pubDate>%s</pubDate>
			<enclosure url="%s" length="%d" type="application/x-nzb"/>
			<newznab:attr name="category" value="%s"/>
			<newznab:attr name="size" value="%d"/>
			<newznab:attr name="audioformat" value="%s"/>
		</item>`, title, guid, time.Now().Format(time.RFC1123Z), downloadURL, estimatedSize, qualityCategory, estimatedSize, qualityAudioFmt)
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

// 0 	 MP3 128 kbps  	category 3010  ~3.3 MB/track
// 1 	 MP3 320 kbps  	category 3010  ~8.3 MB/track
// 2	 FLAC         	category 3040  ~25 MB/track
func qualityForLevel(level int) (label, category, audioFmt string, bytesPerTrack int64) {
	switch level {
	case 0:
		return "MP3 128kbps", "3010", "MP3", 210 * 128 * 1000 / 8
	case 1:
		return "MP3 320kbps", "3010", "MP3", 210 * 320 * 1000 / 8
	default:
		return "FLAC", "3040", "FLAC", 25 * 1024 * 1024
	}
}
