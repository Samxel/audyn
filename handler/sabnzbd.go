package handler

import (
	"audyn/config"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var nzbIDRe = regexp.MustCompile(`audyn-deezer-(\d+)@audyn`)

type SabnzbdHandler struct {
	Config config.Config
}

func (h *SabnzbdHandler) Serve(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	slog.Info("SABnzbd request", "mode", mode, "path", r.URL.Path)

	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path == "/download/config/categories/" {
		json.NewEncoder(w).Encode(map[string]any{
			"categories": []string{"*", "lidarr", "music"},
		})
		return
	}

	switch mode {
	// ------------------------------------------------------------------ meta
	case "version":
		json.NewEncoder(w).Encode(map[string]string{
			"version": "3.7.0",
		})

	case "get_config":
		json.NewEncoder(w).Encode(map[string]any{
			"config": map[string]any{
				"misc": map[string]any{
					"complete_dir": h.Config.CompletePathMapping,
				},
				"categories": []map[string]any{
					{"name": "lidarr", "dir": h.Config.CompletePathMapping, "newzbin": "", "priority": 0, "pp": ""},
					{"name": "music", "dir": h.Config.CompletePathMapping, "newzbin": "", "priority": 0, "pp": ""},
				},
			},
		})

	case "fullstatus":
		json.NewEncoder(w).Encode(map[string]any{
			"status": map[string]any{
				"complete_dir": h.Config.CompletePathMapping,
				"paused":       false,
				"categories":   []string{"*", "lidarr", "music"},
			},
		})

	case "get_categories":
		json.NewEncoder(w).Encode(map[string]any{
			"categories": []string{"*", "lidarr", "music"},
		})

	// ----------------------------------------------------------- add a job
	case "addfile":
		h.handleAddFile(w, r)

	// ---------------------------------------------------------- active jobs
	case "queue":
		if r.URL.Query().Get("name") == "delete" {
			h.handleDeleteJob(w, r)
			return
		}
		active := Queue.Active()
		slots := make([]any, 0, len(active))
		for i, j := range active {
			status := "Downloading"
			if j.Status == StatusQueued {
				status = "Queued"
			}
			slots = append(slots, map[string]any{
				"nzo_id":        j.ID,
				"status":        status,
				"index":         i,
				"filename":      j.Title,
				"cat":           "music",
				"mb":            "150",
				"mbleft":        "75",
				"size":          "150 MB",
				"sizeleft":      "75 MB",
				"percentage":    "50",
				"avg_age":       "0d",
				"script":        "",
				"missing":       0,
				"direct_unpack": 0,
				"mbmissing":     "0",
				"labels":        []string{},
				"priority":      "Normal",
				"unpackopts":    "3",
			})
		}
		json.NewEncoder(w).Encode(map[string]any{
			"queue": map[string]any{
				"status":     "Idle",
				"slots":      slots,
				"paused":     false,
				"speedlimit": "0",
				"noofslots":  len(active),
			},
		})

	// ------------------------------------------------------- completed jobs
	case "history":
		if r.URL.Query().Get("name") == "delete" {
			h.handleDeleteJob(w, r)
			return
		}
		completed := Queue.Completed()
		slots := make([]any, 0, len(completed))
		for _, j := range completed {
			status := "Completed"
			failMsg := ""
			if j.Status == StatusFailed {
				status = "Failed"
				failMsg = j.Error
			}
			slots = append(slots, map[string]any{
				"nzo_id":       j.ID,
				"status":       status,
				"name":         j.Title,
				"storage":      j.FilePath,
				"category":     "music",
				"size":         "150 MB",
				"bytes":        int64(157286400),
				"completed":    j.CompletedAt.Unix(),
				"downloaded":   int64(157286400),
				"fail_message": failMsg,
				"action_line":  "",
				"script_log":   "",
				"script_line":  "",
			})
		}
		json.NewEncoder(w).Encode(map[string]any{
			"history": map[string]any{
				"slots":     slots,
				"noofslots": len(slots),
			},
		})

	default:
		slog.Warn("SABnzbd unknown mode", "mode", mode, "path", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "not implemented",
		})
	}
}

// handleAddFile processes mode=addurl from Lidarr.
// Lidarr sends: url=http://audyn:5000/download/<albumID>&name=<release>&cat=music
func (h *SabnzbdHandler) handleAddFile(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	name := r.FormValue("nzbname")
	if name == "" {
		name = r.FormValue("name")
	}

	albumID := ""

	file, _, err := r.FormFile("name")
	if err == nil {
		defer file.Close()
		var buf strings.Builder
		io.Copy(&buf, file)

		if matches := nzbIDRe.FindStringSubmatch(buf.String()); len(matches) > 1 {
			albumID = matches[1]
		}
	}

	if albumID == "" {
		slog.Warn("addfile: could not extract album ID from NZB")
		json.NewEncoder(w).Encode(map[string]any{"status": false, "error": "could not extract album ID"})
		return
	}

	// Build a human-readable title and folder name from the Deezer API.
	// Fall back to generic names if the lookup fails (no ARL needed for metadata).
	title := name
	folderName := "deezer-" + albumID
	if numID, err := strconv.Atoi(albumID); err == nil {
		if detail, err := GetAlbumDetail(numID); err == nil && detail.Title != "" {
			year := ""
			if len(detail.ReleaseDate) >= 4 {
				year = " (" + detail.ReleaseDate[:4] + ")"
			}
			rawTitle := detail.Artist.Name + " - " + detail.Title + year
			title = rawTitle
			folderName = sanitizeForFS(rawTitle)
		}
	}
	if title == "" || strings.HasPrefix(title, "deezer-") || strings.HasPrefix(title, "audyn-deezer-") {
		title = folderName
	}

	job := Queue.Add(albumID, title, folderName)
	go RunDownloadFunc(job, h.Config)

	slog.Info("addfile: job created", "job_id", job.ID, "album_id", albumID, "title", title, "folder", folderName)

	json.NewEncoder(w).Encode(map[string]any{
		"status":  true,
		"nzo_ids": []string{job.ID},
	})
}

func (h *SabnzbdHandler) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("value")
	if id == "" {
		json.NewEncoder(w).Encode(map[string]any{"status": false, "error": "missing value"})
		return
	}

	if job, ok := Queue.Get(id); ok && r.URL.Query().Get("del_files") == "1" {
		folderName := job.FolderName
		if folderName == "" {
			folderName = job.ID
		}
		localFolder := path.Join(h.Config.CompletePath, folderName)
		if err := os.RemoveAll(localFolder); err != nil {
			slog.Warn("Could not delete download folder", "job_id", id, "folder", localFolder, "err", err)
		} else {
			slog.Info("Deleted download folder", "job_id", id, "folder", localFolder)
		}
	}

	Queue.Remove(id)
	slog.Info("Deleted job from queue", "job_id", id)
	json.NewEncoder(w).Encode(map[string]any{"status": true})
}

// extractAlbumID pulls the Deezer album ID out of a download URL.
// Expected format: http://host/download/{numeric-id}
func extractAlbumID(downloadURL string) string {
	// Strip trailing slash then take the last path segment
	trimmed := strings.TrimRight(downloadURL, "/")
	idx := strings.LastIndex(trimmed, "/")
	if idx < 0 {
		return ""
	}
	last := trimmed[idx+1:]
	// Must be all digits to be a valid Deezer ID
	if last == "" {
		return ""
	}
	for _, c := range last {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return last
}
