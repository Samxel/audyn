package handler

import (
	"audyn/config"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"
	"unicode"
)

// JobStatus represents the lifecycle state of a download job.
type JobStatus string

const (
	StatusQueued      JobStatus = "queued"
	StatusDownloading JobStatus = "downloading"
	StatusComplete    JobStatus = "complete"
	StatusFailed      JobStatus = "failed"
)

type DownloadJob struct {
	ID          string
	AlbumID     string
	Title       string
	FolderName  string
	Status      JobStatus
	FilePath    string
	Error       string
	CreatedAt   time.Time
	CompletedAt time.Time
}

// JobQueue is a thread-safe store for all download jobs.
type JobQueue struct {
	mu   sync.RWMutex
	jobs map[string]*DownloadJob
}

// Queue is the singleton job store used by both the SABnzbd handler and the
// download goroutines.
var Queue = &JobQueue{
	jobs: make(map[string]*DownloadJob),
}

// Reset clears every job from the queue. Only intended for use in tests.
func (q *JobQueue) Reset() {
	q.mu.Lock()
	q.jobs = make(map[string]*DownloadJob)
	q.mu.Unlock()
}

var RunDownloadFunc = RunDownload

// newJobID returns a random 16-character hex string suitable as a job/nzo id.
func newJobID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (q *JobQueue) Add(albumID, title, folderName string) *DownloadJob {
	job := &DownloadJob{
		ID:         newJobID(),
		AlbumID:    albumID,
		Title:      title,
		FolderName: folderName,
		Status:     StatusQueued,
		CreatedAt:  time.Now(),
	}
	q.mu.Lock()
	q.jobs[job.ID] = job
	q.mu.Unlock()
	return job
}

func sanitizeForFS(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '/' || r == '\\' || r == ':' || r == '*' ||
			r == '?' || r == '"' || r == '<' || r == '>' || r == '|':
			b.WriteRune('-')
		case unicode.IsControl(r):
			// skip control characters
		default:
			b.WriteRune(r)
		}
	}
	result := strings.TrimSpace(b.String())
	runes := []rune(result)
	if len(runes) > 120 {
		runes = runes[:120]
	}
	return strings.TrimSpace(string(runes))
}

// Get returns the job with the given ID.
func (q *JobQueue) Get(id string) (*DownloadJob, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	j, ok := q.jobs[id]
	return j, ok
}

// Active returns all jobs that are queued or currently downloading.
func (q *JobQueue) Active() []*DownloadJob {
	q.mu.RLock()
	defer q.mu.RUnlock()
	var out []*DownloadJob
	for _, j := range q.jobs {
		if j.Status == StatusQueued || j.Status == StatusDownloading {
			out = append(out, j)
		}
	}
	return out
}

// Completed returns all jobs that have finished (successfully or with an error).
func (q *JobQueue) Completed() []*DownloadJob {
	q.mu.RLock()
	defer q.mu.RUnlock()
	var out []*DownloadJob
	for _, j := range q.jobs {
		if j.Status == StatusComplete || j.Status == StatusFailed {
			out = append(out, j)
		}
	}
	return out
}

func RunDownload(job *DownloadJob, cfg config.Config) {
	Queue.mu.Lock()
	job.Status = StatusDownloading
	Queue.mu.Unlock()

	slog.Info("Download started", "job_id", job.ID, "album_id", job.AlbumID)

	folderName := job.FolderName
	if folderName == "" {
		folderName = job.ID
	}
	jobFolder := path.Join(cfg.CompletePath, folderName)

	if err := os.MkdirAll(jobFolder, 0o755); err != nil {
		slog.Error("Failed to create job folder", "job_id", job.ID, "err", err)
		Queue.mu.Lock()
		job.Status = StatusFailed
		job.Error = "mkdir: " + err.Error()
		job.CompletedAt = time.Now()
		Queue.mu.Unlock()
		return
	}

	// rip --folder <dir> --no-db --no-progress url https://www.deezer.com/album/<id>
	cmd := exec.Command("rip",
		"--folder", jobFolder,
		"--no-db",
		"--no-progress",
		"url",
		"https://www.deezer.com/album/"+job.AlbumID,
	)

	out, err := cmd.CombinedOutput()
	slog.Info("streamrip output", "job_id", job.ID, "output", string(out))

	Queue.mu.Lock()
	defer Queue.mu.Unlock()
	job.CompletedAt = time.Now()

	if err != nil {
		slog.Error("Download failed", "job_id", job.ID, "err", err)
		job.Status = StatusFailed
		job.Error = err.Error()
		return
	}

	mappedPath := path.Join(cfg.CompletePathMapping, folderName)
	localPath := path.Join(cfg.CompletePath, folderName)

	entries, err := os.ReadDir(localPath)
	if err == nil && len(entries) > 0 {
		artistDir := path.Join(localPath, entries[0].Name())
		albumEntries, err := os.ReadDir(artistDir)
		if err == nil && len(albumEntries) > 0 {
			for _, e := range albumEntries {
				if e.IsDir() {
					mappedPath = path.Join(cfg.CompletePathMapping, job.ID, entries[0].Name(), e.Name())
					break
				}
			}
			if mappedPath == path.Join(cfg.CompletePathMapping, job.ID) {
				mappedPath = path.Join(cfg.CompletePathMapping, job.ID, entries[0].Name())
			}
		}
	}

	job.FilePath = mappedPath
	job.Status = StatusComplete
	slog.Info("Download complete", "job_id", job.ID, "path", job.FilePath)
}
