// Package config / 配置包
// Module: Configuration Example Generator & Validator / 配置示例生成器和验证器
// Function: Generate example config file and validate configuration
//           生成示例配置文件并验证配置
// Author: git-autosync contributors
// Dependencies: fmt, os, strings

package config

import (
	"fmt"
	"os"
	"strings"
)

// ValidateConfig 验证配置有效性
// Validates configuration
// 返回第一个验证错误，或 nil 如果全部有效
// Returns first validation error, or nil if all valid
func ValidateConfig(cfg *Config) error {
	var errors []string

	// 验证数值范围 / Validate numeric ranges
	if cfg.MaxParallelWorkers < 1 || cfg.MaxParallelWorkers > 100 {
		errors = append(errors, fmt.Sprintf("max_parallel_workers 应在 1-100 之间 / should be 1-100, got %d", cfg.MaxParallelWorkers))
	}

	if cfg.MaxConsecutiveFailures < 1 {
		errors = append(errors, fmt.Sprintf("max_consecutive_failures 应大于 0 / should be > 0, got %d", cfg.MaxConsecutiveFailures))
	}

	if cfg.SafeModeMultiplier < 1 {
		errors = append(errors, fmt.Sprintf("safe_mode_multiplier 应大于 0 / should be > 0, got %d", cfg.SafeModeMultiplier))
	}

	// 验证时间值 / Validate duration values
	if cfg.SleepInterval < 1 {
		errors = append(errors, "sleep_interval 应大于 0 / should be > 0")
	}

	if cfg.LockFileMaxAge < 1 {
		errors = append(errors, "lock_file_max_age 应大于 0 / should be > 0")
	}

	// 验证文件大小阈值 / Validate file size thresholds
	if cfg.SmallFileThreshold <= 0 {
		errors = append(errors, "small_file_threshold 应大于 0 / should be > 0")
	}

	if cfg.MediumFileThreshold <= cfg.SmallFileThreshold {
		errors = append(errors, "medium_file_threshold 应大于 small_file_threshold / should be > small_file_threshold")
	}

	// 验证合并策略 / Validate merge strategy
	if cfg.MergeFailureStrategy != "force-push" && cfg.MergeFailureStrategy != "rollback" {
		errors = append(errors, fmt.Sprintf("merge_failure_strategy 应为 'force-push' 或 'rollback' / should be 'force-push' or 'rollback', got '%s'", cfg.MergeFailureStrategy))
	}

	// 验证日志级别 / Validate log level
	validLevels := map[string]bool{"DEBUG": true, "INFO": true, "WARN": true, "ERROR": true}
	if !validLevels[cfg.LogLevel] {
		errors = append(errors, fmt.Sprintf("log_level 应为 DEBUG/INFO/WARN/ERROR / should be DEBUG/INFO/WARN/ERROR, got '%s'", cfg.LogLevel))
	}

	if len(errors) > 0 {
		return fmt.Errorf("配置验证错误 / config validation errors:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// GenerateExampleConfig 生成示例配置文件
// Generates example configuration file
// 所有配置项默认注释，附带中英双语说明
// All options commented by default with bilingual descriptions
func GenerateExampleConfig(path string) error {
	content := `# =============================================================================
# Git Autosync Configuration File / Git自动同步配置文件
# =============================================================================
# 
# 使用方法 / Usage:
# 1. 复制此文件为 git_sync.conf / Copy this file to git_sync.conf
# 2. 取消注释需要修改的配置项 / Uncomment options you want to change
# 3. 重启程序使配置生效 / Restart program to apply changes
#
# 格式说明 / Format:
# - 以 # 开头的行为注释 / Lines starting with # are comments
# - 格式: key = value / Format: key = value
# - 时间格式: 60s, 2m, 1h30m / Duration format: 60s, 2m, 1h30m
# - 大小格式: 字节数 / Size format: bytes (e.g., 5242880 for 5MB)
#
# =============================================================================

# -----------------------------------------------------------------------------
# Git 配置 / Git Configuration
# -----------------------------------------------------------------------------

# 远程仓库名称 / Remote repository name
# remote_name = origin

# 分支名称 / Branch name
# branch_name = main

# -----------------------------------------------------------------------------
# 同步配置 / Sync Configuration
# -----------------------------------------------------------------------------

# 同步间隔 / Sync interval
# 格式: 数字+单位(s/m/h) / Format: number+unit(s/m/h)
# sleep_interval = 60s

# 提交消息前缀 / Commit message prefix
# commit_msg_prefix = Auto-sync / 自动同步:

# -----------------------------------------------------------------------------
# 重试配置 / Retry Configuration
# -----------------------------------------------------------------------------

# git add 最大重试次数 / Max retry attempts for git add
# max_add_attempts = 3

# git add 重试延迟 / Retry delay for git add
# add_retry_delay = 2s

# -----------------------------------------------------------------------------
# 特殊仓库配置 / Special Repository Configuration
# -----------------------------------------------------------------------------

# 特殊仓库基础目录（逗号分隔）/ Special repo base directories (comma-separated)
# subrepo_base_dirs = debian/data/git,debian/data/.oh-my-zsh

# -----------------------------------------------------------------------------
# LFS 配置 / LFS Configuration
# -----------------------------------------------------------------------------

# LFS 文件大小阈值（字节）/ LFS file size threshold (bytes)
# 默认 255MB / Default 255MB
# lfs_size_threshold_bytes = 267386880

# -----------------------------------------------------------------------------
# 文件忽略配置 / File Ignore Configuration
# -----------------------------------------------------------------------------

# 忽略文件大小阈值（字节）/ Ignore file size threshold (bytes)
# 默认 50GB / Default 50GB
# ignore_size_threshold_bytes = 53687091200

# 忽略文件名 / Ignore file name
# ignore_file_name = .gitignore_nopush

# -----------------------------------------------------------------------------
# 空目录配置 / Empty Directory Configuration
# -----------------------------------------------------------------------------

# 空目录占位文件名 / Empty directory placeholder file name
# empty_dir_placeholder_file = .gitkeep

# -----------------------------------------------------------------------------
# 并发配置 / Concurrency Configuration
# -----------------------------------------------------------------------------

# 最大并行工作线程数 / Max parallel workers
# 范围: 1-100 / Range: 1-100
# max_parallel_workers = 16

# -----------------------------------------------------------------------------
# 日志配置 / Log Configuration
# -----------------------------------------------------------------------------

# 日志目录 / Log directory
# log_dir = /var/log/git-autosync

# 单个日志文件最大大小(MB) / Max size per log file (MB)
# log_max_size_mb = 10

# 最大日志备份数量 / Max number of log backups
# log_max_backups = 10

# 日志级别 / Log level
# 可选: DEBUG, INFO, WARN, ERROR
# log_level = INFO

# -----------------------------------------------------------------------------
# 合并失败策略 / Merge Failure Strategy
# -----------------------------------------------------------------------------

# 合并失败时的处理策略 / Strategy when merge fails
# force-push: 强制推送本地状态到远程（适合CNB临时环境）
# rollback: 仅回滚本地，保留备份分支（适合多人协作）
# merge_failure_strategy = force-push

# =============================================================================
# 新增配置 (v2.0) / New Configuration (v2.0)
# =============================================================================

# -----------------------------------------------------------------------------
# 失败处理配置 / Failure Handling Configuration
# -----------------------------------------------------------------------------

# 最大连续失败次数（超过后进入安全模式）/ Max consecutive failures before safe mode
# max_consecutive_failures = 10

# 安全模式休眠倍数 / Safe mode sleep multiplier
# 安全模式下: 实际休眠 = sleep_interval * safe_mode_multiplier
# safe_mode_multiplier = 10

# -----------------------------------------------------------------------------
# 锁文件处理配置 / Lock File Handling Configuration
# -----------------------------------------------------------------------------

# 锁文件最大存活时间（超过认为是残留）/ Max age for stale lock file
# lock_file_max_age = 60s

# 锁文件等待时间 / Wait time when lock exists
# lock_wait_time = 3s

# -----------------------------------------------------------------------------
# 批量处理配置 / Batch Processing Configuration
# -----------------------------------------------------------------------------

# 小文件阈值（字节）/ Small file threshold (bytes)
# 默认 5MB / Default 5MB
# small_file_threshold = 5242880

# 中文件阈值（字节）/ Medium file threshold (bytes)
# 默认 100MB / Default 100MB
# medium_file_threshold = 104857600

# 批处理大小 / Batch size for file operations
# batch_size = 100

# 小批次大小 / Small batch size
# small_batch_size = 50

# -----------------------------------------------------------------------------
# 索引更新重试配置 / Index Update Retry Configuration
# -----------------------------------------------------------------------------

# 索引更新最大重试次数 / Max retries for index update
# index_update_max_retries = 5

# 索引更新重试延迟 / Retry delay for index update
# index_update_retry_delay = 2s

# -----------------------------------------------------------------------------
# 批量操作重试配置 / Batch Operation Retry Configuration
# -----------------------------------------------------------------------------

# 批量操作最大重试次数 / Max retry attempts for batch operations
# batch_retry_max_attempts = 3

# 批量操作重试基础延迟 / Base delay for batch retry
# batch_retry_base_delay = 1s

# -----------------------------------------------------------------------------
# 合并配置 / Merge Configuration
# -----------------------------------------------------------------------------

# 合并日志显示行数 / Lines to show in merge log
# merge_log_lines = 10

# 最大备份分支数量 / Max backup branches to keep
# max_backup_branches = 5

# =============================================================================
# End of Configuration / 配置结束
# =============================================================================
`

	return os.WriteFile(path, []byte(content), 0644)
}
