package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/find-xposed-magisk/git-sync/internal/config"
	"github.com/find-xposed-magisk/git-sync/internal/logger"
)

// GitOps Git操作封装
// Git operations wrapper
type GitOps struct {
	cfg    *config.Config
	logger *logger.Logger
}

// NewGitOps 创建Git操作实例
// Creates a new GitOps instance
func NewGitOps(cfg *config.Config, log *logger.Logger) *GitOps {
	return &GitOps{
		cfg:    cfg,
		logger: log,
	}
}

// execGitCommand 执行Git命令
// Executes a git command
func (g *GitOps) execGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.cfg.RepoRoot
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %v, stderr: %s", 
			strings.Join(args, " "), err, stderr.String())
	}
	
	return strings.TrimSpace(stdout.String()), nil
}

// EnsureDependencies 确保依赖已安装
// Ensures dependencies are installed
func (g *GitOps) EnsureDependencies() error {
	g.logger.Phase("确保依赖已安装并初始化LFS / Ensuring Dependencies & Initializing LFS")
	
	// 检查git和git-lfs是否已安装
	// Check if git and git-lfs are installed
	for _, cmd := range []string{"git", "git-lfs"} {
		if _, err := exec.LookPath(cmd); err != nil {
			g.logger.Warn("依赖 '%s' 未找到，尝试安装 / Dependency '%s' not found, attempting to install", cmd, cmd)
			
			// 尝试安装
			// Attempt to install
			installCmd := exec.Command("apt-get", "install", "-y", cmd)
			if err := installCmd.Run(); err != nil {
				return fmt.Errorf("failed to install %s: %v", cmd, err)
			}
		}
	}
	
	g.logger.Info("所有依赖已满足 / All dependencies are satisfied")
	
	// 初始化Git LFS
	// Initialize Git LFS
	if _, err := g.execGitCommand("lfs", "install"); err != nil {
		return fmt.Errorf("failed to initialize git-lfs: %v", err)
	}
	
	// 追踪预定义的大文件模式
	// Track predefined large file patterns
	if len(g.cfg.LFSTrackPatterns) > 0 {
		g.logger.Debug("追踪预定义的大文件模式 / Tracking predefined large file patterns")
		for _, pattern := range g.cfg.LFSTrackPatterns {
			if _, err := g.execGitCommand("lfs", "track", pattern); err != nil {
				g.logger.Warn("Failed to track LFS pattern %s: %v", pattern, err)
			}
		}
		// 暂存.gitattributes
		// Stage .gitattributes
		if _, err := g.execGitCommand("add", ".gitattributes"); err != nil {
			g.logger.Warn("Failed to stage .gitattributes: %v", err)
		}
	}
	
	// 确保.gitignore_nopush被追踪
	// Ensure .gitignore_nopush is tracked
	ignoreFilePath := filepath.Join(g.cfg.RepoRoot, g.cfg.IgnoreFileName)
	if err := os.WriteFile(ignoreFilePath, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to create ignore file: %v", err)
	}
	
	// 检查文件是否已被追踪
	// Check if file is already tracked
	if _, err := g.execGitCommand("ls-files", "--error-unmatch", ignoreFilePath); err != nil {
		// 文件未被追踪，添加到.gitignore并暂存
		// File not tracked, add to .gitignore and stage
		gitignorePath := filepath.Join(g.cfg.RepoRoot, ".gitignore")
		f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open .gitignore: %v", err)
		}
		defer f.Close()
		
		if _, err := f.WriteString(g.cfg.IgnoreFileName + "\n"); err != nil {
			return fmt.Errorf("failed to write to .gitignore: %v", err)
		}
		
		if _, err := g.execGitCommand("add", gitignorePath); err != nil {
			g.logger.Warn("Failed to stage .gitignore: %v", err)
		}
	}
	
	// 设置diff3冲突样式 (显示共同祖先)
	// Set diff3 conflict style (shows common ancestor)
	if _, err := g.execGitCommand("config", "merge.conflictstyle", "diff3"); err != nil {
		g.logger.Warn("Failed to set merge conflict style: %v", err)
	} else {
		g.logger.Debug("✓ 已启用diff3冲突样式 / diff3 conflict style enabled")
	}
	
	g.logger.Info("--- Git LFS 初始化完成 / Git LFS Initialization Complete ---")
	return nil
}

// HashObject 计算文件的Git对象哈希
// Computes the git object hash for a file
func (g *GitOps) HashObject(filePath string) (string, error) {
	return g.execGitCommand("hash-object", "-w", filePath)
}

// UpdateIndex 更新Git索引
// Updates the git index
func (g *GitOps) UpdateIndex(mode, hash, path string) error {
	_, err := g.execGitCommand("update-index", "--add", "--cacheinfo", mode, hash, path)
	return err
}

// LFSTrack 追踪LFS文件
// Tracks a file with LFS
func (g *GitOps) LFSTrack(filePath string) error {
	_, err := g.execGitCommand("lfs", "track", filePath)
	return err
}

// Add 添加文件到暂存区
// Adds a file to the staging area
func (g *GitOps) Add(filePath string) error {
	_, err := g.execGitCommand("add", "--", filePath)
	return err
}

// AddAll 添加所有变更到暂存区
// Adds all changes to the staging area
func (g *GitOps) AddAll() error {
	_, err := g.execGitCommand("add", "-A")
	return err
}

// Remove 从索引中删除文件
// Removes a file from the index
func (g *GitOps) Remove(filePath string) error {
	_, err := g.execGitCommand("rm", "--cached", "--ignore-unmatch", "--", filePath)
	return err
}

// Commit 提交变更
// Commits changes
func (g *GitOps) Commit(message string) error {
	_, err := g.execGitCommand("commit", "-m", message)
	return err
}

// HasStagedChanges 检查是否有暂存的变更
// Checks if there are staged changes
func (g *GitOps) HasStagedChanges() (bool, error) {
	output, err := g.execGitCommand("diff", "--cached", "--quiet")
	if err != nil {
		// diff --quiet 在有差异时返回非零退出码
		// diff --quiet returns non-zero exit code when there are differences
		if strings.Contains(err.Error(), "exit status 1") {
			return true, nil
		}
		return false, err
	}
	return output != "", nil
}

// Fetch 从远程获取更新
// Fetches updates from remote
func (g *GitOps) Fetch() error {
	g.logger.Debug("正在从远程获取更新 / Fetching updates from remote")
	_, err := g.execGitCommand("fetch", g.cfg.RemoteName)
	return err
}

// parseCorruptRefError 解析损坏引用错误，返回损坏的引用路径列表
// Parses corrupt reference error, returns list of corrupt ref paths
func parseCorruptRefError(errMsg string) []string {
	re := regexp.MustCompile(`bad object (refs/[^\s]+)`)
	var matches []string
	for _, match := range re.FindAllStringSubmatch(errMsg, -1) {
		if len(match) > 1 {
			matches = append(matches, match[1])
		}
	}
	return matches
}

// Push 推送到远程（含自动修复损坏引用）
// Pushes to remote (with auto-fix for corrupt references)
func (g *GitOps) Push() error {
	g.logger.Debug("正在推送到远程 / Pushing to remote")
	_, err := g.execGitCommand("push", g.cfg.RemoteName, g.cfg.BranchName)
	if err == nil || !g.cfg.AutoFixCorruptRefs {
		return err
	}

	// 尝试修复损坏引用后重试
	// Try to fix corrupt refs and retry
	corruptRefs := parseCorruptRefError(err.Error())
	if len(corruptRefs) == 0 {
		return err
	}

	g.logger.Warn("检测到 %d 个损坏的远程引用，尝试自动修复 / Detected %d corrupt remote refs, auto-fixing", len(corruptRefs), len(corruptRefs))
	fixed := false
	for _, ref := range corruptRefs {
		if _, delErr := g.execGitCommand("push", g.cfg.RemoteName, ":"+ref); delErr == nil {
			g.logger.Info("  ✓ 已删除损坏引用 / Deleted corrupt ref: %s", ref)
			fixed = true
		}
	}

	if fixed {
		g.logger.Info("重试推送 / Retrying push")
		_, err = g.execGitCommand("push", g.cfg.RemoteName, g.cfg.BranchName)
	}
	return err
}

// ForcePush 强制推送到远程
// Force pushes to remote
func (g *GitOps) ForcePush() error {
	g.logger.Warn("⚠️ 正在强制推送到远程 / Force pushing to remote")
	_, err := g.execGitCommand("push", "--force", g.cfg.RemoteName, g.cfg.BranchName)
	return err
}

// Pull 从远程拉取
// Pulls from remote
func (g *GitOps) Pull() error {
	g.logger.Debug("正在从远程拉取 / Pulling from remote")
	_, err := g.execGitCommand("pull", "--rebase", g.cfg.RemoteName, g.cfg.BranchName)
	return err
}

// GetRevision 获取提交哈希
// Gets commit hash
func (g *GitOps) GetRevision(ref string) (string, error) {
	return g.execGitCommand("rev-parse", ref)
}

// GetMergeBase 获取共同祖先
// Gets merge base
func (g *GitOps) GetMergeBase(ref1, ref2 string) (string, error) {
	return g.execGitCommand("merge-base", ref1, ref2)
}

// HasUncommittedChanges 检查是否有未提交的变更
// Checks if there are uncommitted changes
func (g *GitOps) HasUncommittedChanges() (bool, error) {
	output, err := g.execGitCommand("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(output)) > 0, nil
}

// HasUnstagedChanges 检查是否有未暂存的变更
// Checks if there are unstaged changes
func (g *GitOps) HasUnstagedChanges() (bool, error) {
	output, err := g.execGitCommand("diff", "--name-only")
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(output)) > 0, nil
}

// Merge 合并分支
// Merges a branch
func (g *GitOps) Merge(branch, message string) error {
	_, err := g.execGitCommand("merge", branch, "--no-edit", "-m", message)
	return err
}

// MergeWithLog 合并分支并显示提交日志
// Merges a branch with commit log
func (g *GitOps) MergeWithLog(branch, message string, logLines int) error {
	args := []string{"merge", branch, "--no-edit", "-m", message}
	if logLines > 0 {
		args = append(args, fmt.Sprintf("--log=%d", logLines))
	}
	_, err := g.execGitCommand(args...)
	return err
}

// MergeAbort 中止合并
// Aborts merge
func (g *GitOps) MergeAbort() error {
	_, err := g.execGitCommand("merge", "--abort")
	return err
}

// CreateBranch 创建分支
// Creates a branch
func (g *GitOps) CreateBranch(branchName string) error {
	_, err := g.execGitCommand("branch", branchName)
	return err
}

// DeleteBranch 删除分支
// Deletes a branch
func (g *GitOps) DeleteBranch(branchName string) error {
	_, err := g.execGitCommand("branch", "-D", branchName)
	return err
}

// Reset 重置到指定提交
// Resets to a specific commit
func (g *GitOps) Reset(ref string, hard bool) error {
	args := []string{"reset"}
	if hard {
		args = append(args, "--hard")
	}
	args = append(args, ref)
	_, err := g.execGitCommand(args...)
	return err
}

// GetConflictedFiles 获取冲突文件列表
// Gets list of conflicted files
func (g *GitOps) GetConflictedFiles() ([]string, error) {
	output, err := g.execGitCommand("diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil, err
	}
	
	if output == "" {
		return []string{}, nil
	}
	
	return strings.Split(output, "\n"), nil
}

// CheckoutTheirs 使用远程版本解决冲突
// Resolves conflict using remote version
func (g *GitOps) CheckoutTheirs(filePath string) error {
	_, err := g.execGitCommand("checkout", "--theirs", filePath)
	return err
}

// CheckoutOurs 使用本地版本解决冲突
// Resolves conflict using local version
func (g *GitOps) CheckoutOurs(filePath string) error {
	_, err := g.execGitCommand("checkout", "--ours", filePath)
	return err
}

// ListFiles 列出文件
// Lists files
func (g *GitOps) ListFiles(args ...string) ([]string, error) {
	cmdArgs := append([]string{"ls-files"}, args...)
	output, err := g.execGitCommand(cmdArgs...)
	if err != nil {
		return nil, err
	}
	
	if output == "" {
		return []string{}, nil
	}
	
	// 检查是否使用-z参数（null分隔）
	// Check if using -z parameter (null-separated)
	for _, arg := range args {
		if arg == "-z" {
			// 使用null字符分割
			// Split by null character
			return strings.Split(output, "\x00"), nil
		}
	}
	
	// 默认使用换行分割
	// Default to newline split
	return strings.Split(output, "\n"), nil
}

// ListBranches 获取所有分支列表
// Gets list of all branches
func (g *GitOps) ListBranches() ([]string, error) {
	output, err := g.execGitCommand("branch", "--list")
	if err != nil {
		return nil, err
	}
	
	if output == "" {
		return []string{}, nil
	}
	
	// 解析分支列表
	// Parse branch list
	lines := strings.Split(output, "\n")
	branches := []string{}
	for _, line := range lines {
		// 移除前导的 * 和空格
		// Remove leading * and spaces
		branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if branch != "" {
			branches = append(branches, branch)
		}
	}
	
	return branches, nil
}

// GetRepoRoot 获取仓库根目录
// Gets repository root directory
func GetRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get repo root: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}
