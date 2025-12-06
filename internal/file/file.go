package file

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/find-xposed-magisk/git-sync/internal/config"
	"github.com/find-xposed-magisk/git-sync/internal/git"
	"github.com/find-xposed-magisk/git-sync/internal/logger"
)

// FileProcessor 文件处理器
// File processor
type FileProcessor struct {
	cfg     *config.Config
	gitOps  *git.GitOps
	logger  *logger.Logger
	ignoreFile string
}

// NewFileProcessor 创建文件处理器
// Creates a new file processor
func NewFileProcessor(cfg *config.Config, gitOps *git.GitOps, log *logger.Logger) *FileProcessor {
	return &FileProcessor{
		cfg:     cfg,
		gitOps:  gitOps,
		logger:  log,
		ignoreFile: filepath.Join(cfg.RepoRoot, cfg.IgnoreFileName),
	}
}

// StageFile 暂存单个文件（带大小检测）
// Stages a single file (with size detection)
func (fp *FileProcessor) StageFile(filePath string) error {
	// 检查文件是否存在
	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，跳过 / File doesn't exist, skip
		}
		return fmt.Errorf("failed to stat file %s: %v", filePath, err)
	}
	
	fileSize := fileInfo.Size()
	
	// 检查是否超过忽略阈值
	// Check if exceeds ignore threshold
	if fileSize > fp.cfg.IgnoreSizeThresholdBytes {
		fp.logger.Warn("已忽略 (大小 > %dB) / IGNORED (size > %dB): 文件 '%s' 过大，添加到 %s / File '%s' is too large. Adding to %s",
			fp.cfg.IgnoreSizeThresholdBytes, fp.cfg.IgnoreSizeThresholdBytes,
			filePath, fp.cfg.IgnoreFileName, filePath, fp.cfg.IgnoreFileName)
		
		// 添加到忽略文件
		// Add to ignore file
		if err := fp.addToIgnoreFile(filePath); err != nil {
			return err
		}
		
		// 暂存忽略文件
		// Stage ignore file
		if err := fp.gitOps.Add(fp.ignoreFile); err != nil {
			fp.logger.Warn("Failed to stage ignore file: %v", err)
		}
		
		return nil
	}
	
	// 检查是否超过LFS阈值
	// Check if exceeds LFS threshold
	if fileSize > fp.cfg.LFSSizeThresholdBytes {
		fp.logger.Warn("LFS 检测 (大小 > %dB) / LFS DETECTED (size > %dB): 使用 Git LFS 追踪 '%s' / Tracking '%s' with Git LFS",
			fp.cfg.LFSSizeThresholdBytes, fp.cfg.LFSSizeThresholdBytes, filePath, filePath)
		
		// 使用LFS追踪
		// Track with LFS
		if err := fp.gitOps.LFSTrack(filePath); err != nil {
			fp.logger.Warn("Failed to track with LFS: %v", err)
		}
		
		// 暂存.gitattributes
		// Stage .gitattributes
		gitattributesPath := filepath.Join(fp.cfg.RepoRoot, ".gitattributes")
		if err := fp.gitOps.Add(gitattributesPath); err != nil {
			fp.logger.Warn("Failed to stage .gitattributes: %v", err)
		}
	}
	
	// 转换为相对路径
	// Convert to relative path
	relPath, err := filepath.Rel(fp.cfg.RepoRoot, filePath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %v", err)
	}
	
	// 直接使用git add命令（更简单可靠）
	// Use git add command directly (simpler and more reliable)
	if err := fp.gitOps.Add(relPath); err != nil {
		return fmt.Errorf("failed to add file %s: %v", relPath, err)
	}
	
	fp.logger.Debug("已暂存文件 / Staged file: %s", relPath)
	
	return nil
}

// addToIgnoreFile 添加文件路径到忽略文件
// Adds file path to ignore file
func (fp *FileProcessor) addToIgnoreFile(filePath string) error {
	// 检查是否已存在
	// Check if already exists
	exists, err := fp.isInIgnoreFile(filePath)
	if err != nil {
		return err
	}
	
	if exists {
		return nil // 已存在，跳过 / Already exists, skip
	}
	
	// 追加到文件
	// Append to file
	f, err := os.OpenFile(fp.ignoreFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open ignore file: %v", err)
	}
	defer f.Close()
	
	if _, err := f.WriteString(filePath + "\n"); err != nil {
		return fmt.Errorf("failed to write to ignore file: %v", err)
	}
	
	return nil
}

// isInIgnoreFile 检查文件路径是否在忽略文件中
// Checks if file path is in ignore file
func (fp *FileProcessor) isInIgnoreFile(filePath string) (bool, error) {
	f, err := os.Open(fp.ignoreFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()
	
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == filePath {
			return true, nil
		}
	}
	
	return false, scanner.Err()
}

// HandleEmptyDirectories 处理空目录
// Handles empty directories
func (fp *FileProcessor) HandleEmptyDirectories() error {
	fp.logger.Debug("部分C：检查并处理空目录 / Part C: Checking and handling empty directories")
	
	// 构建排除路径
	// Build exclude paths
	excludePaths := []string{".git"}
	excludePaths = append(excludePaths, fp.cfg.SubrepoBaseDirs...)
	
	// 遍历目录查找空目录
	// Walk directory tree to find empty directories
	err := filepath.Walk(fp.cfg.RepoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 跳过非目录
		// Skip non-directories
		if !info.IsDir() {
			return nil
		}
		
		// 跳过排除路径
		// Skip excluded paths
		relPath, _ := filepath.Rel(fp.cfg.RepoRoot, path)
		for _, exclude := range excludePaths {
			if strings.HasPrefix(relPath, exclude) {
				return filepath.SkipDir
			}
		}
		
		// 检查目录是否为空
		// Check if directory is empty
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		
		if len(entries) == 0 {
			// 创建占位文件
			// Create placeholder file
			placeholderPath := filepath.Join(path, fp.cfg.EmptyDirPlaceholderFile)
			fp.logger.Debug("在空目录中创建占位文件 / Creating placeholder in empty directory: %s", placeholderPath)
			
			if err := os.WriteFile(placeholderPath, []byte{}, 0644); err != nil {
				fp.logger.Warn("Failed to create placeholder: %v", err)
				return nil
			}
			
			// 暂存占位文件
			// Stage placeholder file
			if err := fp.gitOps.Add(placeholderPath); err != nil {
				fp.logger.Warn("Failed to stage placeholder: %v", err)
			}
		}
		
		return nil
	})
	
	return err
}

// IsInSpecialRepo 检查路径是否在特殊仓库中
// Checks if path is in a special repository
func (fp *FileProcessor) IsInSpecialRepo(path string) bool {
	for _, baseDir := range fp.cfg.SubrepoBaseDirs {
		if strings.HasPrefix(path, baseDir+"/") || path == baseDir {
			return true
		}
	}
	return false
}
