// loader_test.go - Configuration loader unit tests / 配置加载器单元测试
//
// Module: config
// Description: Tests for config file loading, parsing, and validation
// Author: git-autosync contributors
// Dependencies: testing, os, path/filepath, time

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoadConfigFromFile_FileNotFound tests loading when config file doesn't exist
// 测试配置文件不存在时的加载行为
func TestLoadConfigFromFile_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	examplePath := filepath.Join(tmpDir, ExampleConfigFileName)

	cfg, err := LoadConfigFromFile(tmpDir)

	if err != nil {
		t.Errorf("Expected no error when file not found, got: %v", err)
	}

	// Verify example file was generated / 验证示例文件已生成
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		t.Error("Expected example file to be generated")
	}

	// Verify config has default values / 验证配置使用默认值
	if cfg.SleepInterval != 60*time.Second {
		t.Errorf("Expected default SleepInterval 60s, got: %v", cfg.SleepInterval)
	}
}

// TestLoadConfigFromFile_BasicParsing tests basic config file parsing
// 测试基本配置文件解析
func TestLoadConfigFromFile_BasicParsing(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ConfigFileName)

	// Create test config file / 创建测试配置文件
	configContent := `# Test config / 测试配置
remote_name = test-origin
branch_name = develop
sleep_interval = 30s
max_consecutive_failures = 5
log_level = DEBUG
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFromFile(tmpDir)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify parsed values / 验证解析的值
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"RemoteName", cfg.RemoteName, "test-origin"},
		{"BranchName", cfg.BranchName, "develop"},
		{"SleepInterval", cfg.SleepInterval, 30 * time.Second},
		{"MaxConsecutiveFailures", cfg.MaxConsecutiveFailures, 5},
		{"LogLevel", cfg.LogLevel, "DEBUG"},
	}

	for _, tt := range tests {
		if tt.got != tt.expected {
			t.Errorf("%s: expected %v, got %v", tt.name, tt.expected, tt.got)
		}
	}
}

// TestLoadConfigFromFile_DurationParsing tests duration format parsing
// 测试时间格式解析
func TestLoadConfigFromFile_DurationParsing(t *testing.T) {
	testCases := []struct {
		configLine string
		expected   time.Duration
	}{
		{"sleep_interval = 30s", 30 * time.Second},
		{"sleep_interval = 2m", 2 * time.Minute},
		{"sleep_interval = 1h30m", 90 * time.Minute},
		{"sleep_interval = 1h", 1 * time.Hour},
	}

	for _, tc := range testCases {
		t.Run(tc.configLine, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, ConfigFileName)
			if err := os.WriteFile(configPath, []byte(tc.configLine), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, _ := LoadConfigFromFile(tmpDir)

			if cfg.SleepInterval != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, cfg.SleepInterval)
			}
		})
	}
}

// TestLoadConfigFromFile_CommentHandling tests comment line handling
// 测试注释行处理
func TestLoadConfigFromFile_CommentHandling(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ConfigFileName)

	configContent := `# This is a comment / 这是注释
remote_name = origin
# Another comment / 另一个注释
  # Indented comment / 缩进注释
branch_name = main
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _ := LoadConfigFromFile(tmpDir)

	if cfg.RemoteName != "origin" {
		t.Errorf("Expected RemoteName 'origin', got: %s", cfg.RemoteName)
	}
	if cfg.BranchName != "main" {
		t.Errorf("Expected BranchName 'main', got: %s", cfg.BranchName)
	}
}

// TestLoadConfigFromFile_WhitespaceHandling tests whitespace handling
// 测试空白字符处理
func TestLoadConfigFromFile_WhitespaceHandling(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ConfigFileName)

	configContent := `  remote_name  =  origin  
	branch_name	=	main	
sleep_interval= 30s
log_level =INFO
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _ := LoadConfigFromFile(tmpDir)

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"RemoteName", cfg.RemoteName, "origin"},
		{"BranchName", cfg.BranchName, "main"},
		{"LogLevel", cfg.LogLevel, "INFO"},
	}

	for _, tt := range tests {
		if tt.got != tt.expected {
			t.Errorf("%s: expected '%s', got '%s'", tt.name, tt.expected, tt.got)
		}
	}
}

// TestLoadConfigFromFile_InvalidLine tests handling of invalid config lines
// 测试无效配置行处理
func TestLoadConfigFromFile_InvalidLine(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ConfigFileName)

	// Invalid line without '=' should be skipped / 没有 '=' 的无效行应被跳过
	configContent := `remote_name = origin
invalid_line_without_equals
branch_name = main
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFromFile(tmpDir)

	// Should not return error, just warn / 不应返回错误，只是警告
	if err != nil {
		t.Errorf("Expected no error for invalid line, got: %v", err)
	}

	// Valid lines should still be parsed / 有效行仍应被解析
	if cfg.RemoteName != "origin" {
		t.Errorf("Expected RemoteName 'origin', got: %s", cfg.RemoteName)
	}
}

// TestLoadConfigFromFile_IntParsing tests integer value parsing
// 测试整数值解析
func TestLoadConfigFromFile_IntParsing(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ConfigFileName)

	configContent := `max_parallel_workers = 32
max_consecutive_failures = 15
batch_size = 200
small_file_threshold = 10485760
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _ := LoadConfigFromFile(tmpDir)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"MaxParallelWorkers", cfg.MaxParallelWorkers, 32},
		{"MaxConsecutiveFailures", cfg.MaxConsecutiveFailures, 15},
		{"BatchSize", cfg.BatchSize, 200},
		{"SmallFileThreshold", cfg.SmallFileThreshold, int64(10485760)},
	}

	for _, tt := range tests {
		if tt.got != tt.expected {
			t.Errorf("%s: expected %v, got %v", tt.name, tt.expected, tt.got)
		}
	}
}

// TestValidateConfig tests configuration validation
// 测试配置验证
func TestValidateConfig(t *testing.T) {
	t.Run("Valid config", func(t *testing.T) {
		cfg := DefaultConfig()
		err := ValidateConfig(cfg)
		// Should not return error for valid config
		if err != nil {
			t.Errorf("Expected no error for valid config, got: %v", err)
		}
	})

	t.Run("SleepInterval zero", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.SleepInterval = 0 // Zero value, invalid
		err := ValidateConfig(cfg)
		// Should return validation error / 应返回验证错误
		if err == nil {
			t.Error("Expected validation error for SleepInterval = 0")
		}
	})

	t.Run("MaxParallelWorkers zero", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MaxParallelWorkers = 0 // Invalid
		err := ValidateConfig(cfg)
		// Should return validation error / 应返回验证错误
		if err == nil {
			t.Error("Expected validation error for MaxParallelWorkers = 0")
		}
	})

	t.Run("MaxParallelWorkers too high", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MaxParallelWorkers = 200 // Invalid (> 100)
		err := ValidateConfig(cfg)
		// Should return validation error / 应返回验证错误
		if err == nil {
			t.Error("Expected validation error for MaxParallelWorkers > 100")
		}
	})

	t.Run("Invalid merge strategy", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MergeFailureStrategy = "invalid"
		err := ValidateConfig(cfg)
		// Should return validation error / 应返回验证错误
		if err == nil {
			t.Error("Expected validation error for invalid merge strategy")
		}
	})

	t.Run("Invalid log level", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.LogLevel = "INVALID"
		err := ValidateConfig(cfg)
		// Should return validation error / 应返回验证错误
		if err == nil {
			t.Error("Expected validation error for invalid log level")
		}
	})
}

// TestGenerateExampleConfig tests example config file generation
// 测试示例配置文件生成
func TestGenerateExampleConfig(t *testing.T) {
	tmpDir := t.TempDir()
	examplePath := filepath.Join(tmpDir, "git_sync.conf.example")

	err := GenerateExampleConfig(examplePath)
	if err != nil {
		t.Errorf("Failed to generate example config: %v", err)
	}

	// Verify file exists / 验证文件存在
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		t.Error("Example file was not created")
	}

	// Verify file has content / 验证文件有内容
	content, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatal(err)
	}

	if len(content) < 1000 {
		t.Error("Example file content seems too short")
	}

	// Verify it contains expected sections / 验证包含预期的段落
	contentStr := string(content)
	expectedStrings := []string{
		"remote_name",
		"branch_name",
		"sleep_interval",
		"log_level",
		"max_parallel_workers",
	}

	for _, s := range expectedStrings {
		if !contains(contentStr, s) {
			t.Errorf("Expected example config to contain '%s'", s)
		}
	}
}

// TestAllConfigKeys tests that all config keys are recognized
// 测试所有配置键都被识别
func TestAllConfigKeys(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ConfigFileName)

	// Test all supported config keys / 测试所有支持的配置键
	configContent := `remote_name = origin
branch_name = main
sleep_interval = 60s
commit_msg_prefix = Test:
max_add_attempts = 5
add_retry_delay = 3s
lfs_size_threshold_bytes = 100000000
ignore_size_threshold_bytes = 1000000000
ignore_file_name = .customignore
empty_dir_placeholder_file = .placeholder
max_parallel_workers = 8
log_dir = /tmp/logs
log_max_size_mb = 20
log_max_backups = 5
log_level = WARN
merge_failure_strategy = rollback
max_consecutive_failures = 20
safe_mode_multiplier = 5
lock_file_max_age = 120s
lock_wait_time = 5s
small_file_threshold = 1000000
medium_file_threshold = 50000000
batch_size = 50
small_batch_size = 25
index_update_max_retries = 10
index_update_retry_delay = 3s
batch_retry_max_attempts = 5
batch_retry_base_delay = 2s
merge_log_lines = 20
max_backup_branches = 10
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFromFile(tmpDir)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify a sample of values / 验证部分值
	if cfg.RemoteName != "origin" {
		t.Errorf("RemoteName: expected 'origin', got '%s'", cfg.RemoteName)
	}
	if cfg.MaxParallelWorkers != 8 {
		t.Errorf("MaxParallelWorkers: expected 8, got %d", cfg.MaxParallelWorkers)
	}
	if cfg.MergeFailureStrategy != "rollback" {
		t.Errorf("MergeFailureStrategy: expected 'rollback', got '%s'", cfg.MergeFailureStrategy)
	}
	if cfg.MaxConsecutiveFailures != 20 {
		t.Errorf("MaxConsecutiveFailures: expected 20, got %d", cfg.MaxConsecutiveFailures)
	}
	if cfg.LockFileMaxAge != 120*time.Second {
		t.Errorf("LockFileMaxAge: expected 120s, got %v", cfg.LockFileMaxAge)
	}
	if cfg.BatchSize != 50 {
		t.Errorf("BatchSize: expected 50, got %d", cfg.BatchSize)
	}
	if cfg.MergeLogLines != 20 {
		t.Errorf("MergeLogLines: expected 20, got %d", cfg.MergeLogLines)
	}
}

// Helper function / 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
