// Package config / 配置包
// Module: Configuration File Loader / 配置文件加载器
// Function: Load configuration from file and generate example config
//           从文件加载配置并生成示例配置
// Author: git-autosync contributors
// Dependencies: bufio, fmt, os, strconv, strings, time

package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ConfigFileName 配置文件名
// Configuration file name
const ConfigFileName = "git_sync.conf"

// ExampleConfigFileName 示例配置文件名
// Example configuration file name
const ExampleConfigFileName = "git_sync.conf.example"

// LoadConfigFromFile 从指定路径加载配置文件
// Loads configuration from the specified path
// 如果文件不存在，返回默认配置并生成示例文件
// If file does not exist, returns default config and generates example file
func LoadConfigFromFile(workDir string) (*Config, error) {
	cfg := DefaultConfig()
	configPath := workDir + "/" + ConfigFileName
	examplePath := workDir + "/" + ExampleConfigFileName

	// 检查配置文件是否存在 / Check if config file exists
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，生成示例文件 / Config not found, generate example
			fmt.Printf("[INFO] 配置文件未找到，使用默认配置 / Config file not found, using defaults: %s\n", configPath)
			if genErr := GenerateExampleConfig(examplePath); genErr != nil {
				fmt.Printf("[WARN] 生成示例配置失败 / Failed to generate example config: %v\n", genErr)
			} else {
				fmt.Printf("[INFO] 已生成示例配置 / Generated example config: %s\n", examplePath)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("无法打开配置文件 / failed to open config file: %w", err)
	}
	defer file.Close()

	fmt.Printf("[INFO] 正在加载配置文件 / Loading config file: %s\n", configPath)

	// 逐行解析配置 / Parse config line by line
	scanner := bufio.NewScanner(file)
	lineNum := 0
	loadedCount := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释 / Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析 key=value / Parse key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			fmt.Printf("[WARN] 第%d行格式无效 / Invalid format at line %d: %s\n", lineNum, lineNum, line)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// 移除行内注释 / Remove inline comments
		if idx := strings.Index(value, " #"); idx > 0 {
			value = strings.TrimSpace(value[:idx])
		}

		// 应用配置值 / Apply config value
		if applyConfigValue(cfg, key, value, lineNum) {
			loadedCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取配置文件出错 / error reading config file: %w", err)
	}

	fmt.Printf("[INFO] 已加载 %d 个配置项 / Loaded %d config items\n", loadedCount, loadedCount)

	// 验证配置 / Validate config
	if err := ValidateConfig(cfg); err != nil {
		fmt.Printf("[WARN] 配置验证警告 / Config validation warning: %v\n", err)
	}

	return cfg, nil
}

// applyConfigValue 应用单个配置值到配置结构
// Applies a single config value to the config struct
// 返回 true 如果成功应用 / Returns true if successfully applied
func applyConfigValue(cfg *Config, key, value string, lineNum int) bool {
	switch key {
	// Git配置 / Git configuration
	case "remote_name":
		cfg.RemoteName = value
	case "branch_name":
		cfg.BranchName = value

	// 同步配置 / Sync configuration
	case "sleep_interval":
		if d, err := time.ParseDuration(value); err == nil {
			cfg.SleepInterval = d
		} else {
			logParseError(key, value, lineNum, cfg.SleepInterval)
			return false
		}
	case "commit_msg_prefix":
		cfg.CommitMsgPrefix = value

	// 重试配置 / Retry configuration
	case "max_add_attempts":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.MaxAddAttempts = v
		} else {
			logParseError(key, value, lineNum, cfg.MaxAddAttempts)
			return false
		}
	case "add_retry_delay":
		if d, err := time.ParseDuration(value); err == nil {
			cfg.AddRetryDelay = d
		} else {
			logParseError(key, value, lineNum, cfg.AddRetryDelay)
			return false
		}

	// 特殊仓库配置 / Special repository configuration
	case "subrepo_base_dirs":
		cfg.SubrepoBaseDirs = parseStringSlice(value)

	// LFS配置 / LFS configuration
	case "lfs_size_threshold_bytes":
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			cfg.LFSSizeThresholdBytes = v
		} else {
			logParseError(key, value, lineNum, cfg.LFSSizeThresholdBytes)
			return false
		}

	// 文件忽略配置 / File ignore configuration
	case "ignore_size_threshold_bytes":
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			cfg.IgnoreSizeThresholdBytes = v
		} else {
			logParseError(key, value, lineNum, cfg.IgnoreSizeThresholdBytes)
			return false
		}
	case "ignore_file_name":
		cfg.IgnoreFileName = value

	// 空目录占位文件 / Empty directory placeholder
	case "empty_dir_placeholder_file":
		cfg.EmptyDirPlaceholderFile = value

	// 并发配置 / Concurrency configuration
	case "max_parallel_workers":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.MaxParallelWorkers = v
		} else {
			logParseError(key, value, lineNum, cfg.MaxParallelWorkers)
			return false
		}

	// 日志配置 / Log configuration
	case "log_dir":
		cfg.LogDir = value
	case "log_max_size_mb":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.LogMaxSizeMB = v
		} else {
			logParseError(key, value, lineNum, cfg.LogMaxSizeMB)
			return false
		}
	case "log_max_backups":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.LogMaxBackups = v
		} else {
			logParseError(key, value, lineNum, cfg.LogMaxBackups)
			return false
		}
	case "log_level":
		cfg.LogLevel = strings.ToUpper(value)

	// 合并失败策略 / Merge failure strategy
	case "merge_failure_strategy":
		cfg.MergeFailureStrategy = value

	// ============================================================
	// 新增配置字段 (v2.0)
	// New configuration fields (v2.0)
	// ============================================================

	// 失败处理配置 / Failure handling configuration
	case "max_consecutive_failures":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.MaxConsecutiveFailures = v
		} else {
			logParseError(key, value, lineNum, cfg.MaxConsecutiveFailures)
			return false
		}
	case "safe_mode_multiplier":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.SafeModeMultiplier = v
		} else {
			logParseError(key, value, lineNum, cfg.SafeModeMultiplier)
			return false
		}

	// 锁文件处理配置 / Lock file handling configuration
	case "lock_file_max_age":
		if d, err := time.ParseDuration(value); err == nil {
			cfg.LockFileMaxAge = d
		} else {
			logParseError(key, value, lineNum, cfg.LockFileMaxAge)
			return false
		}
	case "lock_wait_time":
		if d, err := time.ParseDuration(value); err == nil {
			cfg.LockWaitTime = d
		} else {
			logParseError(key, value, lineNum, cfg.LockWaitTime)
			return false
		}

	// 批量处理配置 / Batch processing configuration
	case "small_file_threshold":
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			cfg.SmallFileThreshold = v
		} else {
			logParseError(key, value, lineNum, cfg.SmallFileThreshold)
			return false
		}
	case "medium_file_threshold":
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			cfg.MediumFileThreshold = v
		} else {
			logParseError(key, value, lineNum, cfg.MediumFileThreshold)
			return false
		}
	case "batch_size":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.BatchSize = v
		} else {
			logParseError(key, value, lineNum, cfg.BatchSize)
			return false
		}
	case "small_batch_size":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.SmallBatchSize = v
		} else {
			logParseError(key, value, lineNum, cfg.SmallBatchSize)
			return false
		}

	// 索引更新重试配置 / Index update retry configuration
	case "index_update_max_retries":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.IndexUpdateMaxRetries = v
		} else {
			logParseError(key, value, lineNum, cfg.IndexUpdateMaxRetries)
			return false
		}
	case "index_update_retry_delay":
		if d, err := time.ParseDuration(value); err == nil {
			cfg.IndexUpdateRetryDelay = d
		} else {
			logParseError(key, value, lineNum, cfg.IndexUpdateRetryDelay)
			return false
		}

	// 批量操作重试配置 / Batch operation retry configuration
	case "batch_retry_max_attempts":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.BatchRetryMaxAttempts = v
		} else {
			logParseError(key, value, lineNum, cfg.BatchRetryMaxAttempts)
			return false
		}
	case "batch_retry_base_delay":
		if d, err := time.ParseDuration(value); err == nil {
			cfg.BatchRetryBaseDelay = d
		} else {
			logParseError(key, value, lineNum, cfg.BatchRetryBaseDelay)
			return false
		}

	// 合并配置 / Merge configuration
	case "merge_log_lines":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.MergeLogLines = v
		} else {
			logParseError(key, value, lineNum, cfg.MergeLogLines)
			return false
		}
	case "max_backup_branches":
		if v, err := strconv.Atoi(value); err == nil {
			cfg.MaxBackupBranches = v
		} else {
			logParseError(key, value, lineNum, cfg.MaxBackupBranches)
			return false
		}

	// 远程引用修复配置 / Remote reference repair configuration
	case "auto_fix_corrupt_refs":
		if v, err := strconv.ParseBool(value); err == nil {
			cfg.AutoFixCorruptRefs = v
		} else {
			logParseError(key, value, lineNum, cfg.AutoFixCorruptRefs)
			return false
		}

	default:
		fmt.Printf("[WARN] 未知配置项 / Unknown config key at line %d: %s\n", lineNum, key)
		return false
	}

	return true
}

// logParseError 记录解析错误并使用默认值
// Logs parse error and uses default value
func logParseError(key, value string, lineNum int, defaultVal interface{}) {
	fmt.Printf("[WARN] 第%d行 '%s' 值无效: '%s', 使用默认值: %v / "+
		"Invalid value for '%s' at line %d: '%s', using default: %v\n",
		lineNum, key, value, defaultVal, key, lineNum, value, defaultVal)
}

// parseStringSlice 解析逗号分隔的字符串列表
// Parses comma-separated string list
func parseStringSlice(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
