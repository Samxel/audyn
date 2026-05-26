package handler

import (
	"audyn/config"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"
)

// JobStatus represents the lifecycle state of a download job.
type JobStatus string

const (
	StatusQueued      JobStatus = "queued"
	StatusDownloading JobStatus = "downloading"
	StatusComplete    JobStatus = "complete"
	StatusFailed      JobStatus = "failed"
)

// DownloadJob tracks a single Deezer album download driven by streamrip.
type DownloadJob struct {
	ID      string
	AlbumID string
	Title   string
	Status  JobStatus
	// FilePath is the directory Lidarr should import from (mapped path).
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

// Add creates a new job for the given Deezer album ID, stores it and returns it.
func (q *JobQueue) Add(albumID, title string) *DownloadJob {
	job := &DownloadJob{
		ID:        newJobID(),
		AlbumID:   albumID,
		Title:     title,
		Status:    StatusQueued,
		CreatedAt: time.Now(),
	}
	q.mu.Lock()
	q.jobs[job.ID] = job
	q.mu.Unlock()
	return job
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

// RunDownload invokes streamrip to download a Deezer album and updates the job
// status accordingly.  It is intended to be called in a goroutine:
//
//	go RunDownload(job, cfg)
func RunDownload(job *DownloadJob, cfg config.Config) {
	Queue.mu.Lock()
	job.Status = StatusDownloading
	Queue.mu.Unlock()

	slog.Info("Download started", "job_id", job.ID, "album_id", job.AlbumID)

	// Each job gets its own sub-directory so the storage path reported back to
	// Lidarr is unambiguous even when multiple downloads run concurrently.
	jobFolder := path.Join(cfg.CompletePath, job.ID)

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

	mappedPath := path.Join(cfg.CompletePathMapping, job.ID)
	localPath := path.Join(cfg.CompletePath, job.ID)

	entries, err := os.ReadDir(localPath)
	if err == nil && len(entries) > 0 {
		artistDir := path.Join(localPath, entries[0].Name())
		albumEntries, err := os.ReadDir(artistDir)
		if err == nil && len(albumEntries) > 0 {
			mappedPath = path.Join(cfg.CompletePathMapping, job.ID, entries[0].Name(), albumEntries[0].Name())
		}
	}

	job.FilePath = mappedPath
	job.Status = StatusComplete
	slog.Info("Download complete", "job_id", job.ID, "path", job.FilePath)
}
