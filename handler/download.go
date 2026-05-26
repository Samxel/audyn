package handler

import (
	"audyn/config"
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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

	if err := os.MkdirAll(jobFolder, 0o777); err != nil {
		slog.Error("Failed to create job folder", "job_id", job.ID, "err", err)
		Queue.mu.Lock()
		job.Status = StatusFailed
		job.Error = "mkdir: " + err.Error()
		job.CompletedAt = time.Now()
		Queue.mu.Unlock()
		return
	}

	cmd := exec.Command("rip",
		"--folder", jobFolder,
		"--no-db",
		"--no-progress",
		"url",
		"https://www.deezer.com/album/"+job.AlbumID,
	)

	cmd.Env = append(os.Environ(), "COLUMNS=10000", "NO_COLOR=1")
	pr, pw, err := os.Pipe()
	if err != nil {
		slog.Warn("os.Pipe failed, falling back to CombinedOutput", "err", err)
		out, cmdErr := cmd.CombinedOutput()
		slog.Info("streamrip output", "job_id", job.ID, "output", string(out))
		finalizeDownload(job, cfg, folderName, cmdErr)
		return
	}
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		Queue.mu.Lock()
		job.Status = StatusFailed
		job.Error = "start: " + err.Error()
		job.CompletedAt = time.Now()
		Queue.mu.Unlock()
		return
	}

	pw.Close()

	var outBuf bytes.Buffer
	createdDirs := map[string]bool{}
	pathPrefix := cfg.CompletePath + "/"
	scanner := bufio.NewScanner(pr)

	for scanner.Scan() {
		line := scanner.Text()
		outBuf.WriteString(line + "\n")
		ensureAudioDir(line, pathPrefix, createdDirs)
	}

	cmdErr := cmd.Wait()
	slog.Info("streamrip output", "job_id", job.ID, "output", outBuf.String())

	finalizeDownload(job, cfg, folderName, cmdErr)
}

func ensureAudioDir(line, pathPrefix string, made map[string]bool) {
	audioExts := []string{".flac", ".mp3", ".opus", ".ogg", ".m4a", ".wav"}
	for {
		idx := strings.Index(line, "'"+pathPrefix)
		if idx < 0 {
			return
		}
		after := line[idx+1:]
		end := strings.IndexByte(after, '\'')
		if end < 0 {
			return
		}
		filePath := after[:end]
		lower := strings.ToLower(filePath)
		for _, ext := range audioExts {
			if strings.HasSuffix(lower, ext) {
				dir := path.Dir(filePath)
				if !made[dir] {
					if mkErr := os.MkdirAll(dir, 0o755); mkErr == nil {
						slog.Info("Pre-created missing audio directory", "dir", dir)
						made[dir] = true
					} else {
						slog.Warn("Failed to pre-create audio directory", "dir", dir, "err", mkErr)
					}
				}
				break
			}
		}
		line = after[end+1:] // advance past this path and keep scanning
	}
}

var errAudioFound = errors.New("audio file found")

func hasAudioFiles(root string) bool {
	audioExts := map[string]bool{
		".flac": true, ".mp3": true, ".opus": true,
		".ogg": true, ".m4a": true, ".wav": true,
	}
	err := filepath.WalkDir(root, func(_ string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		if audioExts[strings.ToLower(filepath.Ext(d.Name()))] {
			return errAudioFound
		}
		return nil
	})
	return errors.Is(err, errAudioFound)
}

func finalizeDownload(job *DownloadJob, cfg config.Config, folderName string, cmdErr error) {
	localPath := path.Join(cfg.CompletePath, folderName)
	mappedPath := path.Join(cfg.CompletePathMapping, folderName)

	Queue.mu.Lock()
	defer Queue.mu.Unlock()
	job.CompletedAt = time.Now()

	if cmdErr != nil {
		slog.Error("Download failed", "job_id", job.ID, "err", cmdErr)
		job.Status = StatusFailed
		job.Error = cmdErr.Error()
		return
	}

	if !hasAudioFiles(localPath) {
		slog.Error("No audio files found after download", "job_id", job.ID, "path", localPath)
		job.Status = StatusFailed
		job.Error = "no audio files found after download"
		return
	}

	if entries, err := os.ReadDir(localPath); err == nil && len(entries) > 0 {
		first := entries[0].Name()
		artistDir := path.Join(localPath, first)
		if albumEntries, err := os.ReadDir(artistDir); err == nil && len(albumEntries) > 0 {
			for _, e := range albumEntries {
				if e.IsDir() {
					mappedPath = path.Join(cfg.CompletePathMapping, folderName, first, e.Name())
					break
				}
			}
			if mappedPath == path.Join(cfg.CompletePathMapping, folderName) {
				mappedPath = path.Join(cfg.CompletePathMapping, folderName, first)
			}
		}
	}

	job.FilePath = mappedPath
	job.Status = StatusComplete
	slog.Info("Download complete", "job_id", job.ID, "path", job.FilePath)
}
