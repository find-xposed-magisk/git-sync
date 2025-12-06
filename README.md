# Git Auto-Sync (GO版本 / GO Version)

[![Language](https://img.shields.io/badge/Language-Go-00ADD8?logo=go)](https://golang.org/)
[![Version](https://img.shields.io/badge/Version-v2.0.0-blue)](https://github.com/find-xposed-magisk/git-sync/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-green)](LICENSE)
[![Release](https://img.shields.io/github/v/release/find-xposed-magisk/git-sync)](https://github.com/find-xposed-magisk/git-sync/releases/latest)

## 📋 项目简介 / Project Overview

**Git Auto-Sync** 是一个高性能的Git自动同步工具，使用GO语言完整复刻Shell脚本版本，提供10-20倍的性能提升。

**Git Auto-Sync** is a high-performance Git automatic synchronization tool, fully rewritten in GO from the Shell script version, providing 10-20x performance improvement.

### ✨ 核心特性 / Core Features

- 🚀 **高性能并发处理** / High-performance concurrent processing
  - 原生goroutine并发，轻松处理14万+文件
  - Native goroutine concurrency, easily handles 140k+ files
  
- 🧠 **智能三路合并** / Intelligent three-way merge
  - 自动检测分支状态（最新/落后/领先/分叉）
  - Automatically detects branch status (up-to-date/behind/ahead/diverged)
  - 智能解决锁文件冲突
  - Intelligently resolves lock file conflicts
  
- 🔒 **特殊仓库支持** / Special repository support
  - 处理包含.git目录的子仓库
  - Handles sub-repositories containing .git directories
  - .git → gitdir 自动转换
  - Automatic .git → gitdir conversion
  
- 📦 **大文件管理** / Large file management
  - 255MB → Git LFS自动追踪
  - 255MB → Automatic Git LFS tracking
  - 50GB → 完全忽略
  - 50GB → Complete ignore
  
- 🌳 **虚拟环境过滤** / Virtual environment filtering
  - 内存中排除规则，不污染.gitignore
  - In-memory exclusion rules, doesn't pollute .gitignore
  - 自动排除venv/node_modules等
  - Automatically excludes venv/node_modules etc.
  
- ⚙️ **可配置合并策略** / Configurable merge strategy
  - 默认force-push，适合CNB临时环境
  - Default force-push, suitable for CNB ephemeral environment
  - 可切换为rollback，适合多人协作
  - Switchable to rollback for team collaboration

---

## 🏗️ 项目结构 / Project Structure

```
git-autosync/
├── cmd/
│   └── git-autosync/
│       └── main.go              # 主程序入口 / Main entry point
├── internal/
│   ├── config/
│   │   ├── config.go            # 配置结构定义 / Configuration structure
│   │   ├── loader.go            # 配置文件加载器 / Config file loader
│   │   └── example.go           # 示例配置生成 / Example config generator
│   ├── git/
│   │   └── git.go               # Git操作封装 / Git operations wrapper
│   ├── file/
│   │   └── file.go              # 文件处理 / File processing
│   ├── subrepo/
│   │   └── subrepo.go           # 特殊仓库处理 / Special repo processing
│   ├── merge/
│   │   └── merge.go             # 智能合并 / Intelligent merge
│   └── logger/
│       └── logger.go            # 日志记录 / Logging
├── go.mod                       # Go模块定义 / Go module definition
├── build.sh                     # 编译脚本 / Build script
└── README.md                    # 项目文档 / Project documentation
```

---

## 🚀 快速开始 / Quick Start

### 1. 前置要求 / Prerequisites

- Go 1.19+ 
- Git 2.x
- Git LFS

### 2. 下载预编译二进制 / Download Pre-built Binary

```bash
# 从 GitHub Releases 下载 / Download from GitHub Releases
# https://github.com/find-xposed-magisk/git-sync/releases

# Linux amd64
curl -LO https://github.com/find-xposed-magisk/git-sync/releases/latest/download/git-sync_linux_amd64.tar.gz
tar -xzf git-sync_linux_amd64.tar.gz
chmod +x git-sync
sudo mv git-sync /usr/local/bin/

# 验证安装 / Verify installation
git-sync -version
```

### 3. 从源码编译 / Build from Source

```bash
# 克隆仓库 / Clone repository
git clone https://github.com/find-xposed-magisk/git-sync.git
cd git-sync

# 编译 / Build
./build.sh

# 或手动编译 / Or manual build
go build -o git-sync ./cmd/git-autosync
```

### 4. 运行 / Run

```bash
# 在Git仓库根目录运行 / Run in git repository root
cd /path/to/your/git/repo
git-sync
```

### 5. 后台运行 / Run in background

```bash
# 使用nohup后台运行 / Run in background with nohup
nohup /workspace/tmpdev/git/git-autosync > /tmp/git-autosync.log 2>&1 &

# 查看日志 / View logs
tail -f /tmp/git-autosync.log
```

---

## ⚙️ 配置说明 / Configuration

### 配置文件 / Config File (v2.0 新增)

程序启动时会自动从工作目录加载 `git_sync.conf` 配置文件：

The program automatically loads `git_sync.conf` from the working directory on startup:

```bash
# 如果配置文件不存在，会自动生成示例文件
# If config file doesn't exist, an example file will be generated
ls git_sync.conf.example

# 复制示例文件并修改
# Copy example file and modify
cp git_sync.conf.example git_sync.conf
vim git_sync.conf
```

### 配置文件格式 / Config File Format

```ini
# Git配置 / Git configuration
remote_name = origin
branch_name = main

# 同步间隔 / Sync interval (支持 s/m/h 格式)
sleep_interval = 60s

# 日志配置 / Log configuration
log_dir = /var/log/git-autosync
log_level = INFO

# 并发配置 / Concurrency
max_parallel_workers = 16

# 失败处理 / Failure handling
max_consecutive_failures = 10
lock_file_max_age = 60s

# 批量处理 / Batch processing
small_file_threshold = 5242880    # 5MB
medium_file_threshold = 104857600 # 100MB
batch_size = 100
```

### 所有配置项 / All Configuration Options

完整配置项列表请参考自动生成的 `git_sync.conf.example` 文件。

For a complete list of options, refer to the auto-generated `git_sync.conf.example` file.

---

## 📊 性能对比 / Performance Comparison

| 指标 / Metric | Shell版本 / Shell | GO版本 / GO | 提升 / Improvement |
|--------------|------------------|-------------|-------------------|
| 处理14万文件 / Process 140k files | 5-10分钟 / 5-10 min | 30秒-1分钟 / 30s-1min | **10-20倍 / 10-20x** |
| 内存占用 / Memory usage | ~500MB | ~100MB | **5倍 / 5x** |
| CPU利用率 / CPU utilization | 单核 / Single core | 多核 / Multi-core | **4倍 / 4x** |
| 启动时间 / Startup time | 即时 / Instant | <100ms | 相当 / Similar |

---

## 🔄 工作流程 / Workflow

```
启动 / Start
  ↓
检查依赖 / Check dependencies
  ↓
主循环(60秒) / Main loop (60s)
  ├─ 阶段1: 特殊仓库处理 / Phase 1: Special repo processing
  │   ├─ 并行处理文件 / Parallel file processing
  │   ├─ 排除虚拟环境 / Exclude virtual environments
  │   └─ .git → gitdir 转换 / .git → gitdir conversion
  ├─ 阶段1.5: 清理孤儿gitdir / Phase 1.5: Clean orphaned gitdir
  ├─ 阶段2: .gitignore清理 / Phase 2: .gitignore cleanup
  ├─ 阶段3: 常规文件处理 / Phase 3: Regular file processing
  │   ├─ 大小检测(LFS/忽略) / Size detection (LFS/ignore)
  │   └─ 空目录处理 / Empty directory handling
  ├─ 提交 / Commit
  └─ 阶段4: 智能三路合并 / Phase 4: Intelligent three-way merge
      ├─ 检测分支状态 / Detect branch status
      ├─ 自动合并 / Auto merge
      ├─ 智能冲突解决 / Intelligent conflict resolution
      └─ 推送 / Push
  ↓
等待60秒 / Wait 60s
  ↓
循环 / Loop
```

---

## 🛠️ 开发说明 / Development

### 代码规范 / Code Standards

- **注释**: 所有关键代码必须有中英双语注释
- **Comments**: All key code must have bilingual comments (Chinese/English)

- **模块化**: 严格遵循单一职责原则
- **Modularity**: Strictly follow Single Responsibility Principle

- **错误处理**: 显式错误处理，不忽略任何错误
- **Error handling**: Explicit error handling, don't ignore any errors

### 测试 / Testing

```bash
# 运行测试 / Run tests
go test ./...

# 运行特定模块测试 / Run specific module tests
go test ./internal/git/...
```

### 调试 / Debugging

```bash
# 启用详细日志 / Enable verbose logging
# 修改logger.NewLogger(true) 中的参数
# Modify the parameter in logger.NewLogger(true)
```

---

## 📝 与Shell版本的差异 / Differences from Shell Version

### 优势 / Advantages

1. **性能**: 10-20倍提升
   - **Performance**: 10-20x improvement

2. **并发**: 原生goroutine，无需手动管理进程池
   - **Concurrency**: Native goroutine, no manual process pool management

3. **内存**: 更低的内存占用
   - **Memory**: Lower memory footprint

4. **错误处理**: 更清晰的错误处理机制
   - **Error handling**: Clearer error handling mechanism

5. **维护性**: 模块化设计，易于维护和扩展
   - **Maintainability**: Modular design, easy to maintain and extend

### 功能一致性 / Feature Parity

- ✅ 所有Shell版本功能已完整实现
- ✅ All Shell version features fully implemented

- ✅ 配置参数保持一致
- ✅ Configuration parameters remain consistent

- ✅ 行为逻辑完全相同
- ✅ Behavior logic exactly the same

---

## 🐛 故障排查 / Troubleshooting

### 问题1: 编译失败 / Issue 1: Build fails

```bash
# 检查GO版本 / Check GO version
go version  # 需要 >= 1.19 / Requires >= 1.19

# 更新依赖 / Update dependencies
go mod tidy
```

### 问题2: Git LFS错误 / Issue 2: Git LFS errors

```bash
# 安装Git LFS / Install Git LFS
apt-get install git-lfs

# 初始化 / Initialize
git lfs install
```

### 问题3: 权限错误 / Issue 3: Permission errors

```bash
# 确保可执行权限 / Ensure executable permission
chmod +x git-autosync
```

---

## 📄 许可证 / License

Apache License 2.0

---

## 👥 贡献 / Contributing

欢迎提交Issue和Pull Request！

Welcome to submit Issues and Pull Requests!

---

## 📧 联系方式 / Contact

- GitHub: [find-xposed-magisk/git-sync](https://github.com/find-xposed-magisk/git-sync)
- Issues: [Report Bug](https://github.com/find-xposed-magisk/git-sync/issues)

---

## 🙏 致谢 / Acknowledgments

- 原始Shell脚本版本 / Original Shell script version
- GO语言社区 / GO language community
- Git和Git LFS项目 / Git and Git LFS projects

---

**注意 / Note**: 此项目适用于CNB云原生临时环境，每次重启会清空工作区。
**Note**: This project is suitable for CNB cloud-native ephemeral environments where the workspace is cleared on every restart.

---

## 📋 更新日志 / Changelog

### v2.0 (2025-12-07)

**新增功能 / New Features**:
- ✅ **外部配置文件支持** / External configuration file support
  - 支持从 `git_sync.conf` 加载配置
  - Support loading config from `git_sync.conf`
  - 配置文件不存在时自动生成带注释的示例文件
  - Auto-generate commented example file when config not found
  - 所有硬编码值均可通过配置文件灵活调整
  - All hardcoded values configurable via config file

- ✅ **配置验证** / Configuration validation
  - 自动验证配置值范围和格式
  - Auto-validate config value ranges and formats
  - 无效配置时优雅降级到默认值
  - Graceful fallback to defaults on invalid config

- ✅ **新增可配置项** / New configurable options
  - 失败处理: `max_consecutive_failures`, `safe_mode_multiplier`
  - 锁文件: `lock_file_max_age`, `lock_wait_time`
  - 批量处理: `small_file_threshold`, `medium_file_threshold`, `batch_size`
  - 重试配置: `index_update_max_retries`, `batch_retry_max_attempts`
  - 合并配置: `merge_log_lines`, `max_backup_branches`

### v12.3 (2025-12-02)

**BUG修复 / Bug Fixes**:
- 🐛 **修复Git路径引号解析问题** / Fixed Git path quote parsing issue
  - 问题: 包含中文等特殊字符的文件路径无法正确删除同步
  - Issue: File paths containing special characters (like Chinese) couldn't be deleted/synced correctly
  - 根因: `git ls-files -s` 输出带引号和八进制转义，未正确解析
  - Root cause: `git ls-files -s` outputs quoted paths with octal escapes, not parsed correctly
  - 修复: 新增 `unquoteGitPath()` 函数处理引号和八进制转义
  - Fix: Added `unquoteGitPath()` function to handle quotes and octal escapes
  - 影响: 修复了1100+个文件无法正确同步删除状态的问题
  - Impact: Fixed 1100+ files unable to sync deletion status
  - 详细文档: `docs/fix/20251202-Git路径引号解析BUG修复.md`
  - Documentation: `docs/fix/20251202-Git路径引号解析BUG修复.md`

### v12.2 (2025-12-01)

**新增功能 / New Features**:
- ✅ 智能三路合并: 自动检测分支分叉并尝试智能合并
- ✅ Intelligent three-way merge: Auto-detect branch divergence and attempt smart merge
- ✅ 虚拟环境过滤: 特殊仓库处理时自动排除Python/Node虚拟环境目录
- ✅ Virtual environment filtering: Auto-exclude Python/Node virtual env directories
- ✅ 冲突自动解决: 自动解决锁文件(package-lock.json等)冲突
- ✅ Auto conflict resolution: Auto-resolve lock file conflicts
- ✅ 安全备份机制: 合并前自动创建备份分支，失败时自动回滚
- ✅ Safe backup mechanism: Auto-create backup branch before merge, auto-rollback on failure
