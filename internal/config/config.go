package config

import (
	"time"
)

// Config 全局配置结构
// Global configuration structure
type Config struct {
	// Git配置 / Git configuration
	RemoteName string
	BranchName string

	// 同步配置 / Sync configuration
	SleepInterval   time.Duration
	CommitMsgPrefix string

	// 重试配置 / Retry configuration
	MaxAddAttempts int
	AddRetryDelay  time.Duration

	// 特殊仓库配置 / Special repository configuration
	SubrepoBaseDirs []string

	// LFS配置 / LFS configuration
	LFSSizeThresholdBytes int64
	LFSTrackPatterns      []string

	// 文件忽略配置 / File ignore configuration
	IgnoreSizeThresholdBytes int64
	IgnoreFileName           string

	// 空目录占位文件 / Empty directory placeholder
	EmptyDirPlaceholderFile string

	// 仓库根目录 / Repository root directory
	RepoRoot string

	// 并发配置 / Concurrency configuration
	MaxParallelWorkers int

	// 日志配置 / Log configuration
	LogDir        string // 日志目录 / Log directory
	LogMaxSizeMB  int    // 单个日志文件最大大小(MB) / Max size per log file (MB)
	LogMaxBackups int    // 最大备份数量 / Max number of backups
	LogLevel      string // 日志级别: DEBUG/INFO/WARN/ERROR

	// 合并失败策略 / Merge failure strategy
	// "force-push": 强制推送本地状态到远程（默认，适合CNB环境）
	// "rollback": 仅回滚本地，保留备份分支（适合多人协作）
	MergeFailureStrategy string // "force-push" or "rollback"

	// ============================================================
	// 以下为新增配置字段 (v2.0)
	// New configuration fields below (v2.0)
	// ============================================================

	// 失败处理配置 / Failure handling configuration
	MaxConsecutiveFailures int // 最大连续失败次数 / Max consecutive failures before safe mode
	SafeModeMultiplier     int // 安全模式倍数(SleepInterval * N) / Safe mode multiplier

	// 锁文件处理配置 / Lock file handling configuration
	LockFileMaxAge time.Duration // 锁文件最大存活时间 / Max age for stale lock file
	LockWaitTime   time.Duration // 锁文件等待时间 / Wait time when lock exists

	// 批量处理配置 / Batch processing configuration
	SmallFileThreshold  int64 // 小文件阈值(字节) / Small file threshold (bytes)
	MediumFileThreshold int64 // 中文件阈值(字节) / Medium file threshold (bytes)
	BatchSize           int   // 批处理大小 / Batch size for file operations
	SmallBatchSize      int   // 小批次大小 / Small batch size

	// 索引更新重试配置 / Index update retry configuration
	IndexUpdateMaxRetries int           // 索引更新最大重试次数 / Max retries for index update
	IndexUpdateRetryDelay time.Duration // 索引更新重试延迟 / Retry delay for index update

	// 批量操作重试配置 / Batch operation retry configuration
	BatchRetryMaxAttempts int           // 批量操作最大重试次数 / Max retry attempts for batch ops
	BatchRetryBaseDelay   time.Duration // 批量操作重试基础延迟 / Base delay for batch retry

	// 合并配置 / Merge configuration
	MergeLogLines      int // 合并日志显示行数 / Lines to show in merge log
	MaxBackupBranches  int // 最大备份分支数量 / Max backup branches to keep
}

// DefaultConfig 返回默认配置
// Returns default configuration
func DefaultConfig() *Config {
	return &Config{
		// Git配置 / Git configuration
		RemoteName: "origin",
		BranchName: "main",

		// 同步配置 / Sync configuration
		SleepInterval:   60 * time.Second,
		CommitMsgPrefix: "Auto-sync / 自动同步:",

		// 重试配置 / Retry configuration
		MaxAddAttempts: 3,
		AddRetryDelay:  2 * time.Second,

		// 特殊仓库配置 / Special repository configuration
		SubrepoBaseDirs: []string{"debian/data/git", "debian/data/.oh-my-zsh"},

		// LFS配置 / LFS configuration
		LFSSizeThresholdBytes: 255 * 1024 * 1024, // 255MB
		LFSTrackPatterns:      []string{},

		// 文件忽略配置 / File ignore configuration
		IgnoreSizeThresholdBytes: 50 * 1024 * 1024 * 1024, // 50GB
		IgnoreFileName:           ".gitignore_nopush",

		// 空目录占位文件 / Empty directory placeholder
		EmptyDirPlaceholderFile: ".gitkeep",

		// 并发配置 / Concurrency configuration
		MaxParallelWorkers: 16, // 从4增加到16 / Increased from 4 to 16

		// 日志配置 / Log configuration
		LogDir:        "/var/log/git-autosync",
		LogMaxSizeMB:  10,
		LogMaxBackups: 10,
		LogLevel:      "INFO",

		// 合并失败策略 / Merge failure strategy
		// 默认使用 force-push 策略，适合 CNB 临时环境
		// Default to force-push strategy, suitable for CNB ephemeral environment
		MergeFailureStrategy: "force-push",

		// ============================================================
		// 新增配置默认值 (v2.0)
		// New configuration defaults (v2.0)
		// ============================================================

		// 失败处理配置 / Failure handling configuration
		MaxConsecutiveFailures: 10, // 连续失败10次后进入安全模式
		SafeModeMultiplier:     10, // 安全模式下休眠时间为 SleepInterval * 10

		// 锁文件处理配置 / Lock file handling configuration
		LockFileMaxAge: 60 * time.Second, // 锁文件超过60秒认为是残留
		LockWaitTime:   3 * time.Second,  // 等待锁释放的时间

		// 批量处理配置 / Batch processing configuration
		SmallFileThreshold:  5 * 1024 * 1024,   // 5MB - 小文件阈值
		MediumFileThreshold: 100 * 1024 * 1024, // 100MB - 中文件阈值
		BatchSize:           100,               // 批处理大小
		SmallBatchSize:      50,                // 小批次大小

		// 索引更新重试配置 / Index update retry configuration
		IndexUpdateMaxRetries: 5,                // 最大重试5次
		IndexUpdateRetryDelay: 2 * time.Second,  // 重试间隔2秒

		// 批量操作重试配置 / Batch operation retry configuration
		BatchRetryMaxAttempts: 3,               // 最大重试3次
		BatchRetryBaseDelay:   1 * time.Second, // 重试基础延迟1秒

		// 合并配置 / Merge configuration
		MergeLogLines:     10, // 显示10行合并日志
		MaxBackupBranches: 5,  // 最多保留5个备份分支
	}
}

// VirtualEnvExcludePatterns 虚拟环境排除规则
// Virtual environment exclusion patterns
// 仅在特殊仓库处理时应用，不污染.gitignore
// Only applied during special repository processing, does not pollute .gitignore
var VirtualEnvExcludePatterns = []string{
	"venv",          // Python虚拟环境 / Python virtual environment
	"env",           // Python虚拟环境 / Python virtual environment
	".venv",         // Python虚拟环境 / Python virtual environment
	"__pycache__",   // Python缓存 / Python cache
	"node_modules",  // Node.js模块 / Node.js modules
	"vendor",        // 依赖目录 / Dependency directory
}

// LockFilePatterns 锁文件模式（用于智能冲突解决）
// Lock file patterns (for intelligent conflict resolution)
var LockFilePatterns = []string{
	"package-lock.json",  // npm
	"yarn.lock",          // yarn
	"pnpm-lock.yaml",     // pnpm
	"Pipfile.lock",       // Python pipenv
	"composer.lock",      // PHP composer
	"Gemfile.lock",       // Ruby bundler
	"go.sum",             // Go modules
	"Cargo.lock",         // Rust cargo
}
