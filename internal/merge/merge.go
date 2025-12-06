package merge

import (
	"fmt"
	"strings"
	"time"

	"github.com/find-xposed-magisk/git-sync/internal/config"
	"github.com/find-xposed-magisk/git-sync/internal/git"
	"github.com/find-xposed-magisk/git-sync/internal/logger"
)

// MergeManager 合并管理器
// Merge manager
type MergeManager struct {
	cfg    *config.Config
	gitOps *git.GitOps
	logger *logger.Logger
}

// NewMergeManager 创建合并管理器
// Creates a new merge manager
func NewMergeManager(cfg *config.Config, gitOps *git.GitOps, log *logger.Logger) *MergeManager {
	return &MergeManager{
		cfg:    cfg,
		gitOps: gitOps,
		logger: log,
	}
}

// SmartThreeWayMerge 智能三路合并
// Intelligent three-way merge
func (mm *MergeManager) SmartThreeWayMerge() error {
	mm.logger.Phase("智能三路合并 / Intelligent Three-Way Merge")
	
	// 【与 Shell 保持一致】合并前只处理暂存区变更，不执行 git add -A
	// [Shell-compatible] Only handle staged changes before merge, no git add -A
	// 原因：git update-index --index-info 添加的 gitdir 文件在索引中存在但工作目录中不存在，
	//       如果执行 git add -A 会把这些"不存在"的状态暂存为删除操作，导致反复添加-删除的死循环
	// Reason: gitdir files added via git update-index exist in index but not in working directory,
	//         git add -A would stage their "absence" as deletions, causing add-delete loop
	
	// 只检查暂存区变更（不处理工作目录未暂存变更）
	// Only check staged changes (don't handle unstaged working directory changes)
	if hasStaged, _ := mm.gitOps.HasStagedChanges(); hasStaged {
		mm.logger.Warn("检测到残留的暂存变更，自动提交 / Detected remaining staged changes, auto-committing")
		if err := mm.gitOps.Commit("chore: Auto-commit staged changes before merge"); err != nil {
			mm.logger.Warn("Failed to commit staged changes: %v", err)
		}
	}
	
	mm.logger.Debug("✓ 暂存区状态检查通过 / Staged changes check passed")
	
	// 获取本地、远程和共同祖先的提交哈希
	// Get local, remote, and merge base commit hashes
	local, err := mm.gitOps.GetRevision("@")
	if err != nil {
		mm.logger.Error("[错误] 无法获取本地提交信息 / [ERROR] Failed to get local commit info: %v", err)
		return err
	}
	
	remoteRef := fmt.Sprintf("%s/%s", mm.cfg.RemoteName, mm.cfg.BranchName)
	remote, err := mm.gitOps.GetRevision(remoteRef)
	if err != nil {
		mm.logger.Error("[错误] 无法获取远程提交信息 / [ERROR] Failed to get remote commit info: %v", err)
		return err
	}
	
	base, err := mm.gitOps.GetMergeBase("@", remoteRef)
	if err != nil {
		mm.logger.Error("[错误] 无法获取共同祖先 / [ERROR] Failed to get merge base: %v", err)
		return err
	}
	
	// 情况1：本地和远程相同
	// Case 1: Local and remote are the same
	if local == remote {
		mm.logger.Info("✓ 仓库已是最新 / Repository is up-to-date")
		return nil
	}
	
	// 情况2：本地落后（Fast-forward）
	// Case 2: Local is behind (Fast-forward)
	if local == base {
		mm.logger.Debug("→ 本地分支落后，执行快进合并 / Local branch is behind, performing fast-forward merge")
		if err := mm.gitOps.Pull(); err != nil {
			mm.logger.Error("✗ 快进合并失败 / Fast-forward merge failed: %v", err)
			return err
		}
		mm.logger.Info("✓ 快进合并成功 / Fast-forward merge successful")
		return nil
	}
	
	// 情况3：本地领先
	// Case 3: Local is ahead
	if remote == base {
		mm.logger.Debug("→ 本地分支领先，推送变更 / Local branch is ahead, pushing changes")
		if err := mm.gitOps.Push(); err != nil {
			mm.logger.Error("✗ 推送失败 / Push failed: %v", err)
			return err
		}
		mm.logger.Info("✓ 推送成功 / Push successful")
		return nil
	}
	
	// 情况4：分支分叉，需要三路合并
	// Case 4: Branches have diverged, need three-way merge
	mm.logger.Warn("⚠ 分支已分叉，尝试智能三路合并 / Branches have diverged, attempting intelligent three-way merge")
	
	// 创建合并前的备份点
	// Create backup point before merge
	backupBranch := fmt.Sprintf("backup-before-merge-%s", time.Now().Format("20060102-150405"))
	if err := mm.gitOps.CreateBranch(backupBranch); err != nil {
		mm.logger.Error("Failed to create backup branch: %v", err)
		return err
	}
	mm.logger.Debug("→ 已创建备份分支: %s / Backup branch created: %s", backupBranch, backupBranch)
	
	// 尝试自动合并
	// Attempt automatic merge
	mm.logger.Debug("→ 尝试自动合并 / Attempting automatic merge")
	mergeMsg := fmt.Sprintf("Auto-merge: Intelligent three-way merge at %s", time.Now().Format("2006-01-02 15:04:05"))
	
	// 使用MergeWithLog显示合并的提交日志（使用配置的行数）
	// Use MergeWithLog to show merged commit logs (using configured line count)
	err = mm.gitOps.MergeWithLog(remoteRef, mergeMsg, mm.cfg.MergeLogLines)
	if err == nil {
		// 合并成功
		// Merge successful
		mm.logger.Info("✓ 自动合并成功 / Automatic merge successful")
		
		// 推送合并结果
		// Push merge result
		mm.logger.Debug("→ 推送合并结果 / Pushing merge result")
		if err := mm.gitOps.Push(); err != nil {
			mm.logger.Error("✗ 推送失败，但本地合并已完成 / Push failed, but local merge is complete: %v", err)
			return err
		}
		
		mm.logger.Info("✓ 合并结果已推送 / Merge result pushed successfully")
		
		// 删除备份分支
		// Delete backup branch
		mm.logger.Debug("清理备份分支 / Cleaning up backup branch: %s", backupBranch)
		if err := mm.gitOps.DeleteBranch(backupBranch); err != nil {
			mm.logger.Warn("删除备份分支失败 (已忽略) / Failed to delete backup branch (ignored): %v", err)
		} else {
			mm.logger.Debug("  ✓ 备份分支已删除 / Backup branch deleted")
		}
		
		return nil
	}
	
	// 合并冲突
	// Merge conflicts
	mm.logger.Error("✗ 检测到合并冲突 / Merge conflicts detected")
	
	// 显示冲突文件列表
	// Display conflicted files list
	conflictFiles, err := mm.gitOps.GetConflictedFiles()
	if err != nil {
		mm.logger.Error("Failed to get conflicted files: %v", err)
		return err
	}
	
	mm.logger.Warn("冲突文件列表 / Conflicted files:")
	for _, file := range conflictFiles {
		mm.logger.Error("  - %s", file)
	}
	
	// 尝试智能解决冲突
	// Attempt intelligent conflict resolution
	mm.logger.Debug("→ 尝试智能解决冲突 / Attempting intelligent conflict resolution")
	
	conflictsResolved := 0
	conflictsTotal := len(conflictFiles)
	
	for _, conflictFile := range conflictFiles {
		// 对于自动生成的文件，优先使用远程版本
		// For auto-generated files, prefer remote version
		isLockFile := false
		for _, pattern := range config.LockFilePatterns {
			if strings.Contains(conflictFile, pattern) {
				isLockFile = true
				break
			}
		}
		
		if isLockFile {
			mm.logger.Debug("  → 自动解决锁文件冲突（使用远程版本）/ Auto-resolving lock file conflict (using remote): %s", conflictFile)
			if err := mm.gitOps.CheckoutTheirs(conflictFile); err != nil {
				mm.logger.Warn("Failed to checkout theirs for %s: %v", conflictFile, err)
				continue
			}
			
			if err := mm.gitOps.Add(conflictFile); err != nil {
				mm.logger.Warn("Failed to add resolved file %s: %v", conflictFile, err)
				continue
			}
			
			conflictsResolved++
		}
	}
	
	if conflictsResolved > 0 {
		mm.logger.Info("  → 已自动解决 %d / %d 个冲突 / Auto-resolved %d / %d conflicts", 
			conflictsResolved, conflictsTotal, conflictsResolved, conflictsTotal)
	}
	
	// 检查是否所有冲突都已解决
	// Check if all conflicts are resolved
	remainingConflicts, err := mm.gitOps.GetConflictedFiles()
	if err != nil {
		mm.logger.Error("Failed to check remaining conflicts: %v", err)
		return err
	}
	
	if len(remainingConflicts) == 0 {
		mm.logger.Info("✓ 所有冲突已自动解决 / All conflicts automatically resolved")
		
		// 完成合并
		// Complete merge
		if err := mm.gitOps.Commit(mergeMsg); err != nil {
			mm.logger.Error("Failed to commit merge: %v", err)
			return err
		}
		
		// 推送合并结果
		// Push merge result
		if err := mm.gitOps.Push(); err != nil {
			mm.logger.Error("✗ 推送失败 / Push failed: %v", err)
			return err
		}
		
		mm.logger.Info("✓ 合并完成并已推送 / Merge completed and pushed")
		
		// 删除备份分支
		// Delete backup branch
		mm.logger.Debug("清理备份分支 / Cleaning up backup branch: %s", backupBranch)
		if err := mm.gitOps.DeleteBranch(backupBranch); err != nil {
			mm.logger.Warn("删除备份分支失败 (已忽略) / Failed to delete backup branch (ignored): %v", err)
		} else {
			mm.logger.Debug("  ✓ 备份分支已删除 / Backup branch deleted")
		}
		
		return nil
	}
	
	// 仍有未解决的冲突
	// Unresolved conflicts remain
	mm.logger.Error("✗ 仍有未解决的冲突，需要手动干预 / Unresolved conflicts remain, manual intervention required")
	mm.logger.Warn("→ 中止合并并恢复到合并前状态 / Aborting merge and restoring to pre-merge state")
	
	// 使用增强的安全回滚机制
	// Use enhanced safe rollback mechanism
	if err := mm.SafeRollback(backupBranch); err != nil {
		mm.logger.Error("安全回滚失败 / Safe rollback failed: %v", err)
		return fmt.Errorf("rollback failed: %w", err)
	}
	
	mm.logger.Debug("→ 已恢复到备份分支，请手动解决冲突 / Restored to backup branch. Please resolve conflicts manually")
	mm.logger.Debug("→ 备份分支: %s / Backup branch: %s", backupBranch, backupBranch)
	
	return fmt.Errorf("merge conflicts require manual resolution")
}
