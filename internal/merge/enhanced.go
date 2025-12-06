package merge

import (
	"fmt"
	"sort"
	"strings"
)

// CleanupOldBackups 清理旧的备份分支
// Cleans up old backup branches
func (mm *MergeManager) CleanupOldBackups(keepLast int) error {
	mm.logger.Debug("清理旧备份分支，保留最近 %d 个 / Cleaning old backup branches, keeping last %d", keepLast, keepLast)
	
	// 获取所有分支列表
	// Get all branches
	branches, err := mm.gitOps.ListBranches()
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}
	
	// 过滤出备份分支
	// Filter backup branches
	backupBranches := []string{}
	for _, branch := range branches {
		if strings.HasPrefix(branch, "backup-before-merge-") {
			backupBranches = append(backupBranches, branch)
		}
	}
	
	if len(backupBranches) <= keepLast {
		mm.logger.Debug("备份分支数量 %d <= %d，无需清理 / Backup branches count %d <= %d, no cleanup needed", 
			len(backupBranches), keepLast, len(backupBranches), keepLast)
		return nil
	}
	
	// 按时间排序（分支名包含时间戳）
	// Sort by time (branch name contains timestamp)
	sort.Strings(backupBranches)
	
	// 删除旧备份
	// Delete old backups
	toDelete := backupBranches[:len(backupBranches)-keepLast]
	mm.logger.Info("清理 %d 个旧备份分支 / Cleaning %d old backup branches", len(toDelete), len(toDelete))
	
	for _, old := range toDelete {
		mm.logger.Debug("  删除备份分支 / Deleting backup branch: %s", old)
		if err := mm.gitOps.DeleteBranch(old); err != nil {
			mm.logger.Warn("删除备份分支失败 / Failed to delete backup branch %s: %v", old, err)
			// 继续删除其他分支
			// Continue deleting other branches
		}
	}
	
	return nil
}

// SafeRollback 安全回滚到备份分支
// Safely rollback to backup branch
func (mm *MergeManager) SafeRollback(backupBranch string) error {
	mm.logger.Warn("执行安全回滚 / Performing safe rollback")
	
	// 尝试1: 标准回滚
	// Attempt 1: Standard rollback
	if err := mm.gitOps.MergeAbort(); err != nil {
		mm.logger.Error("MergeAbort失败 / MergeAbort failed: %v", err)
		
		// 尝试2: 强制清理合并状态
		// Attempt 2: Force clean merge state
		mm.logger.Info("尝试强制清理合并状态 / Attempting to force clean merge state")
		if err := mm.gitOps.Reset("HEAD", false); err != nil {
			mm.logger.Error("Reset失败 / Reset failed: %v", err)
		}
	}
	
	// 尝试3: 恢复到备份分支
	// Attempt 3: Restore to backup branch
	mm.logger.Info("恢复到备份分支 / Restoring to backup branch: %s", backupBranch)
	if err := mm.gitOps.Reset(backupBranch, true); err != nil {
		mm.logger.Error("Reset到备份分支失败 / Reset to backup branch failed: %v", err)
		
		// 尝试4: 最后的救命稻草
		// Attempt 4: Last resort
		mm.logger.Warn("尝试最后的恢复方案 / Attempting last resort recovery")
		if err := mm.gitOps.Reset("HEAD", true); err != nil {
			return fmt.Errorf("rollback failed, repository may be in inconsistent state: %w", err)
		}
	}
	
	mm.logger.Info("回滚完成 / Rollback completed")
	
	// 根据配置决定是否强制推送
	// Decide whether to force push based on configuration
	if mm.cfg.MergeFailureStrategy == "force-push" {
		mm.logger.Warn("⚠️ 合并失败策略: force-push / Merge failure strategy: force-push")
		mm.logger.Warn("强制同步本地状态到远程 / Force syncing local state to remote")
		mm.logger.Warn("⚠️ 远程的新提交将被覆盖 / Remote commits will be overwritten")
		
		if err := mm.gitOps.ForcePush(); err != nil {
			mm.logger.Error("强制推送失败 / Force push failed: %v", err)
			return fmt.Errorf("force push failed: %w", err)
		}
		
		mm.logger.Info("✓ 已强制同步远程仓库 / Remote repository force synced")
	} else {
		mm.logger.Info("合并失败策略: rollback / Merge failure strategy: rollback")
		mm.logger.Info("保留备份分支供手动处理 / Keeping backup branch for manual intervention")
		mm.logger.Info("备份分支: %s / Backup branch: %s", backupBranch, backupBranch)
	}
	
	return nil
}
