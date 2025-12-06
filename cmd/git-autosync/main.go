package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/find-xposed-magisk/git-sync/internal/batch"
	"github.com/find-xposed-magisk/git-sync/internal/config"
	"github.com/find-xposed-magisk/git-sync/internal/file"
	"github.com/find-xposed-magisk/git-sync/internal/git"
	"github.com/find-xposed-magisk/git-sync/internal/logger"
	"github.com/find-xposed-magisk/git-sync/internal/merge"
	"github.com/find-xposed-magisk/git-sync/internal/subrepo"
)

// Version information injected by GoReleaser via ldflags
// 由 GoReleaser 通过 ldflags 注入的版本信息
var (
	version = "dev"     // Version tag (e.g., v2.0.0) / 版本标签
	commit  = "none"    // Git commit hash / Git 提交哈希
	date    = "unknown" // Build date / 构建日期
)

func main() {
	// 解析命令行参数
	// Parse command line arguments
	debugMode := flag.Bool("debug", false, "Enable debug mode (verbose logging)")
	showVersion := flag.Bool("version", false, "Show version information and exit")
	flag.Parse()

	// 显示版本信息后退出
	// Show version info and exit
	if *showVersion {
		fmt.Printf("git-sync %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", date)
		os.Exit(0)
	}
	
	// 创建日志记录器
	// Create logger
	log := logger.NewLogger(true)

	// 获取工作目录用于加载配置文件
	// Get working directory for loading config file
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}

	// 加载配置（从文件或使用默认值）
	// Load configuration (from file or use defaults)
	cfg, err := config.LoadConfigFromFile(workDir)
	if err != nil {
		log.Warn("配置加载警告 / Config load warning: %v", err)
		cfg = config.DefaultConfig()
	}

	// 从配置读取日志级别
	// Read log level from config
	logLevel := parseLogLevel(cfg.LogLevel)

	// 命令行参数可以覆盖配置
	// Command line parameter can override config
	if *debugMode {
		logLevel = logger.DEBUG
		log.Info("⚙️ DEBUG模式已启用 / DEBUG mode enabled")
	}

	log.SetLevel(logLevel)

	// 初始化分级日志系统（使用配置值）
	// Initialize multi-level log system (using config values)
	multiWriter, err := logger.NewMultiLevelWriter(cfg.LogDir, cfg.LogMaxSizeMB, cfg.LogMaxBackups)
	if err != nil {
		// 如果创建失败，只输出到终端
		// If creation fails, only output to terminal
		fmt.Printf("Warning: Failed to create multi-level log writer: %v\n", err)
		fmt.Println("Logs will only be output to terminal.")
	} else {
		log.SetMultiLevelWriter(multiWriter)
		defer multiWriter.Close()
	}
	
	log.Info("=================================================================================")
	log.Info("  Advanced Git Auto-Sync (GO版本 / GO Version)")
	log.Info("  v12.2 智能合并与虚拟环境过滤 / Intelligent Merge & Virtual Env Filter")
	log.Info("=================================================================================")
	
	// 获取仓库根目录
	// Get repository root directory
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		log.Error("Failed to get repository root: %v", err)
		os.Exit(1)
	}
	
	// 配置已在上面加载
	// Config already loaded above
	cfg.RepoRoot = repoRoot
	
	log.Info("仓库根目录 / Repository root: %s", repoRoot)
	
	// 创建Git操作实例
	// Create Git operations instance
	gitOps := git.NewGitOps(cfg, log)
	
	// 确保依赖已安装
	// Ensure dependencies are installed
	if err := gitOps.EnsureDependencies(); err != nil {
		log.Error("Failed to ensure dependencies: %v", err)
		os.Exit(1)
	}
	
	// 创建各个处理器
	// Create processors
	fileProc := file.NewFileProcessor(cfg, gitOps, log)
	subrepoProc := subrepo.NewSubrepoProcessor(cfg, gitOps, log)
	mergeManager := merge.NewMergeManager(cfg, gitOps, log)
	
	// 主循环
	// Main loop
	log.Info("开始主循环，同步间隔: %v / Starting main loop, sync interval: %v", cfg.SleepInterval, cfg.SleepInterval)
	
	// 失败计数器 / Failure counter
	consecutiveFailures := 0
	maxConsecutiveFailures := cfg.MaxConsecutiveFailures
	
	for {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		log.Timestamp("开始同步周期 / Starting sync cycle")
		
		// =================== 阶段-1: 全局锁检测 / Phase -1: Global lock check ===================
		// 在每个周期开始前检测并清理过期的 index.lock 文件
		// Check and clean stale index.lock before each cycle
		lockPath := filepath.Join(repoRoot, ".git", "index.lock")
		if info, err := os.Stat(lockPath); err == nil {
			lockAge := time.Since(info.ModTime())
			log.Debug("[全局LOCK检测] index.lock 存在，年龄: %v / index.lock exists, age: %v", lockAge, lockAge)
			
			// 如果 lock 文件超过配置时间，认为是残留文件
			// If lock file is older than configured time, consider it stale
			if lockAge > cfg.LockFileMaxAge {
				log.Warn("[全局LOCK清理] 发现过期 index.lock (年龄: %v)，尝试清理... / Found stale index.lock (age: %v), cleaning...", lockAge, lockAge)
				if err := os.Remove(lockPath); err != nil {
					log.Error("[全局LOCK清理] 清理失败 / Cleanup failed: %v", err)
				} else {
					log.Info("[全局LOCK清理] ✓ 过期 lock 文件已清理 / Stale lock file cleaned")
				}
			} else {
				// lock 文件较新，可能是 CNB 平台的 git notes 操作，等待释放
				// Lock file is recent, might be CNB platform git notes operation, wait for release
				log.Info("[全局LOCK等待] lock 文件较新 (年龄: %v)，等待 %v 后继续... / Lock file is recent (age: %v), waiting %v...", lockAge, cfg.LockWaitTime, lockAge, cfg.LockWaitTime)
				time.Sleep(cfg.LockWaitTime)
			}
		}
		
		// =================== 阶段0: 健康检查 / Phase 0: Health check ===================
		if err := performHealthCheck(gitOps, log); err != nil {
			log.Error("健康检查失败 / Health check failed: %v", err)
			// 尝试修复后继续
			// Continue after attempting repair
		}
		
		// =================== 阶段1: 特殊仓库处理 / Phase 1: Special repository processing ===================
		log.Info("阶段1：处理特殊仓库 / Phase 1: Processing special repositories")
		if err := subrepoProc.ProcessAllSubrepos(); err != nil {
			log.Error("Failed to process subrepos: %v", err)
		}
		
		// =================== 阶段1.5: 清理孤儿gitdir / Phase 1.5: Clean orphaned gitdir ===================
		log.Info("阶段1.5：清理孤儿gitdir目录 / Phase 1.5: Cleaning orphaned gitdir directories")
		if err := subrepoProc.CleanOrphanedGitdirs(); err != nil {
			log.Error("Failed to clean orphaned gitdirs: %v", err)
		}
		
		// =================== 阶段2: 智能.gitignore清理 / Phase 2: Intelligent .gitignore cleanup ===================
		log.Info("阶段2：智能清理.gitignore规则变化 / Phase 2: Intelligent cleanup of .gitignore rule changes")
		if err := cleanIgnoredFiles(cfg, gitOps, fileProc, log); err != nil {
			log.Error("Failed to clean ignored files: %v", err)
		}
		
		// =================== 阶段3: 常规文件处理 / Phase 3: Regular file processing ===================
		log.Info("阶段3：处理常规文件变更 / Phase 3: Processing regular file changes")
		
		// 处理已删除文件
		// Process deleted files
		log.Debug("处理已删除文件 / Processing deleted files")
		if err := processDeletedFiles(cfg, gitOps, fileProc, log); err != nil {
			log.Error("Failed to process deleted files: %v", err)
		}
		
		// 处理修改和新增文件
		// Process modified and new files
		log.Debug("处理修改和新增文件 / Processing modified and new files")
		if err := processModifiedFiles(cfg, gitOps, fileProc, log); err != nil {
			log.Error("Failed to process modified files: %v", err)
		}
		
		// 处理空目录
		// Process empty directories
		if err := fileProc.HandleEmptyDirectories(); err != nil {
			log.Error("Failed to handle empty directories: %v", err)
		}
		
		// =================== 统一提交阶段 / Unified commit phase ===================
		// 【核心改进】学习Shell版本的统一提交点设计
		// [Core Improvement] Learn from Shell version's unified commit point design
		log.Info("统一提交阶段：提交所有暂存变更 / Unified commit phase: Committing all staged changes")
		hasChanges, err := gitOps.HasStagedChanges()
		if err != nil {
			log.Error("Failed to check staged changes: %v", err)
		}
		
		if hasChanges {
			log.Info("提交所有阶段的暂存变更 / Committing staged changes from all phases")
			commitMsg := fmt.Sprintf("%s All changes at %s", cfg.CommitMsgPrefix, timestamp)
			if err := gitOps.Commit(commitMsg); err != nil {
				log.Error("Failed to commit: %v", err)
			} else {
				// 【核心改进】提交后立即推送，避免时序竞态
				// [Core Improvement] Push immediately after commit to avoid race condition
				log.Info("立即推送当前提交 / Pushing current commit immediately")
				if err := gitOps.Push(); err != nil {
					log.Warn("推送失败，将在合并后重试 / Push failed, will retry after merge: %v", err)
				}
			}
		} else {
			log.Info("无新变更需要提交 / No new changes to commit")
		}
		
		// =================== 阶段4: 远程同步 / Phase 4: Remote sync ===================
		log.Info("")
		log.Info("阶段4：与远程同步（智能三路合并）/ Phase 4: Syncing with remote (Intelligent three-way merge)")
		
		if err := gitOps.Fetch(); err != nil {
			log.Error("Failed to fetch: %v", err)
			consecutiveFailures++
		} else {
			if err := mergeManager.SmartThreeWayMerge(); err != nil {
				consecutiveFailures++
				log.Warn("[警告] 智能合并未完全成功 (%d/%d) / [WARNING] Intelligent merge not fully successful (%d/%d)", 
					consecutiveFailures, maxConsecutiveFailures, consecutiveFailures, maxConsecutiveFailures)
				
				// 失败保护机制 / Failure protection mechanism
				if consecutiveFailures >= maxConsecutiveFailures {
					log.Error("连续失败 %d 次，进入安全模式 / Consecutive failures %d times, entering safe mode", 
						maxConsecutiveFailures, maxConsecutiveFailures)
					safeSleep := cfg.SleepInterval * time.Duration(cfg.SafeModeMultiplier)
					log.Info("延长等待时间至 %v / Extending wait time to %v", safeSleep, safeSleep)
					time.Sleep(safeSleep)
					consecutiveFailures = 0 // 重置计数器 / Reset counter
					continue
				}
			} else {
				// 成功后重置失败计数器 / Reset failure counter on success
				if consecutiveFailures > 0 {
					log.Info("合并成功，重置失败计数器 / Merge successful, resetting failure counter")
					consecutiveFailures = 0
				}
				
				// 定期清理旧备份分支 / Periodically clean old backup branches
				if err := mergeManager.CleanupOldBackups(cfg.MaxBackupBranches); err != nil {
					log.Warn("Failed to cleanup old backups: %v", err)
				}
			}
		}
		
		// 等待下一个周期
		// Wait for next cycle
		log.Info("--- 周期完成，等待 %v / Cycle complete. Waiting for %v ---", cfg.SleepInterval, cfg.SleepInterval)
		log.Info("")
		time.Sleep(cfg.SleepInterval)
	}
}

// performHealthCheck 执行仓库健康检查
// Performs repository health check
func performHealthCheck(gitOps *git.GitOps, log *logger.Logger) error {
	log.Debug("执行仓库健康检查 / Performing repository health check")
	
	// 检查工作区状态 / Check working directory status
	if hasUncommitted, err := gitOps.HasUncommittedChanges(); err != nil {
		log.Warn("无法检查工作区状态 / Cannot check working directory status: %v", err)
		// 尝试修复：重建Git索引 / Attempt repair: rebuild index
		log.Info("尝试重建Git索引 / Attempting to rebuild git index")
		if err := gitOps.Reset("HEAD", false); err != nil {
			log.Error("索引重建失败 / Index rebuild failed: %v", err)
			return err
		}
	} else if hasUncommitted {
		log.Debug("工作区有未提交变更（正常）/ Working directory has uncommitted changes (normal)")
	}
	
	// 检查暂存区状态 / Check staging area status
	if hasStaged, err := gitOps.HasStagedChanges(); err != nil {
		log.Warn("无法检查暂存区状态 / Cannot check staging area status: %v", err)
	} else if hasStaged {
		log.Debug("暂存区有变更（正常）/ Staging area has changes (normal)")
	}
	
	return nil
}

// cleanIgnoredFiles 清理被.gitignore忽略但仍被追踪的文件
// Cleans files that are ignored by .gitignore but still tracked
func cleanIgnoredFiles(cfg *config.Config, gitOps *git.GitOps, fileProc *file.FileProcessor, log *logger.Logger) error {
	// 获取应被忽略的已追踪文件
	// Get tracked files that should be ignored
	ignoredFiles, err := gitOps.ListFiles("-z", "--cached", "--ignored", "--exclude-standard", "--", ".")
	if err != nil {
		return err
	}
	
	// 构建特殊仓库路径列表
	// Build special repository paths list
	specialRepoPaths := []string{}
	for _, baseDir := range cfg.SubrepoBaseDirs {
		basePath := filepath.Join(cfg.RepoRoot, baseDir)
		
		// 添加base_dir本身
		// Add base_dir itself
		if info, err := os.Stat(basePath); err == nil && info.IsDir() {
			// 检查是否为特殊仓库
			// Check if it's a special repository
			if isSpecialRepo(basePath) {
				specialRepoPaths = append(specialRepoPaths, baseDir)
			}
		}
		
		// 查找一级子目录
		// Find first-level subdirectories
		if entries, err := os.ReadDir(basePath); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					subPath := filepath.Join(basePath, entry.Name())
					if isSpecialRepo(subPath) {
						specialRepoPaths = append(specialRepoPaths, filepath.Join(baseDir, entry.Name()))
					}
				}
			}
		}
	}
	
	// 过滤需要取消追踪的文件
	// Filter files to untrack
	filesToUntrack := []string{}
	for _, filePath := range ignoredFiles {
		if filePath == "" {
			continue
		}
		
		shouldUntrack := true
		
		// 检查是否属于特殊仓库
		// Check if belongs to special repository
		for _, specialRepo := range specialRepoPaths {
			if strings.HasPrefix(filePath, specialRepo+"/") {
				log.Debug("保护特殊仓库文件 / Protecting special repo file: %s", filePath)
				shouldUntrack = false
				break
			}
		}
		
		// 排除关键文件
		// Exclude critical files
		if strings.Contains(filePath, "/gitdir/") || 
		   strings.Contains(filePath, "/gitdir.tar") || 
		   filePath == cfg.IgnoreFileName {
			shouldUntrack = false
		}
		
		if shouldUntrack {
			filesToUntrack = append(filesToUntrack, filePath)
		}
	}
	
	// 执行取消追踪
	// Execute untracking
	if len(filesToUntrack) > 0 {
		// 使用统一的批量处理框架（使用配置值）
		// Use unified batch processing framework (using config values)
		batchConfig := &batch.BatchConfig{
			SmallFileThreshold:  cfg.SmallFileThreshold,
			MediumFileThreshold: cfg.MediumFileThreshold,
			BatchSize:           cfg.BatchSize,
			MaxWorkers:          cfg.MaxParallelWorkers,
			EnableProgress:      true,
			EnableMetrics:       true,
			RetryMaxAttempts:    cfg.BatchRetryMaxAttempts,
			RetryBaseDelay:      cfg.BatchRetryBaseDelay,
		}
		batchProcessor := batch.NewGitBatchProcessorWithConfig(cfg.RepoRoot, log, batchConfig)
		if err := batchProcessor.BatchRemove(filesToUntrack); err != nil {
			log.Warn("Batch remove failed: %v", err)
		}
		log.Info("已取消追踪 %d 个文件，等待统一提交 / Untracked %d files, waiting for unified commit", len(filesToUntrack), len(filesToUntrack))
		// 【核心改进】移除内部提交，由统一提交点处理
		// [Core Improvement] Remove internal commit, handled by unified commit point
	} else {
		log.Info("无需取消追踪文件 / No files need to be untracked")
	}
	
	return nil
}

// isSpecialRepo 检查是否为特殊仓库
// Checks if it's a special repository
func isSpecialRepo(path string) bool {
	// 检查.git目录
	// Check for .git directory
	if info, err := os.Stat(filepath.Join(path, ".git")); err == nil && info.IsDir() {
		return true
	}
	
	// 检查gitdir目录
	// Check for gitdir directory
	if info, err := os.Stat(filepath.Join(path, "gitdir")); err == nil && info.IsDir() {
		return true
	}
	
	// 检查gitdir.tar文件
	// Check for gitdir.tar file
	if info, err := os.Stat(filepath.Join(path, "gitdir.tar")); err == nil && !info.IsDir() {
		return true
	}
	
	return false
}

// processDeletedFiles 处理已删除的文件
// Processes deleted files
func processDeletedFiles(cfg *config.Config, gitOps *git.GitOps, fileProc *file.FileProcessor, log *logger.Logger) error {
	// 获取已删除的文件列表
	// Get list of deleted files
	deletedFiles, err := gitOps.ListFiles("-z", "--deleted", "--exclude-standard", "--", ".")
	if err != nil {
		return err
	}
	
	for _, filePath := range deletedFiles {
		if filePath == "" {
			continue
		}
		
		// 跳过特殊仓库中的文件
		// Skip files in special repositories
		if fileProc.IsInSpecialRepo(filePath) {
			continue
		}
		
		// 从索引中删除
		// Remove from index
		if err := gitOps.Remove(filePath); err != nil {
			log.Warn("Failed to remove deleted file %s: %v", filePath, err)
		}
	}
	
	return nil
}

// processModifiedFiles 处理修改和新增的文件
// Processes modified and new files
func processModifiedFiles(cfg *config.Config, gitOps *git.GitOps, fileProc *file.FileProcessor, log *logger.Logger) error {
	startTime := time.Now()
	
	// 获取修改和新增的文件列表
	// Get list of modified and new files
	modifiedFiles, err := gitOps.ListFiles("-z", "--modified", "--others", "--exclude-standard", "--", ".")
	if err != nil {
		return err
	}
	
	log.Debug("获取到 %d 个修改/新增文件 / Got %d modified/new files", len(modifiedFiles), len(modifiedFiles))
	
	// 收集需要暂存的文件
	// Collect files to stage
	filesToStage := []string{}
	skippedCount := 0
	
	for _, filePath := range modifiedFiles {
		if filePath == "" {
			continue
		}
		
		// 跳过特殊仓库中的文件
		// Skip files in special repositories
		if fileProc.IsInSpecialRepo(filePath) {
			log.Debug("跳过特殊仓库文件 / Skipping special repo file: %s", filePath)
			skippedCount++
			continue
		}
		
		// 检查文件大小
		// Check file size
		fullPath := filepath.Join(cfg.RepoRoot, filePath)
		if info, err := os.Stat(fullPath); err == nil {
			fileSize := info.Size()
			
			// 超过忽略阈值
			// Exceeds ignore threshold
			if fileSize > cfg.IgnoreSizeThresholdBytes {
				log.Warn("忽略大文件 / Ignoring large file: %s (%d bytes) -> 添加到 %s", filePath, fileSize, cfg.IgnoreFileName)
				
				// 将路径写入 .gitignore_nopush（如果不存在则追加）
				// Append path to .gitignore_nopush if not already present
				ignoreFilePath := filepath.Join(cfg.RepoRoot, cfg.IgnoreFileName)
				
				// 读取现有内容
				// Read existing content
				existingContent, _ := os.ReadFile(ignoreFilePath)
				existingLines := strings.Split(string(existingContent), "\n")
				
				// 检查是否已存在
				// Check if already exists
				alreadyExists := false
				for _, line := range existingLines {
					if strings.TrimSpace(line) == filePath {
						alreadyExists = true
						break
					}
				}
				
				// 如果不存在则追加
				// Append if not exists
				if !alreadyExists {
					f, err := os.OpenFile(ignoreFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err == nil {
						f.WriteString(filePath + "\n")
						f.Close()
						log.Debug("  ↳ 已添加到 %s / Added to %s", cfg.IgnoreFileName, cfg.IgnoreFileName)
					} else {
						log.Warn("  ↳ 写入 %s 失败 / Failed to write to %s: %v", cfg.IgnoreFileName, cfg.IgnoreFileName, err)
					}
				}
				
				// 暂存 .gitignore_nopush 文件
				// Stage .gitignore_nopush file
				if err := gitOps.Add(cfg.IgnoreFileName); err != nil {
					log.Warn("  ↳ 暂存 %s 失败 / Failed to stage %s: %v", cfg.IgnoreFileName, cfg.IgnoreFileName, err)
				}
				
				continue
			}
			
			// 超过LFS阈值
			// Exceeds LFS threshold
			if fileSize > cfg.LFSSizeThresholdBytes {
				log.Warn("LFS追踪 / LFS tracking: %s (%d bytes)", filePath, fileSize)
				gitOps.LFSTrack(filePath)
				gitOps.Add(".gitattributes")
			}
		}
		
		filesToStage = append(filesToStage, filePath)
	}
	
	// 批量git add
	// Batch git add
	if len(filesToStage) > 0 {
		// 使用统一的批量处理框架（使用配置值）
		// Use unified batch processing framework (using config values)
		batchConfig := &batch.BatchConfig{
			SmallFileThreshold:  cfg.SmallFileThreshold,
			MediumFileThreshold: cfg.MediumFileThreshold,
			BatchSize:           cfg.BatchSize,
			MaxWorkers:          cfg.MaxParallelWorkers,
			EnableProgress:      true,
			EnableMetrics:       true,
			RetryMaxAttempts:    cfg.BatchRetryMaxAttempts,
			RetryBaseDelay:      cfg.BatchRetryBaseDelay,
		}
		batchProcessor := batch.NewGitBatchProcessorWithConfig(cfg.RepoRoot, log, batchConfig)
		if err := batchProcessor.BatchAdd(filesToStage); err != nil {
			log.Error("Failed to batch add files: %v", err)
			return err
		}
	}
	
	totalDuration := time.Since(startTime)
	log.Info("处理完成 / Processing complete: 暂存 %d 个文件, 跳过 %d 个文件 / Staged %d files, skipped %d files (耗时 / took: %v)", 
		len(filesToStage), skippedCount, len(filesToStage), skippedCount, totalDuration)
	
	return nil
}

// parseLogLevel 解析日志级别字符串
// Parses log level string
func parseLogLevel(s string) logger.LogLevel {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return logger.DEBUG
	case "INFO":
		return logger.INFO
	case "WARN":
		return logger.WARN
	case "ERROR":
		return logger.ERROR
	default:
		return logger.INFO // 默认INFO级别 / Default INFO level
	}
}

// batchAddFiles is deprecated, use batch.GitBatchProcessor instead
// batchAddFiles 已废弃，请使用 batch.GitBatchProcessor
