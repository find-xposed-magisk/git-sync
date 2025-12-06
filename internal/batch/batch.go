package batch

import (
	"bytes"
	"math"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/find-xposed-magisk/git-sync/internal/logger"
)

// FileClassification File classification by size / 按大小分类文件
// FileClassification 文件分类
type FileClassification struct {
	Small  []string // <5MB
	Medium []string // 5-100MB
	Large  []string // >100MB
}

// ClassifyFilesBySize Classify files by size / 按大小分类文件
// ClassifyFilesBySize 按大小分类文件
func ClassifyFilesBySize(files []string) *FileClassification {
	classification := &FileClassification{
		Small:  make([]string, 0),
		Medium: make([]string, 0),
		Large:  make([]string, 0),
	}

	for _, file := range files {
		if info, err := os.Stat(file); err == nil {
			fileSize := info.Size()
			if fileSize < 5*1024*1024 {
				classification.Small = append(classification.Small, file)
			} else if fileSize < 100*1024*1024 {
				classification.Medium = append(classification.Medium, file)
			} else {
				classification.Large = append(classification.Large, file)
			}
		} else {
			// If cannot stat, treat as small file / 无法获取大小，视为小文件
			classification.Small = append(classification.Small, file)
		}
	}

	return classification
}

// BatchConfig Batch processing configuration / 批量处理配置
// BatchConfig 批量处理配置
type BatchConfig struct {
	SmallFileThreshold  int64 // <5MB
	MediumFileThreshold int64 // <100MB
	BatchSize           int
	MaxWorkers          int
	EnableProgress      bool
	EnableMetrics       bool
	// 重试配置 / Retry configuration
	RetryMaxAttempts int           // 最大重试次数 / Max retry attempts
	RetryBaseDelay   time.Duration // 重试基础延迟 / Base delay for retry
}

// DefaultBatchConfig Default batch configuration / 默认批量配置
// DefaultBatchConfig 默认批量配置
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		SmallFileThreshold:  5 * 1024 * 1024,   // 5MB
		MediumFileThreshold: 100 * 1024 * 1024, // 100MB
		BatchSize:           100,
		MaxWorkers:          4,
		EnableProgress:      true,
		EnableMetrics:       true,
		RetryMaxAttempts:    3,               // 默认重试3次 / Default 3 retries
		RetryBaseDelay:      1 * time.Second, // 默认延迟1秒 / Default 1s delay
	}
}

// PerformanceMetrics Performance metrics / 性能指标
// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	TotalFiles     int
	ProcessedFiles int
	FailedFiles    int
	TotalDuration  time.Duration
	AvgBatchTime   time.Duration
	BatchCount     int
}

// GitBatchProcessor Git batch operation processor / Git批量操作处理器
// GitBatchProcessor Git批量操作处理器
type GitBatchProcessor struct {
	repoRoot string
	logger   *logger.Logger
	config   *BatchConfig
	metrics  *PerformanceMetrics
}

// NewGitBatchProcessor Create new Git batch processor / 创建新的Git批量处理器
// NewGitBatchProcessor 创建新的Git批量处理器
func NewGitBatchProcessor(repoRoot string, log *logger.Logger, maxWorkers int) *GitBatchProcessor {
	config := DefaultBatchConfig()
	config.MaxWorkers = maxWorkers
	
	return &GitBatchProcessor{
		repoRoot: repoRoot,
		logger:   log,
		config:   config,
		metrics: &PerformanceMetrics{
			TotalFiles:     0,
			ProcessedFiles: 0,
			FailedFiles:    0,
		},
	}
}

// NewGitBatchProcessorWithConfig Create new Git batch processor with custom config / 使用自定义配置创建批量处理器
// NewGitBatchProcessorWithConfig 使用自定义配置创建批量处理器
func NewGitBatchProcessorWithConfig(repoRoot string, log *logger.Logger, config *BatchConfig) *GitBatchProcessor {
	return &GitBatchProcessor{
		repoRoot: repoRoot,
		logger:   log,
		config:   config,
		metrics: &PerformanceMetrics{
			TotalFiles:     0,
			ProcessedFiles: 0,
			FailedFiles:    0,
		},
	}
}

// SetBatchSize Set batch size / 设置批次大小
// SetBatchSize 设置批次大小
func (p *GitBatchProcessor) SetBatchSize(size int) {
	p.config.BatchSize = size
}

// SetEnableProgress Enable/disable progress feedback / 启用/禁用进度反馈
// SetEnableProgress 启用/禁用进度反馈
func (p *GitBatchProcessor) SetEnableProgress(enable bool) {
	p.config.EnableProgress = enable
}

// GetMetrics Get performance metrics / 获取性能指标
// GetMetrics 获取性能指标
func (p *GitBatchProcessor) GetMetrics() *PerformanceMetrics {
	return p.metrics
}

// ResetMetrics Reset performance metrics / 重置性能指标
// ResetMetrics 重置性能指标
func (p *GitBatchProcessor) ResetMetrics() {
	p.metrics = &PerformanceMetrics{
		TotalFiles:     0,
		ProcessedFiles: 0,
		FailedFiles:    0,
	}
}

// calculateDynamicBatchSize Calculate dynamic batch size based on file characteristics / 根据文件特征动态计算批次大小
// calculateDynamicBatchSize 根据文件特征动态计算批次大小
func (p *GitBatchProcessor) calculateDynamicBatchSize(files []string) int {
	if len(files) == 0 {
		return p.config.BatchSize
	}
	
	// Calculate average file size / 计算平均文件大小
	var totalSize int64
	validFiles := 0
	
	for _, file := range files {
		if info, err := os.Stat(file); err == nil {
			totalSize += info.Size()
			validFiles++
		}
	}
	
	if validFiles == 0 {
		return p.config.BatchSize
	}
	
	avgSize := totalSize / int64(validFiles)
	totalFiles := len(files)
	
	// Dynamic batch size strategy / 动态批次大小策略
	if totalFiles < 50 {
		return totalFiles // Process all at once / 一次处理完
	} else if totalFiles < 100 {
		return 50
	} else if avgSize < 1*1024*1024 { // <1MB
		return 200 // Large batch for small files / 小文件大批次
	} else if avgSize < 10*1024*1024 { // <10MB
		return 100 // Medium batch / 中等批次
	} else {
		return 50 // Small batch for large files / 大文件小批次
	}
}

// BatchAdd Batch add files with intelligent classification / 智能分类批量添加文件
// BatchAdd 智能分类批量添加文件
func (p *GitBatchProcessor) BatchAdd(files []string) error {
	if len(files) == 0 {
		return nil
	}

	p.logger.Info("批量添加 %d 个文件 / Batch adding %d files", len(files), len(files))
	startTime := time.Now()
	
	// Initialize metrics / 初始化指标
	if p.config.EnableMetrics {
		p.metrics.TotalFiles = len(files)
		p.metrics.ProcessedFiles = 0
		p.metrics.FailedFiles = 0
		p.metrics.BatchCount = 0
	}

	// Classify files by size / 按大小分类文件
	classification := ClassifyFilesBySize(files)

	totalProcessed := 0
	var mu sync.Mutex

	// Process small files in parallel / 并行处理小文件
	if len(classification.Small) > 0 {
		p.logger.Debug("并行处理 %d 个小文件 / Parallel processing %d small files", 
			len(classification.Small), len(classification.Small))
		
		processed := p.processFilesParallel(classification.Small, "add")
		mu.Lock()
		totalProcessed += processed
		mu.Unlock()
	}

	// Process medium files in batches / 批量处理中文件
	if len(classification.Medium) > 0 {
		p.logger.Debug("批量处理 %d 个中文件 / Batch processing %d medium files", 
			len(classification.Medium), len(classification.Medium))
		
		processed := p.processFilesBatch(classification.Medium, "add")
		mu.Lock()
		totalProcessed += processed
		mu.Unlock()
	}

	// Process large files serially / 串行处理大文件
	if len(classification.Large) > 0 {
		p.logger.Warn("串行处理 %d 个大文件 / Serial processing %d large files", 
			len(classification.Large), len(classification.Large))
		
		processed := p.processFilesSerial(classification.Large, "add")
		mu.Lock()
		totalProcessed += processed
		mu.Unlock()
	}

	duration := time.Since(startTime)
	
	// Update metrics / 更新指标
	if p.config.EnableMetrics {
		p.metrics.TotalDuration = duration
		if p.metrics.BatchCount > 0 {
			p.metrics.AvgBatchTime = duration / time.Duration(p.metrics.BatchCount)
		}
		p.metrics.FailedFiles = p.metrics.TotalFiles - totalProcessed
		
		p.logger.Info("批量添加完成 / Batch add complete: %d/%d 文件 / files (耗时 / took: %v, 平均批次 / avg batch: %v)", 
			totalProcessed, len(files), duration, p.metrics.AvgBatchTime)
	} else {
		p.logger.Info("批量添加完成 / Batch add complete: %d/%d 文件 / files (耗时 / took: %v)", 
			totalProcessed, len(files), duration)
	}

	return nil
}

// BatchRemove Batch remove files with intelligent classification / 智能分类批量删除文件
// BatchRemove 智能分类批量删除文件
func (p *GitBatchProcessor) BatchRemove(files []string) error {
	if len(files) == 0 {
		return nil
	}

	p.logger.Info("开始批量删除 / Starting batch remove: %d files", len(files))
	startTime := time.Now()
	
	// Initialize metrics / 初始化指标
	if p.config.EnableMetrics {
		p.metrics.TotalFiles = len(files)
		p.metrics.ProcessedFiles = 0
		p.metrics.FailedFiles = 0
		p.metrics.BatchCount = 0
	}
	
	// Calculate dynamic batch size / 计算动态批次大小
	dynamicBatchSize := p.calculateDynamicBatchSize(files)
	p.logger.Debug("  ↳ 动态批次大小 / Dynamic batch size: %d", dynamicBatchSize)
	p.logger.Debug("  ↳ 最大并发数 / Max workers: %d", p.config.MaxWorkers)

	// For remove operations, use batch processing for all files / 删除操作统一使用批量处理
	// Because remove is usually fast and doesn't need size classification / 因为删除通常很快，不需要按大小分类
	processed := p.processFilesBatchWithSize(files, "rm", dynamicBatchSize)

	duration := time.Since(startTime)
	
	// Update metrics / 更新指标
	if p.config.EnableMetrics {
		p.metrics.TotalDuration = duration
		if p.metrics.BatchCount > 0 {
			p.metrics.AvgBatchTime = duration / time.Duration(p.metrics.BatchCount)
		}
		p.metrics.FailedFiles = p.metrics.TotalFiles - processed
		
		successCount := processed
		failedCount := p.metrics.FailedFiles
		
		p.logger.Info("✓ 批量删除完成 / Batch remove complete: %d/%d 文件 / files (耗时 / took: %v, 平均批次 / avg batch: %v, 批次数 / batches: %d)", 
			successCount, len(files), duration, p.metrics.AvgBatchTime, p.metrics.BatchCount)
		
		if failedCount > 0 {
			p.logger.Warn("  ⚠ 失败文件数 / Failed files: %d", failedCount)
		}
	} else {
		p.logger.Info("✓ 批量删除完成 / Batch remove complete: %d/%d 文件 / files (耗时 / took: %v)", 
			processed, len(files), duration)
	}

	return nil
}

// processFilesParallel Process files in parallel / 并行处理文件
// processFilesParallel 并行处理文件
func (p *GitBatchProcessor) processFilesParallel(files []string, operation string) int {
	var wg sync.WaitGroup
	sem := make(chan struct{}, p.config.MaxWorkers)
	successCount := 0
	var mu sync.Mutex

	// Split into small batches for parallel processing / 分成小批次并行处理
	smallBatchSize := 50
	batches := p.splitIntoBatches(files, smallBatchSize)

	for i, batch := range batches {
		wg.Add(1)
		go func(batchIndex int, batchFiles []string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if p.executeGitCommandWithRetry(operation, batchFiles) {
				mu.Lock()
				successCount += len(batchFiles)
				if p.config.EnableMetrics {
					p.metrics.ProcessedFiles += len(batchFiles)
					p.metrics.BatchCount++
				}
				mu.Unlock()
			}

			if p.config.EnableProgress {
				mu.Lock()
				progress := float64(successCount) / float64(len(files)) * 100
				p.logger.Debug("并行进度 / Parallel progress: %d/%d (%.1f%%)", 
					successCount, len(files), progress)
				mu.Unlock()
			}
		}(i, batch)
	}

	wg.Wait()
	return successCount
}

// processFilesBatch Process files in batches / 批量处理文件
// processFilesBatch 批量处理文件
func (p *GitBatchProcessor) processFilesBatch(files []string, operation string) int {
	return p.processFilesBatchWithSize(files, operation, p.config.BatchSize)
}

// processFilesBatchWithSize Process files in batches with custom batch size / 使用自定义批次大小批量处理文件
// processFilesBatchWithSize 使用自定义批次大小批量处理文件
func (p *GitBatchProcessor) processFilesBatchWithSize(files []string, operation string, batchSize int) int {
	batches := p.splitIntoBatches(files, batchSize)
	successCount := 0

	for i, batch := range batches {
		if p.executeGitCommandWithRetry(operation, batch) {
			successCount += len(batch)
			if p.config.EnableMetrics {
				p.metrics.ProcessedFiles += len(batch)
				p.metrics.BatchCount++
			}
		}

		if p.config.EnableProgress {
			processed := (i + 1) * batchSize
			if processed > len(files) {
				processed = len(files)
			}
			progress := float64(processed) / float64(len(files)) * 100
			p.logger.Debug("批量进度 / Batch progress: %d/%d (%.1f%%)", 
				processed, len(files), progress)
		}
	}

	return successCount
}

// processFilesSerial Process files serially / 串行处理文件
// processFilesSerial 串行处理文件
func (p *GitBatchProcessor) processFilesSerial(files []string, operation string) int {
	successCount := 0

	for i, file := range files {
		if info, err := os.Stat(file); err == nil {
			fileSize := float64(info.Size()) / 1024 / 1024
			p.logger.Warn("处理大文件 / Processing large file [%d/%d]: %s (%.2f MB)", 
				i+1, len(files), file, fileSize)
		}

		if p.executeGitCommandWithRetry(operation, []string{file}) {
			successCount++
			if p.config.EnableMetrics {
				p.metrics.ProcessedFiles++
				p.metrics.BatchCount++
			}
		}

		if p.config.EnableProgress {
			progress := float64(i+1) / float64(len(files)) * 100
			p.logger.Debug("串行进度 / Serial progress: %d/%d (%.1f%%)", 
				i+1, len(files), progress)
		}
	}

	return successCount
}

// executeGitCommand Execute git command without retry / 执行git命令（无重试）
// executeGitCommand 执行git命令（无重试）
// Note: For operations that may encounter lock contention, use executeGitCommandWithRetry instead
// 注意：对于可能遇到锁冲突的操作，请使用 executeGitCommandWithRetry
func (p *GitBatchProcessor) executeGitCommand(operation string, files []string) bool {
	var args []string

	switch operation {
	case "add":
		args = append([]string{"add", "--"}, files...)
	case "rm":
		args = append([]string{"rm", "--cached", "--ignore-unmatch", "--"}, files...)
	default:
		p.logger.Error("Unknown operation: %s", operation)
		return false
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = p.repoRoot

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		p.logger.Warn("Git %s failed (ignored): %v, stderr: %s", operation, err, stderr.String())
		return false
	}

	return true
}

// executeGitCommandWithRetry Execute git command with exponential backoff for lock contention / 使用指数退避重试执行git命令
// executeGitCommandWithRetry 使用指数退避重试执行git命令（处理锁冲突）
// This method implements intelligent retry logic for transient index.lock conflicts
// 该方法实现了针对瞬时 index.lock 冲突的智能重试逻辑
func (p *GitBatchProcessor) executeGitCommandWithRetry(operation string, files []string) bool {
	var args []string

	switch operation {
	case "add":
		args = append([]string{"add", "--"}, files...)
	case "rm":
		args = append([]string{"rm", "--cached", "--ignore-unmatch", "--"}, files...)
	default:
		p.logger.Error("Unknown operation: %s", operation)
		return false
	}

	// 使用配置的重试参数 / Use configured retry parameters
	maxRetries := p.config.RetryMaxAttempts
	baseDelay := p.config.RetryBaseDelay
	if maxRetries == 0 {
		maxRetries = 3 // 默认值 / Default value
	}
	if baseDelay == 0 {
		baseDelay = 1 * time.Second // 默认值 / Default value
	}

	for i := 0; i < maxRetries; i++ {
		// Create a new command object for each attempt / 每次尝试都创建新的命令对象
		cmd := exec.Command("git", args...)
		cmd.Dir = p.repoRoot

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		// Execute the command / 执行命令
		err := cmd.Run()

		// Success case / 成功情况
		if err == nil {
			if i > 0 {
				p.logger.Info("✓ Git %s succeeded after %d retries / Git %s 在 %d 次重试后成功", operation, i, operation, i)
			}
			return true
		}

		// Failure case: Check if it's a retryable lock error / 失败情况：检查是否为可重试的锁错误
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "index.lock") {
			// This is the error we want to retry on / 这是我们想要重试的错误
			delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(i)))
			p.logger.Info(
				"⚠ Git %s failed due to lock contention. Retrying in %v... (Attempt %d/%d) / Git %s 因锁冲突失败，%v 后重试...（第 %d/%d 次尝试）",
				operation,
				delay,
				i+1,
				maxRetries,
				operation,
				delay,
				i+1,
				maxRetries,
			)
			time.Sleep(delay)
			continue // Go to the next iteration / 进入下一次迭代
		}

		// Non-retryable error: Log and fail immediately / 不可重试的错误：记录并立即失败
		p.logger.Warn(
			"Git %s failed with a non-retryable error: %v, stderr: %s",
			operation,
			err,
			stderrStr,
		)
		return false
	}

	// If all retries failed / 如果所有重试都失败了
	p.logger.Error("✗ Git %s failed after %d attempts due to persistent lock contention / Git %s 在 %d 次尝试后因持续锁冲突失败", operation, maxRetries, operation, maxRetries)
	return false
}

// splitIntoBatches Split files into batches / 将文件分批
// splitIntoBatches 将文件分批
func (p *GitBatchProcessor) splitIntoBatches(files []string, batchSize int) [][]string {
	var batches [][]string

	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}
		batches = append(batches, files[i:end])
	}

	return batches
}
