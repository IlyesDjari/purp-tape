package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/IlyesDjari/purp-tape/backend/internal/db"
	"github.com/IlyesDjari/purp-tape/backend/internal/finops"
	"github.com/IlyesDjari/purp-tape/backend/internal/storage"
)

// JobProcessor handles background jobs.
type JobProcessor struct {
	db                *db.Database
	r2                *storage.R2Client
	log               *slog.Logger
	workerConcurrency int
	batchSize         int
}

// NewJobProcessor creates new job processor
func NewJobProcessor(database *db.Database, r2Client *storage.R2Client, log *slog.Logger, workerConcurrency, batchSize int) *JobProcessor {
	if workerConcurrency <= 0 {
		workerConcurrency = 1
	}
	if batchSize <= 0 {
		batchSize = 10
	}
	return &JobProcessor{
		db:                database,
		r2:                r2Client,
		log:               log,
		workerConcurrency: workerConcurrency,
		batchSize:         batchSize,
	}
}

// ProcessPendingJobs runs background jobs (call this in a goroutine/cron)
func (jp *JobProcessor) ProcessPendingJobs(ctx context.Context) {
	jp.log.Info("starting background job processor",
		"worker_concurrency", jp.workerConcurrency,
		"batch_size", jp.batchSize)
	idleSleep := 5 * time.Second
	const maxIdleSleep = 60 * time.Second

	for {
		select {
		case <-ctx.Done():
			jp.log.Info("background job processor stopped")
			return
		default:
			// Process one batch of jobs
			processed, err := jp.processBatch(ctx)
			if err != nil {
				jp.log.Error("error processing job batch", "error", err)
				time.Sleep(idleSleep)
				continue
			}

			if processed == 0 {
				time.Sleep(idleSleep)
				if idleSleep < maxIdleSleep {
					idleSleep *= 2
					if idleSleep > maxIdleSleep {
						idleSleep = maxIdleSleep
					}
				}
				continue
			}

			idleSleep = 5 * time.Second
		}
	}
}

// processBatch gets pending jobs and processes them
func (jp *JobProcessor) processBatch(ctx context.Context) (int, error) {
	jobs, err := jp.db.ClaimPendingJobs(ctx, jp.batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to claim pending jobs: %w", err)
	}

	if len(jobs) == 0 {
		return 0, nil
	}

	workers := jp.workerConcurrency
	if len(jobs) < workers {
		workers = len(jobs)
	}

	jobCh := make(chan *db.JobData, len(jobs))
	for _, job := range jobs {
		jobCh <- job
	}
	close(jobCh)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				if err := jp.processJob(ctx, job); err != nil {
					jp.log.Error("failed to process job", "job_id", job.ID, "error", err)
				}
			}
		}()
	}

	wg.Wait()
	return len(jobs), nil
}

// processJob handles individual job based on type.
func (jp *JobProcessor) processJob(ctx context.Context, job *db.JobData) error {
	jp.log.Info("processing job", "job_id", job.ID, "job_type", job.JobType)

	var result interface{}
	var err error

	if isExpensiveFinOpsJob(job.JobType) {
		decision, guardErr := finops.EvaluateExpensiveJobGuard(ctx, jp.db)
		if guardErr != nil {
			jp.log.Warn("finops guard check failed; continuing job execution", "error", guardErr)
		} else if decision.Block {
			snapshot := decision.Snapshot
			result = map[string]interface{}{
				"status":                   "skipped_budget_guard",
				"reason":                   decision.Reason,
				"budget_utilization_ratio": decision.UtilizationRatio,
				"estimated_monthly_usd":    snapshot.EstimatedMonthlyCostUSD,
				"actual_monthly_usd":       snapshot.ActualMonthlyCostUSD,
				"governing_monthly_usd":    snapshot.GoverningMonthlyCostUSD,
				"pending_cleanup_jobs":     snapshot.PendingCleanupJobs,
				"pending_cleanup_bytes":    snapshot.PendingCleanupBytes,
				"guarded_at":               time.Now().UTC().Format(time.RFC3339),
			}
			jp.log.Warn("skipping expensive job due to FinOps budget guard",
				"job_id", job.ID,
				"job_type", job.JobType,
				"reason", decision.Reason,
				"budget_utilization_ratio", decision.UtilizationRatio)

			resultJSON, _ := json.Marshal(result)
			return jp.db.MarkJobCompleted(ctx, job.ID, resultJSON)
		}
	}

	switch job.JobType {
	case "cleanup_r2_file":
		result, err = jp.cleanupR2File(ctx, job.Data)
	case "convert_video_to_audio":
		result, err = jp.convertVideoToAudio(ctx, job.Data)
	case "generate_waveform":
		result, err = jp.generateWaveform(ctx, job.Data)
	default:
		err = fmt.Errorf("unknown job type: %s", job.JobType)
	}

	// Update job status
	if err != nil {
		jp.log.Error("job failed", "job_id", job.ID, "error", err)
		return jp.db.MarkJobFailed(ctx, job.ID, err.Error())
	}

	jp.log.Info("job completed successfully", "job_id", job.ID)
	resultJSON, _ := json.Marshal(result)
	return jp.db.MarkJobCompleted(ctx, job.ID, resultJSON)
}

func isExpensiveFinOpsJob(jobType string) bool {
	switch jobType {
	case "convert_video_to_audio", "generate_waveform":
		return true
	default:
		return false
	}
}

// cleanupR2File removes files from R2 storage.
func (jp *JobProcessor) cleanupR2File(ctx context.Context, jobDataJSON json.RawMessage) (interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(jobDataJSON, &data); err != nil {
		return nil, fmt.Errorf("failed to parse job data: %w", err)
	}

	r2ObjectKey, ok := data["r2_object_key"].(string)
	if !ok || r2ObjectKey == "" {
		return nil, fmt.Errorf("missing r2_object_key in job data")
	}

	// Validate key to prevent accidental deletion
	if len(r2ObjectKey) < 10 || len(r2ObjectKey) > 1024 {
		return nil, fmt.Errorf("invalid r2_object_key length")
	}

	jp.log.Info("deleting file from R2", "r2_key", r2ObjectKey)

	// Delete from R2
	if err := jp.r2.DeleteFile(ctx, r2ObjectKey); err != nil {
		jp.log.Error("failed to delete file from R2", "error", err, "r2_key", r2ObjectKey)
		return nil, fmt.Errorf("failed to delete file from R2: %w", err)
	}

	fileSizeBytes := int64(0)
	if fileSize, ok := data["file_size"].(float64); ok {
		fileSizeBytes = int64(fileSize)
	}

	jp.log.Info("file deleted from R2 successfully",
		"r2_key", r2ObjectKey,
		"file_size_bytes", fileSizeBytes)

	return map[string]interface{}{
		"r2_object_key":   r2ObjectKey,
		"deleted_at":      time.Now().Unix(),
		"file_size_bytes": fileSizeBytes,
	}, nil
}

// convertVideoToAudio extracts audio from video (requires FFmpeg on server)
func (jp *JobProcessor) convertVideoToAudio(ctx context.Context, jobDataJSON json.RawMessage) (interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(jobDataJSON, &data); err != nil {
		return nil, fmt.Errorf("failed to parse job data: %w", err)
	}

	videoR2Key, ok := data["video_r2_key"].(string)
	if !ok {
		return nil, fmt.Errorf("missing video_r2_key in job data")
	}

	trackVersionID, ok := data["track_version_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing track_version_id in job data")
	}
	trackVersionID = strings.TrimSpace(trackVersionID)
	if trackVersionID == "" {
		return nil, fmt.Errorf("track_version_id is empty")
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg not available on host: %w", err)
	}

	jp.log.Info("converting video to audio", "video_key", videoR2Key, "track_version_id", trackVersionID)

	workDir, err := os.MkdirTemp("", "purptape-convert-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create conversion temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	inputPath := filepath.Join(workDir, "input_video")
	inputFile, err := os.Create(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %w", err)
	}

	if _, err := jp.r2.DownloadFileToWriter(ctx, videoR2Key, inputFile); err != nil {
		inputFile.Close()
		return nil, fmt.Errorf("failed to download source video: %w", err)
	}
	if err := inputFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize temp input file: %w", err)
	}

	outputPath := filepath.Join(workDir, "output_audio.mp3")
	ffmpegCmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", inputPath, "-vn", "-acodec", "libmp3lame", "-b:a", "192k", outputPath)
	if output, err := ffmpegCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg video conversion failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	audioR2Key := fmt.Sprintf("processed/audio/%s/%d.mp3", trackVersionID, time.Now().UTC().Unix())
	outputFile, err := os.Open(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open converted audio: %w", err)
	}
	defer outputFile.Close()

	uploadResult, err := jp.r2.UploadFile(ctx, audioR2Key, outputFile, "audio/mpeg")
	if err != nil {
		return nil, fmt.Errorf("failed to upload converted audio: %w", err)
	}

	return map[string]interface{}{
		"status":              "completed",
		"track_version_id":    trackVersionID,
		"source_video_key":    videoR2Key,
		"converted_audio_key": uploadResult.Key,
		"file_size_bytes":     uploadResult.FileSize,
		"checksum":            uploadResult.Checksum,
	}, nil
}

// generateWaveform creates visual waveform data
func (jp *JobProcessor) generateWaveform(ctx context.Context, jobDataJSON json.RawMessage) (interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(jobDataJSON, &data); err != nil {
		return nil, fmt.Errorf("failed to parse job data: %w", err)
	}

	audioR2Key, ok := data["audio_r2_key"].(string)
	if !ok {
		return nil, fmt.Errorf("missing audio_r2_key in job data")
	}

	trackVersionID, ok := data["track_version_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing track_version_id in job data")
	}
	trackVersionID = strings.TrimSpace(trackVersionID)
	if trackVersionID == "" {
		return nil, fmt.Errorf("track_version_id is empty")
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg not available on host: %w", err)
	}

	jp.log.Info("generating waveform", "audio_key", audioR2Key, "track_version_id", trackVersionID)

	workDir, err := os.MkdirTemp("", "purptape-waveform-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create waveform temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	inputPath := filepath.Join(workDir, "input_audio")
	inputFile, err := os.Create(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp waveform input file: %w", err)
	}

	if _, err := jp.r2.DownloadFileToWriter(ctx, audioR2Key, inputFile); err != nil {
		inputFile.Close()
		return nil, fmt.Errorf("failed to download source audio: %w", err)
	}
	if err := inputFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize temp waveform input file: %w", err)
	}

	outputPath := filepath.Join(workDir, "waveform.png")
	ffmpegCmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i",
		inputPath,
		"-filter_complex",
		"aformat=channel_layouts=mono,showwavespic=s=1200x300:colors=white",
		"-frames:v",
		"1",
		outputPath,
	)
	if output, err := ffmpegCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg waveform generation failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	waveformR2Key := fmt.Sprintf("processed/waveforms/%s/%d.png", trackVersionID, time.Now().UTC().Unix())
	waveformFile, err := os.Open(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open generated waveform file: %w", err)
	}
	defer waveformFile.Close()

	uploadResult, err := jp.r2.UploadFile(ctx, waveformR2Key, waveformFile, "image/png")
	if err != nil {
		return nil, fmt.Errorf("failed to upload generated waveform: %w", err)
	}

	return map[string]interface{}{
		"status":             "completed",
		"track_version_id":   trackVersionID,
		"source_audio_key":   audioR2Key,
		"waveform_image_key": uploadResult.Key,
		"file_size_bytes":    uploadResult.FileSize,
		"checksum":           uploadResult.Checksum,
	}, nil
}

// EnqueueJob creates a new background job
func (jp *JobProcessor) EnqueueJob(ctx context.Context, jobType string, data map[string]interface{}) (string, error) {
	jobType = strings.TrimSpace(jobType)
	if jobType == "" {
		return "", fmt.Errorf("job type is required")
	}

	if data == nil {
		data = map[string]interface{}{}
	}

	jobID, err := jp.db.CreateBackgroundJob(ctx, jobType, data, 3)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue job: %w", err)
	}

	jp.log.Info("enqueued background job", "job_id", jobID, "job_type", jobType)
	return jobID, nil
}

// EnqueueCleanup queues an orphaned file for deletion
func (jp *JobProcessor) EnqueueCleanup(ctx context.Context, r2ObjectKey string) (string, error) {
	return jp.EnqueueJob(ctx, "cleanup_r2_file", map[string]interface{}{
		"r2_object_key": r2ObjectKey,
	})
}

// EnqueueVideoConversion queues video-to-audio conversion
func (jp *JobProcessor) EnqueueVideoConversion(ctx context.Context, videoR2Key, trackVersionID string) (string, error) {
	return jp.EnqueueJob(ctx, "convert_video_to_audio", map[string]interface{}{
		"video_r2_key":     videoR2Key,
		"track_version_id": trackVersionID,
	})
}

// EnqueueWaveformGeneration queues waveform generation
func (jp *JobProcessor) EnqueueWaveformGeneration(ctx context.Context, audioR2Key, trackVersionID string) (string, error) {
	return jp.EnqueueJob(ctx, "generate_waveform", map[string]interface{}{
		"audio_r2_key":     audioR2Key,
		"track_version_id": trackVersionID,
	})
}
