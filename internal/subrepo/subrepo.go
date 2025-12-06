package subrepo

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/find-xposed-magisk/git-sync/internal/config"
	"github.com/find-xposed-magisk/git-sync/internal/git"
	"github.com/find-xposed-magisk/git-sync/internal/logger"
)

// SubrepoProcessor 特殊仓库处理器
// Special repository processor
type SubrepoProcessor struct {
	cfg       *config.Config
	gitOps    *git.GitOps
	logger    *logger.Logger
	hashCache *HashCache // hash缓存 / Hash cache
}

// NewSubrepoProcessor 创建特殊仓库处理器
// Creates a new special repository processor
func NewSubrepoProcessor(cfg *config.Config, gitOps *git.GitOps, log *logger.Logger) *SubrepoProcessor {
	return &SubrepoProcessor{
		cfg:       cfg,
		gitOps:    gitOps,
		logger:    log,
		hashCache: NewHashCache(), // 初始化hash缓存 / Initialize hash cache
	}
}

// fileOperation 文件操作结果
// File operation result
type fileOperation struct {
	mode string
	hash string
	path string
}

// subrepoJob 子仓库处理任务
// Subrepo processing job
type subrepoJob struct {
	path string // 仓库完整路径 / Full repository path
	name string // 仓库名称 / Repository name
}

// ProcessAllSubrepos 处理所有特殊仓库（并发优化版）
// Processes all special repositories (concurrent optimized version)
func (sp *SubrepoProcessor) ProcessAllSubrepos() error {
	sp.logger.Phase("部分A：协调子仓库状态 (并发模式) / Part A: Reconciling sub-repository states (Concurrent Mode)")
	
	// 阶段1：收集所有需要处理的子仓库目录
	// Phase 1: Collect all sub-repository directories to process
	subrepoMap := make(map[string]bool)
	
	for _, baseDir := range sp.cfg.SubrepoBaseDirs {
		basePath := filepath.Join(sp.cfg.RepoRoot, baseDir)
		
		// 添加base_dir本身
		// Add base_dir itself
		if info, err := os.Stat(basePath); err == nil && info.IsDir() {
			subrepoMap[baseDir] = true
		}
		
		// 查找一级子目录
		// Find first-level subdirectories
		if entries, err := os.ReadDir(basePath); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					subPath := filepath.Join(baseDir, entry.Name())
					subrepoMap[subPath] = true
				}
			}
		}
		
		// 从Git索引中获取已追踪的目录
		// Get tracked directories from git index
		files, err := sp.gitOps.ListFiles("-d", "--name-only", "HEAD", baseDir)
		if err == nil {
			for _, file := range files {
				if file != "" {
					subrepoMap[file] = true
				}
			}
		}
	}
	
	// 阶段2：筛选特殊仓库并准备并发任务
	// Phase 2: Filter special repos and prepare concurrent jobs
	var jobs []subrepoJob
	for subrepoDir := range subrepoMap {
		subrepoPath := filepath.Join(sp.cfg.RepoRoot, subrepoDir)
		subrepoName := filepath.Base(subrepoDir)
		
		// 检查是否为特殊仓库
		// Check if it's a special repository
		if sp.isSpecialRepo(subrepoPath) {
			jobs = append(jobs, subrepoJob{
				path: subrepoPath,
				name: subrepoName,
			})
		}
	}
	
	numRepos := len(jobs)
	if numRepos == 0 {
		sp.logger.Info("无特殊仓库需要处理 / No special repositories to process")
		return nil
	}
	
	// 阶段3：设置并发处理 Worker Pool
	// Phase 3: Set up concurrent processing Worker Pool
	// 使用配置的worker数量，但不超过仓库数量
	// Use configured worker count, but don't exceed number of repos
	numWorkers := sp.cfg.MaxParallelWorkers
	if numWorkers > numRepos {
		numWorkers = numRepos
	}
	
	jobsChan := make(chan subrepoJob, numRepos)
	errsChan := make(chan error, numRepos)
	var wg sync.WaitGroup
	
	sp.logger.Info("🚀 启动并发处理：%d 个特殊仓库，%d 个并发worker / Starting concurrent processing: %d special repos with %d workers", 
		numRepos, numWorkers, numRepos, numWorkers)
	
	// 阶段4：启动 worker goroutines
	// Phase 4: Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobsChan {
				sp.logger.Info("[Worker %d] 协调特殊仓库 / Reconciling special repo: %s", workerID, job.name)
				if err := sp.processSpecialRepoFastAndSafe(job.path, job.name); err != nil {
					// 装饰错误信息并发送到错误通道
					// Decorate error with context and send to error channel
					errsChan <- fmt.Errorf("[Worker %d] 处理仓库 %s 失败 / Failed to process repo %s: %w", 
						workerID, job.name, job.name, err)
				} else {
					sp.logger.Debug("[Worker %d] ✓ 完成 / Completed: %s", workerID, job.name)
				}
			}
		}(i + 1)
	}
	
	// 阶段5：分发任务到 workers
	// Phase 5: Distribute jobs to workers
	for _, job := range jobs {
		jobsChan <- job
	}
	close(jobsChan) // 发送完毕，关闭任务通道 / All jobs sent, close channel
	
	// 阶段6：等待所有 workers 完成并收集错误
	// Phase 6: Wait for all workers to finish and collect errors
	wg.Wait()
	close(errsChan)
	
	var processingErrors []string
	for err := range errsChan {
		sp.logger.Error(err.Error())
		processingErrors = append(processingErrors, err.Error())
	}
	
	if len(processingErrors) > 0 {
		return fmt.Errorf("%d/%d 个仓库处理失败 / %d/%d repositories failed to process:\n- %s",
			len(processingErrors),
			numRepos,
			len(processingErrors),
			numRepos,
			strings.Join(processingErrors, "\n- "),
		)
	}
	
	sp.logger.Info("✅ 成功处理所有 %d 个特殊仓库 / Successfully processed all %d special repositories", numRepos, numRepos)
	sp.logger.Debug("--- 部分A：子仓库协调完成 / Part A: Sub-repository reconciliation complete ---")
	return nil
}

// isSpecialRepo 检查是否为特殊仓库
// Checks if it's a special repository
func (sp *SubrepoProcessor) isSpecialRepo(path string) bool {
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

// processSpecialRepoFastAndSafe 高性能安全处理特殊仓库
// High-performance safe processing of special repository
func (sp *SubrepoProcessor) processSpecialRepoFastAndSafe(subrepoDir, subrepoName string) error {
	startTime := time.Now()
	sp.logger.Debug("使用高性能安全模式 / Using high-performance safe mode")
	
	// 检查目录是否存在
	// Check if directory exists
	if _, err := os.Stat(subrepoDir); os.IsNotExist(err) {
		sp.logger.Info("确认删除 / Confirmed deletion of: %s", subrepoName)
		// 删除索引中的所有文件
		// Remove all files from index
		files, err := sp.gitOps.ListFiles(subrepoDir)
		if err == nil {
			for _, file := range files {
				if file != "" {
					sp.gitOps.Remove(file)
				}
			}
		}
		return nil
	}
	
	// 创建当前索引状态的备份
	// Create backup of current index state
	backupStart := time.Now()
	sp.logger.Debug("创建安全备份 / Creating safety backup")
	indexBackup, err := sp.gitOps.ListFiles("-s", subrepoDir)
	sp.logger.Debug("备份完成，耗时 / Backup complete, took: %v", time.Since(backupStart))
	if err != nil {
		sp.logger.Warn("Failed to create index backup: %v", err)
	}
	
	// 收集需要处理的文件
	// Collect files to process
	collectStart := time.Now()
	sp.logger.Debug("高效收集文件 / Efficiently collecting files")
	sp.logger.Debug("  ↳ 扫描目录 / Scanning directory: %s", subrepoDir)
	
	// 收集工作文件（排除虚拟环境）
	// Collect work files (excluding virtual environments)
	workFiles, excludedDirs, err := sp.collectWorkFiles(subrepoDir)
	if err != nil {
		return fmt.Errorf("failed to collect work files: %v", err)
	}
	
	if len(excludedDirs) > 0 {
		sp.logger.Info("排除了 %d 个虚拟环境目录 / Excluded %d virtual env directories", len(excludedDirs), len(excludedDirs))
		if len(excludedDirs) <= 20 {
			for _, dir := range excludedDirs {
				sp.logger.Debug("  • %s", dir)
			}
		} else {
			sp.logger.Debug("  • %s ... (共%d个)", excludedDirs[0], len(excludedDirs))
		}
	}
	
	sp.logger.Info("收集到 %d 个工作文件 / Collected %d work files", len(workFiles), len(workFiles))
	sp.logger.Debug("工作文件收集完成，耗时 / Work files collected, took: %v", time.Since(collectStart))
	if err != nil {
		return fmt.Errorf("failed to collect work files: %v", err)
	}
	
	gitCollectStart := time.Now()
	gitFiles, err := sp.collectGitFiles(subrepoDir)
	if err != nil {
		return fmt.Errorf("failed to collect git files: %v", err)
	}
	sp.logger.Debug("Git文件收集完成，耗时 / Git files collected, took: %v", time.Since(gitCollectStart))
	
	totalFiles := len(workFiles) + len(gitFiles)
	sp.logger.Debug("高速处理 %d 个文件 / High-speed processing %d files", totalFiles, totalFiles)
	
	// 智能文件分类处理
	// Intelligent file classification processing
	operations := make([]fileOperation, 0, totalFiles)
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	// 创建工作池
	// Create worker pool
	sem := make(chan struct{}, sp.cfg.MaxParallelWorkers)
	
	// 分类文件（使用配置的阈值）
	// Classify files (using configured thresholds)
	smallFiles := []string{}   // < SmallFileThreshold
	mediumFiles := []string{}  // SmallFileThreshold - MediumFileThreshold
	largeFiles := []string{}   // > MediumFileThreshold

	for _, fp := range workFiles {
		if info, err := os.Stat(fp); err == nil {
			fileSize := info.Size()
			if fileSize < sp.cfg.SmallFileThreshold {
				smallFiles = append(smallFiles, fp)
			} else if fileSize < sp.cfg.MediumFileThreshold {
				mediumFiles = append(mediumFiles, fp)
			} else {
				largeFiles = append(largeFiles, fp)
			}
		}
	}
	
	sp.logger.Debug("文件分类 / File classification: 小文件 %d, 中文件 %d, 大文件 %d / small %d, medium %d, large %d",
		len(smallFiles), len(mediumFiles), len(largeFiles), len(smallFiles), len(mediumFiles), len(largeFiles))
	
	// 处理小文件（并行）
	// Process small files (parallel)
	if len(smallFiles) > 0 {
		sp.logger.Debug("并行处理小文件 / Parallel processing small files")
		smallStart := time.Now()
		
		for _, filePath := range smallFiles {
			wg.Add(1)
			go func(fp string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				
				if op, err := sp.processWorkFile(fp); err == nil {
					mu.Lock()
					operations = append(operations, op)
					mu.Unlock()
				}
			}(filePath)
		}
		wg.Wait()
		sp.logger.Debug("小文件处理完成，耗时 / Small files processed, took: %v", time.Since(smallStart))
	}
	
	// 处理中文件（串行）
	// Process medium files (serial)
	if len(mediumFiles) > 0 {
		sp.logger.Debug("串行处理中文件 / Serial processing medium files")
		mediumStart := time.Now()
		
		for _, filePath := range mediumFiles {
			if op, err := sp.processWorkFile(filePath); err == nil {
				operations = append(operations, op)
			}
		}
		sp.logger.Debug("中文件处理完成，耗时 / Medium files processed, took: %v", time.Since(mediumStart))
	}
	
	// 处理大文件（特殊处理）
	// Process large files (special handling)
	if len(largeFiles) > 0 {
		sp.logger.Warn("特殊处理大文件 / Special processing large files: %d 个 / %d files", len(largeFiles), len(largeFiles))
		largeStart := time.Now()
		
		for _, filePath := range largeFiles {
			fileInfo, _ := os.Stat(filePath)
			fileSize := fileInfo.Size()
			sp.logger.Info("处理大文件 / Processing large file: %s (%.2f MB)", 
				filePath, float64(fileSize)/1024/1024)
			
			if op, err := sp.processWorkFile(filePath); err == nil {
				operations = append(operations, op)
			}
		}
		sp.logger.Info("大文件处理完成，耗时 / Large files processed, took: %v", time.Since(largeStart))
	}
	
	// 处理.git文件（并行）
	// Process .git files (parallel)
	if len(gitFiles) > 0 {
		sp.logger.Debug("并行转换 git 目录 / Parallel converting git directory")
		gitStart := time.Now()
		
		for _, filePath := range gitFiles {
			wg.Add(1)
			go func(fp string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				
				if op, err := sp.processGitFile(fp, subrepoDir); err == nil {
					mu.Lock()
					operations = append(operations, op)
					mu.Unlock()
				}
			}(filePath)
		}
		wg.Wait()
		sp.logger.Debug("Git文件转换完成，耗时 / Git files converted, took: %v", time.Since(gitStart))
	}
	
	// 等待所有任务完成
	// Wait for all tasks to complete
	processStart := time.Now()
	wg.Wait()
	processDuration := time.Since(processStart)
	
	sp.logger.Info("已准备 %d 个文件操作 / Prepared %d file operations", len(operations), len(operations))
	sp.logger.Debug("并行处理耗时 / Parallel processing took: %v (速度 / speed: %.0f files/sec)", 
		processDuration, float64(totalFiles)/processDuration.Seconds())
	
	// 安全的原子性应用所有变更
	// Safely apply all changes atomically
	if len(operations) > 0 {
		batchStart := time.Now()
		sp.logger.Debug("安全原子性应用变更 / Safely applying changes atomically")
		
		// 批量应用所有操作（使用单个git update-index --index-info命令）
		// Batch apply all operations (using single git update-index --index-info command)
		if err := sp.batchUpdateIndex(operations); err != nil {
			sp.logger.Error("Failed to batch update index: %v", err)
			return fmt.Errorf("failed to batch update index: %v", err)
		}
		sp.logger.Debug("批量更新耗时 / Batch update took: %v", time.Since(batchStart))
		
		// 清理不再存在的文件
		// Clean up files that no longer exist
		cleanupStart := time.Now()
		if len(indexBackup) > 0 {
			sp.logger.Debug("开始清理不存在的文件 / Starting cleanup of non-existent files: %d 个索引条目 / %d index entries", 
				len(indexBackup), len(indexBackup))
			
			operationPaths := make(map[string]bool)
			for _, op := range operations {
				operationPaths[op.path] = true
			}
			
			filesToRemove := []string{}
			
			for _, line := range indexBackup {
				if line == "" {
					continue
				}
				
				// 解析索引行: mode hash stage path
				// Parse index line: mode hash stage path
				parts := strings.Fields(line)
				if len(parts) < 4 {
					continue
				}
				
				path := strings.Join(parts[3:], " ")
				
				// 去除引号和解码八进制转义（处理包含特殊字符的路径）
				// Remove quotes and decode octal escapes (handle paths with special characters)
				path = unquoteGitPath(path)
				
				// 检查文件是否应该删除
				// Check if file should be deleted
				shouldDelete := false
				
				// 如果是.git路径，检查对应的gitdir路径
				// If it's a .git path, check corresponding gitdir path
				if strings.Contains(path, "/.git/") {
					gitdirPath := strings.Replace(path, "/.git/", "/gitdir/", 1)
					if !operationPaths[gitdirPath] {
						fullPath := filepath.Join(sp.cfg.RepoRoot, path)
						if _, err := os.Stat(fullPath); os.IsNotExist(err) {
							shouldDelete = true
						}
					}
				} else {
					// 对于非.git文件，检查文件是否存在
					// For non-.git files, check if file exists
					if !operationPaths[path] {
						fullPath := filepath.Join(sp.cfg.RepoRoot, path)
						if _, err := os.Stat(fullPath); os.IsNotExist(err) {
							shouldDelete = true
						}
					}
				}
				
				if shouldDelete {
					filesToRemove = append(filesToRemove, path)
				}
			}
			
			// 批量删除
			// Batch remove
			if len(filesToRemove) > 0 {
				sp.logger.Debug("批量删除 %d 个不存在的文件 / Batch removing %d non-existent files", 
					len(filesToRemove), len(filesToRemove))
				
				// 使用单个git rm命令批量删除
				// Use single git rm command for batch removal
				if err := sp.batchRemoveFiles(filesToRemove); err != nil {
					sp.logger.Warn("Failed to batch remove files: %v", err)
				}
			}
			
			sp.logger.Debug("清理完成，耗时 / Cleanup complete, took: %v", time.Since(cleanupStart))
		}
		
		// 确保gitdir目录结构存在，并从索引检出文件到工作目录
		// Ensure gitdir directory structure exists and checkout files from index to working directory
		// 与 Shell 版本保持一致：索引中的 gitdir 文件需要实际存在于工作目录
		// Shell-compatible: gitdir files in index should also exist in working directory
		if len(gitFiles) > 0 {
			sp.logger.Debug("创建gitdir目录结构并检出文件 / Creating gitdir directory structure and checking out files")
			
			// 获取该子仓库的所有 gitdir 文件
			// Get all gitdir files for this subrepo
			relSubrepoDir, _ := filepath.Rel(sp.cfg.RepoRoot, subrepoDir)
			gitdirPrefix := filepath.Join(relSubrepoDir, "gitdir")
			gitdirFiles, err := sp.gitOps.ListFiles(gitdirPrefix)
			if err == nil && len(gitdirFiles) > 0 {
				for _, gitdirFile := range gitdirFiles {
					if gitdirFile == "" {
						continue
					}
					
					// 创建目录结构
					// Create directory structure
					fullPath := filepath.Join(sp.cfg.RepoRoot, gitdirFile)
					if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
						sp.logger.Debug("  ↳ 创建目录失败 / Failed to create directory: %v", err)
						continue
					}
					
					// 从索引检出文件内容 (git show :path)
					// Checkout file content from index (git show :path)
					cmd := exec.Command("git", "show", ":"+gitdirFile)
					cmd.Dir = sp.cfg.RepoRoot
					output, err := cmd.Output()
					if err != nil {
						sp.logger.Debug("  ↳ 检出失败 / Checkout failed: %s, %v", gitdirFile, err)
						continue
					}
					
					// 写入文件
					// Write file
					if err := os.WriteFile(fullPath, output, 0644); err != nil {
						sp.logger.Debug("  ↳ 写入失败 / Write failed: %s, %v", gitdirFile, err)
					}
				}
				sp.logger.Debug("  ✓ 已检出 %d 个 gitdir 文件 / Checked out %d gitdir files", len(gitdirFiles), len(gitdirFiles))
			}
		}
		
		totalDuration := time.Since(startTime)
		cacheSize := sp.hashCache.Size()
		sp.logger.Info("高性能安全重建完成 / High-performance safe rebuild complete: %s (总耗时 / total: %v, 缓存 / cache: %d)", 
			subrepoName, totalDuration, cacheSize)
	} else {
		sp.logger.Warn("无文件需要处理 / No files to process: %s", subrepoName)
	}
	
	return nil
}

// collectWorkFiles 收集工作文件（排除虚拟环境）
// Collects work files (excluding virtual environments)
func (sp *SubrepoProcessor) collectWorkFiles(subrepoDir string) ([]string, []string, error) {
	var files []string
	var excludedDirs []string
	
	err := filepath.Walk(subrepoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 跳过.git目录
		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		
		// 跳过虚拟环境目录
		// Skip virtual environment directories
		if info.IsDir() {
			for _, pattern := range config.VirtualEnvExcludePatterns {
				if info.Name() == pattern {
					relPath, _ := filepath.Rel(sp.cfg.RepoRoot, path)
					excludedDirs = append(excludedDirs, relPath)
					sp.logger.Debug("  ✗ 排除虚拟环境 / Excluding venv: %s", relPath)
					return filepath.SkipDir
				}
			}
		}
		
		// 只收集文件
		// Only collect files
		if !info.IsDir() {
			files = append(files, path)
		}
		
		return nil
	})
	
	return files, excludedDirs, err
}

// collectGitFiles 收集.git目录中的文件
// Collects files in .git directory
func (sp *SubrepoProcessor) collectGitFiles(subrepoDir string) ([]string, error) {
	gitDir := filepath.Join(subrepoDir, ".git")
	
	// 检查.git目录是否存在
	// Check if .git directory exists
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return []string{}, nil
	}
	
	var files []string
	
	err := filepath.Walk(gitDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 只收集文件
		// Only collect files
		if !info.IsDir() {
			files = append(files, path)
		}
		
		return nil
	})
	
	return files, err
}

// processWorkFile 处理工作文件
// Processes a work file
func (sp *SubrepoProcessor) processWorkFile(filePath string) (fileOperation, error) {
	relPath, _ := filepath.Rel(sp.cfg.RepoRoot, filePath)
	sp.logger.Debug("处理工作文件 / Processing work file: %s", relPath)
	
	info, err := os.Stat(filePath)
	if err != nil {
		sp.logger.Warn("获取文件信息失败 / Failed to stat file: %s, error: %v", relPath, err)
		return fileOperation{}, err
	}
	
	mode := "100644"
	if info.Mode()&0111 != 0 {
		mode = "100755"
		sp.logger.Debug("  ↳ 可执行文件 / Executable file: mode=%s", mode)
	}
	
	// 尝试从缓存获取hash
	// Try to get hash from cache
	var hash string
	if cachedHash, ok := sp.hashCache.Get(filePath, info.ModTime(), info.Size()); ok {
		hash = cachedHash
		sp.logger.Debug("  ✓ 使用缓存 / Using cache (hash: %s)", hash[:8]+"...")
	} else {
		// 计算hash
		// Compute hash
		hash, err = sp.gitOps.HashObject(filePath)
		if err != nil {
			sp.logger.Warn("计算hash失败 / Hash calculation failed: %s, error: %v", relPath, err)
			return fileOperation{}, err
		}
		
		sp.logger.Debug("  ↻ 计算hash / Computed hash: %s", hash[:8]+"...")
		
		// 缓存hash
		// Cache hash
		sp.hashCache.Set(filePath, hash, info.ModTime(), info.Size())
	}
	
	sp.logger.Debug("  ✓ 已加入操作队列 / Added to operation queue")
	
	return fileOperation{
		mode: mode,
		hash: hash,
		path: relPath,
	}, nil
}

// processGitFile 处理.git文件
// Processes a .git file
func (sp *SubrepoProcessor) processGitFile(filePath, subrepoDir string) (fileOperation, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return fileOperation{}, err
	}
	
	mode := "100644"
	if info.Mode()&0111 != 0 {
		mode = "100755"
	}
	
	hash, err := sp.gitOps.HashObject(filePath)
	if err != nil {
		return fileOperation{}, err
	}
	
	// 转换路径: .git -> gitdir
	// Convert path: .git -> gitdir
	relPath, _ := filepath.Rel(sp.cfg.RepoRoot, filePath)
	targetPath := strings.Replace(relPath, "/.git/", "/gitdir/", 1)
	
	return fileOperation{
		mode: mode,
		hash: hash,
		path: targetPath,
	}, nil
}

// CleanOrphanedGitdirs 清理孤儿gitdir目录
// Cleans orphaned gitdir directories
func (sp *SubrepoProcessor) CleanOrphanedGitdirs() error {
	sp.logger.Debug("阶段1.5：清理孤儿gitdir目录 / Phase 1.5: Cleaning orphaned gitdir directories")
	
	// 方法1: 检查文件系统中的孤儿gitdir
	// Method 1: Check orphaned gitdir in filesystem
	for _, baseDir := range sp.cfg.SubrepoBaseDirs {
		basePath := filepath.Join(sp.cfg.RepoRoot, baseDir)
		
		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			continue
		}
		
		// 查找所有gitdir目录
		// Find all gitdir directories
		filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			
			if info.IsDir() && info.Name() == "gitdir" {
				parentDir := filepath.Dir(path)
				
				// 检查父目录是否只包含gitdir
				// Check if parent directory only contains gitdir
				entries, err := os.ReadDir(parentDir)
				if err != nil {
					return nil
				}
				
				onlyGitdir := true
				for _, entry := range entries {
					if entry.Name() != "gitdir" {
						onlyGitdir = false
						break
					}
				}
				
				if onlyGitdir {
					sp.logger.Info("发现孤儿gitdir目录 / Found orphaned gitdir: %s", filepath.Base(parentDir))
					sp.logger.Info("删除孤儿目录 / Removing orphaned directory: %s", parentDir)
					sp.logger.Debug("  ↳ 完整路径 / Full path: %s", parentDir)
					
					if err := os.RemoveAll(parentDir); err != nil {
						sp.logger.Error("删除失败 / Remove failed: %v", err)
						return err
					}
					
					sp.logger.Debug("  ✓ 目录已删除 / Directory removed")
					
					relPath, _ := filepath.Rel(sp.cfg.RepoRoot, parentDir)
					sp.gitOps.Add(relPath)
				}
			}
			
			return nil
		})
	}
	
	// 方法2: 检查Git索引中的孤儿gitdir文件
	// Method 2: Check orphaned gitdir files in git index
	sp.logger.Debug("检查Git索引中的孤儿gitdir文件 / Checking orphaned gitdir files in Git index")
	
	files, err := sp.gitOps.ListFiles("--cached")
	if err != nil {
		return err
	}
	
	processedParents := make(map[string]bool)
	orphanedFiles := []string{}
	
	for _, file := range files {
		if !strings.Contains(file, "/gitdir/") {
			continue
		}
		
		// 提取父目录路径
		// Extract parent directory path
		parts := strings.Split(file, "/gitdir/")
		if len(parts) < 2 {
			continue
		}
		
		parentDir := parts[0]
		
		// 检查是否已处理过此父目录
		// Check if this parent directory has been processed
		if processedParents[parentDir] {
			continue
		}
		
		// 检查父目录是否存在
		// Check if parent directory exists
		parentPath := filepath.Join(sp.cfg.RepoRoot, parentDir)
		if _, err := os.Stat(parentPath); os.IsNotExist(err) {
			sp.logger.Warn("发现Git索引中的孤儿gitdir父目录 / Found orphaned gitdir parent directory: %s", parentDir)
			processedParents[parentDir] = true
			
			// 收集该父目录下的所有gitdir文件
			// Collect all gitdir files under this parent directory
			for _, f := range files {
				if strings.HasPrefix(f, parentDir+"/gitdir/") {
					orphanedFiles = append(orphanedFiles, f)
				}
			}
		}
	}
	
	// 批量删除孤儿gitdir文件
	// Batch delete orphaned gitdir files
	if len(orphanedFiles) > 0 {
		sp.logger.Info("清理 %d 个孤儿gitdir文件 / Cleaning %d orphaned gitdir files", len(orphanedFiles), len(orphanedFiles))
		
		for _, file := range orphanedFiles {
			sp.logger.Debug("  ↳ 删除 / Removing: %s", file)
			if err := sp.gitOps.Remove(file); err != nil {
				sp.logger.Error("删除失败 / Remove failed: %s, error: %v", file, err)
			}
		}
		
		sp.logger.Info("✓ 孤儿文件清理完成 / Orphaned files cleanup complete")
	}
	
	return nil
}

// batchUpdateIndex 批量更新Git索引（带锁检测和重试机制）
// Batch updates git index (with lock detection and retry mechanism)
func (sp *SubrepoProcessor) batchUpdateIndex(operations []fileOperation) error {
	if len(operations) == 0 {
		return nil
	}
	
	// 构建索引信息字符串
	// Build index info string
	// 格式：mode hash path
	// Format: mode hash path
	var indexInfo strings.Builder
	for _, op := range operations {
		indexInfo.WriteString(fmt.Sprintf("%s %s\t%s\n", op.mode, op.hash, op.path))
	}
	
	// 最大重试次数（使用配置值）
	// Maximum retry count (using config values)
	maxRetries := sp.cfg.IndexUpdateMaxRetries
	retryDelay := sp.cfg.IndexUpdateRetryDelay
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// 检查并清理过期的 index.lock 文件
		// Check and clean stale index.lock file
		lockPath := filepath.Join(sp.cfg.RepoRoot, ".git", "index.lock")
		if info, err := os.Stat(lockPath); err == nil {
			lockAge := time.Since(info.ModTime())
			sp.logger.Debug("[LOCK检测] index.lock 存在，年龄: %v / index.lock exists, age: %v", lockAge, lockAge)
			
			// 如果 lock 文件超过配置时间，认为是残留文件
			// If lock file is older than configured time, consider it stale
			if lockAge > sp.cfg.LockFileMaxAge {
				sp.logger.Warn("[LOCK清理] 清理过期的 index.lock (年龄: %v) / Cleaning stale index.lock (age: %v)", lockAge, lockAge)
				if err := os.Remove(lockPath); err != nil {
					sp.logger.Warn("[LOCK清理] 清理失败 / Cleanup failed: %v", err)
				} else {
					sp.logger.Info("[LOCK清理] 过期 lock 文件已清理 / Stale lock file cleaned")
				}
			} else {
				// lock 文件较新，可能有其他进程正在使用
				// Lock file is recent, another process might be using it
				sp.logger.Debug("[LOCK等待] lock 文件较新，等待释放... / Lock file is recent, waiting for release...")
				time.Sleep(retryDelay)
				continue
			}
		}
		
		// 使用单个git update-index --index-info命令批量更新
		// Use single git update-index --index-info command for batch update
		sp.logger.Debug("[INDEX更新] 尝试 %d/%d: 批量更新 %d 个文件 / Attempt %d/%d: Batch updating %d files", 
			attempt, maxRetries, len(operations), attempt, maxRetries, len(operations))
		
		cmd := exec.Command("git", "update-index", "--index-info")
		cmd.Dir = sp.cfg.RepoRoot
		cmd.Stdin = strings.NewReader(indexInfo.String())
		
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		
		if err := cmd.Run(); err != nil {
			stderrStr := stderr.String()
			
			// 检查是否是 lock 文件冲突
			// Check if it's a lock file conflict
			if strings.Contains(stderrStr, "index.lock") || strings.Contains(stderrStr, "文件已存在") {
				sp.logger.Warn("[INDEX更新] 尝试 %d/%d 失败: index.lock 冲突 / Attempt %d/%d failed: index.lock conflict", 
					attempt, maxRetries, attempt, maxRetries)
				
				if attempt < maxRetries {
					sp.logger.Info("[INDEX更新] 等待 %v 后重试... / Waiting %v before retry...", retryDelay, retryDelay)
					time.Sleep(retryDelay)
					// 增加重试延迟（指数退避）
					// Increase retry delay (exponential backoff)
					retryDelay = retryDelay * 2
					continue
				}
			}
			
			return fmt.Errorf("git update-index --index-info failed: %v, stderr: %s", err, stderrStr)
		}
		
		// 成功
		// Success
		sp.logger.Debug("[INDEX更新] 成功！批量更新了 %d 个文件的索引 / Success! Batch updated index for %d files", len(operations), len(operations))
		return nil
	}
	
	return fmt.Errorf("git update-index failed after %d retries", maxRetries)
}

// batchRemoveFiles 批量删除文件
// Batch removes files
func (sp *SubrepoProcessor) batchRemoveFiles(files []string) error {
	if len(files) == 0 {
		return nil
	}
	
	sp.logger.Info("批量删除 %d 个文件 / Batch removing %d files", len(files), len(files))
	
	// 分批处理（使用配置的批次大小）
	// Process in batches (using configured batch size)
	batchSize := sp.cfg.BatchSize
	sp.logger.Debug("  ↳ 批次大小 / Batch size: %d", batchSize)
	
	successCount := 0
	failedFiles := []string{}
	
	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}
		
		batch := files[i:end]
		batchNum := (i / batchSize) + 1
		totalBatches := (len(files) + batchSize - 1) / batchSize
		
		sp.logger.Debug("  ↳ 处理批次 %d/%d / Processing batch %d/%d (%d files)", 
			batchNum, totalBatches, batchNum, totalBatches, len(batch))
		
		// 详细记录每个文件 (批次小于等于10个文件时)
		// Detailed logging for small batches
		if len(batch) <= 10 {
			for _, f := range batch {
				sp.logger.Debug("    • %s", f)
			}
		} else {
			sp.logger.Debug("    • %s ... (共%d个文件)", batch[0], len(batch))
		}
		
		// 使用git rm批量删除
		// Use git rm to batch remove files
		cmd := exec.Command("git", append([]string{"rm", "--cached", "--ignore-unmatch", "--"}, batch...)...)
		cmd.Dir = sp.cfg.RepoRoot
		
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		
		if err := cmd.Run(); err != nil {
			sp.logger.Debug("批次 %d 删除失败 (已忽略) / Batch %d remove failed (ignored): %v", batchNum, batchNum, err)
			if stderr.Len() > 0 {
				sp.logger.Debug("  ↳ stderr: %s", stderr.String())
			}
			failedFiles = append(failedFiles, batch...)
		} else {
			successCount += len(batch)
			sp.logger.Debug("  ✓ 批次 %d 完成 / Batch %d complete", batchNum, batchNum)
		}
	}
	
	// 总结
	// Summary
	if len(failedFiles) > 0 {
		sp.logger.Warn("批量删除完成，但有 %d 个文件失败 / Batch remove complete, but %d files failed", len(failedFiles), len(failedFiles))
		sp.logger.Debug("失败文件列表 / Failed files:")
		for _, f := range failedFiles {
			sp.logger.Debug("  • %s", f)
		}
	} else {
		sp.logger.Info("✓ 批量删除完成 / Batch remove complete: %d files", successCount)
	}
	
	return nil
}

// unquoteGitPath 去除Git引号并解码八进制转义序列
// Removes Git quotes and decodes octal escape sequences
// Git对包含特殊字符（如中文、空格等）的路径会添加引号并使用八进制转义
// Git adds quotes and uses octal escapes for paths with special characters (like Chinese, spaces, etc.)
// 例如 / Example: "debian/data/git/dev/\345\220\216\347\253\257" -> debian/data/git/dev/后端
func unquoteGitPath(path string) string {
	// 检查是否被引号包围
	// Check if surrounded by quotes
	if len(path) >= 2 && path[0] == '"' && path[len(path)-1] == '"' {
		// 去除引号
		// Remove quotes
		path = path[1 : len(path)-1]
		
		// 解码八进制转义序列（如 \345\220\216 -> 后）
		// Decode octal escape sequences (e.g., \345\220\216 -> 后)
		var result strings.Builder
		i := 0
		for i < len(path) {
			if path[i] == '\\' && i+3 < len(path) {
				// 检查是否是八进制转义序列（\ddd 格式）
				// Check if it's an octal escape sequence (\ddd format)
				if isOctalDigit(path[i+1]) && isOctalDigit(path[i+2]) && isOctalDigit(path[i+3]) {
					// 解析八进制值
					// Parse octal value
					octalStr := path[i+1 : i+4]
					if val, err := strconv.ParseInt(octalStr, 8, 32); err == nil {
						result.WriteByte(byte(val))
						i += 4
						continue
					}
				}
				// 处理其他转义序列
				// Handle other escape sequences
				if i+1 < len(path) {
					switch path[i+1] {
					case 'n':
						result.WriteByte('\n')
						i += 2
						continue
					case 't':
						result.WriteByte('\t')
						i += 2
						continue
					case '\\':
						result.WriteByte('\\')
						i += 2
						continue
					case '"':
						result.WriteByte('"')
						i += 2
						continue
					}
				}
			}
			result.WriteByte(path[i])
			i++
		}
		return result.String()
	}
	return path
}

// isOctalDigit 检查字符是否是八进制数字
// Checks if a character is an octal digit
func isOctalDigit(c byte) bool {
	return c >= '0' && c <= '7'
}
