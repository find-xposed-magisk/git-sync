# Git Auto-Sync 模块/类功能记录
# Module/Class Function Records

本文件记录项目中所有模块和类的功能及对应路径，便于复用和维护。
This file records all modules and classes in the project with their functions and paths for reuse and maintenance.

---

## 核心模块 / Core Modules

### 1. 配置管理 / Configuration Management
**模块名**: config
**功能**: 全局配置管理，包括Git配置、LFS阈值、特殊仓库列表等
**Function**: Global configuration management, including Git config, LFS thresholds, special repo lists, etc.
**路径**: `internal/config/config.go`

**主要结构体 / Main Structures**:
- `Config`: 全局配置结构 / Global configuration structure
- `DefaultConfig()`: 返回默认配置 / Returns default configuration
- `VirtualEnvExcludePatterns`: 虚拟环境排除规则 / Virtual environment exclusion patterns
- `LockFilePatterns`: 锁文件模式 / Lock file patterns

---

### 2. 批量处理框架 / Batch Processing Framework
**模块名**: batch
**功能**: 统一的批量处理框架，智能文件分类、动态批次大小、性能监控
**Function**: Unified batch processing framework with intelligent file classification, dynamic batch sizing, and performance monitoring
**路径**: `internal/batch/batch.go`

**主要结构体 / Main Structures**:
- `BatchConfig`: 批量处理配置 / Batch processing configuration
  - `SmallFileThreshold`: 小文件阈值 (5MB) / Small file threshold
  - `MediumFileThreshold`: 中文件阈值 (100MB) / Medium file threshold
  - `BatchSize`: 批次大小 / Batch size
  - `MaxWorkers`: 最大并发数 / Max workers
  - `EnableProgress`: 启用进度反馈 / Enable progress feedback
  - `EnableMetrics`: 启用性能监控 / Enable performance metrics

- `PerformanceMetrics`: 性能指标 / Performance metrics
  - `TotalFiles`: 总文件数 / Total files
  - `ProcessedFiles`: 已处理文件数 / Processed files
  - `FailedFiles`: 失败文件数 / Failed files
  - `TotalDuration`: 总耗时 / Total duration
  - `AvgBatchTime`: 平均批次耗时 / Average batch time
  - `BatchCount`: 批次数 / Batch count

- `GitBatchProcessor`: Git批量处理器 / Git batch processor

**主要方法 / Main Methods**:
- `NewGitBatchProcessor()`: 创建批量处理器 / Creates batch processor
- `NewGitBatchProcessorWithConfig()`: 使用自定义配置创建 / Creates with custom config
- `BatchAdd()`: 智能批量添加文件 / Intelligent batch add files
- `BatchRemove()`: 智能批量删除文件 / Intelligent batch remove files
- `ClassifyFilesBySize()`: 按大小分类文件 / Classifies files by size
- `calculateDynamicBatchSize()`: 动态计算批次大小 / Calculates dynamic batch size
- `GetMetrics()`: 获取性能指标 / Gets performance metrics
- `ResetMetrics()`: 重置性能指标 / Resets performance metrics

**核心特性 / Core Features**:
1. **智能文件分类** / Intelligent File Classification
   - 小文件 (<5MB): 并行处理 / Small files: Parallel processing
   - 中文件 (5-100MB): 批量处理 / Medium files: Batch processing
   - 大文件 (>100MB): 串行处理 / Large files: Serial processing

2. **动态批次大小** / Dynamic Batch Sizing
   - 根据文件总数和平均大小自动调整 / Auto-adjusts based on file count and average size
   - <50个文件: 一次处理完 / <50 files: Process all at once
   - 小文件: 大批次(200) / Small files: Large batch (200)
   - 大文件: 小批次(50) / Large files: Small batch (50)

3. **性能监控** / Performance Monitoring
   - 实时进度反馈 / Real-time progress feedback
   - 详细性能指标 / Detailed performance metrics
   - 平均批次耗时统计 / Average batch time statistics

4. **容错机制** / Fault Tolerance
   - 单批失败不影响整体 / Single batch failure doesn't affect overall
   - 自动重试机制 / Automatic retry mechanism
   - 失败文件记录 / Failed file tracking

**性能数据 / Performance Data**:
- 批量删除2670个文件: 6.6秒 / Batch remove 2670 files: 6.6s
- 对比优化前: 214.7秒 → 6.6秒 (提升97%) / vs before: 214.7s → 6.6s (97% improvement)
- 平均批次耗时: ~247ms/批 / Average batch time: ~247ms/batch

---

### 3. Git操作封装 / Git Operations Wrapper
**模块名**: git
**功能**: 封装所有Git命令操作，提供统一接口
**Function**: Wraps all Git command operations, provides unified interface
**路径**: `internal/git/git.go`

**主要方法 / Main Methods**:
- `NewGitOps()`: 创建Git操作实例 / Creates Git operations instance
- `EnsureDependencies()`: 确保依赖已安装 / Ensures dependencies are installed
- `HashObject()`: 计算文件哈希 / Computes file hash
- `UpdateIndex()`: 更新Git索引 / Updates git index
- `LFSTrack()`: LFS追踪 / LFS tracking
- `Add()`: 添加文件 / Adds file
- `Remove()`: 删除文件 / Removes file
- `Commit()`: 提交变更 / Commits changes
- `Fetch()`: 获取远程更新 / Fetches remote updates
- `Push()`: 推送到远程 / Pushes to remote
- `Pull()`: 从远程拉取 / Pulls from remote
- `Merge()`: 合并分支 / Merges branch
- `GetConflictedFiles()`: 获取冲突文件 / Gets conflicted files
- `CheckoutTheirs()`: 使用远程版本 / Uses remote version

---

### 4. 文件处理 / File Processing
**模块名**: file
**功能**: 文件暂存、大小检测、空目录处理
**Function**: File staging, size detection, empty directory handling
**路径**: `internal/file/file.go`

**主要方法 / Main Methods**:
- `NewFileProcessor()`: 创建文件处理器 / Creates file processor
- `StageFile()`: 暂存单个文件（带大小检测）/ Stages single file (with size detection)
- `HandleEmptyDirectories()`: 处理空目录 / Handles empty directories
- `IsInSpecialRepo()`: 检查是否在特殊仓库中 / Checks if in special repository
- `addToIgnoreFile()`: 添加到忽略文件 / Adds to ignore file

---

### 5. 特殊仓库处理 / Special Repository Processing
**模块名**: subrepo
**功能**: 处理包含.git目录的子仓库，高性能并发处理
**Function**: Processes sub-repositories containing .git directories, high-performance concurrent processing
**路径**: `internal/subrepo/subrepo.go`

**主要方法 / Main Methods**:
- `NewSubrepoProcessor()`: 创建特殊仓库处理器 / Creates special repo processor
- `ProcessAllSubrepos()`: 处理所有特殊仓库 / Processes all special repositories
- `processSpecialRepoFastAndSafe()`: 高性能安全处理 / High-performance safe processing
- `collectWorkFiles()`: 收集工作文件（排除虚拟环境）/ Collects work files (excluding virtual envs)
- `collectGitFiles()`: 收集.git文件 / Collects .git files
- `processWorkFile()`: 处理工作文件 / Processes work file
- `processGitFile()`: 处理.git文件（转换为gitdir）/ Processes .git file (converts to gitdir)
- `CleanOrphanedGitdirs()`: 清理孤儿gitdir / Cleans orphaned gitdirs
- `unquoteGitPath()`: 去除Git引号并解码八进制转义 / Removes Git quotes and decodes octal escapes
- `isOctalDigit()`: 检查是否为八进制数字 / Checks if octal digit

**核心特性 / Core Features**:
- 并发处理（goroutine池）/ Concurrent processing (goroutine pool)
- 虚拟环境过滤 / Virtual environment filtering
- .git → gitdir 转换 / .git → gitdir conversion
- 安全备份与恢复 / Safe backup and recovery
- 特殊字符路径处理 / Special character path handling (v12.3.2)

---

### 6. 智能合并 / Intelligent Merge
**模块名**: merge
**功能**: 智能三路合并，自动冲突解决
**Function**: Intelligent three-way merge, automatic conflict resolution
**路径**: `internal/merge/merge.go`

**主要方法 / Main Methods**:
- `NewMergeManager()`: 创建合并管理器 / Creates merge manager
- `SmartThreeWayMerge()`: 智能三路合并 / Intelligent three-way merge

**处理场景 / Handling Scenarios**:
1. 本地=远程 → 无操作 / Local=Remote → No action
2. 本地落后 → 快进合并 / Local behind → Fast-forward
3. 本地领先 → 推送 / Local ahead → Push
4. 分支分叉 → 三路合并 / Diverged → Three-way merge
   - 自动合并 / Auto merge
   - 智能冲突解决（锁文件）/ Intelligent conflict resolution (lock files)
   - 备份与回滚 / Backup and rollback

---

### 7. 日志记录 / Logging
**模块名**: logger
**功能**: 多级结构化日志系统，支持文件轮转和级别过滤
**Function**: Multi-level structured logging system with file rotation and level filtering
**路径**: `internal/logger/logger.go`

**核心特性 / Core Features**:
- 四个日志级别: DEBUG, INFO, WARN, ERROR / Four log levels
- 彩色终端输出 / Colored terminal output
- 文件轮转 (基于大小) / File rotation (size-based)
- 分级文件写入器 / Multi-level file writers
- 线程安全 / Thread-safe

**主要方法 / Main Methods**:
- `NewLogger()`: 创建日志记录器 / Creates logger
- `SetLevel()`: 设置日志级别 / Sets log level
- `SetMultiLevelWriter()`: 设置分级写入器 / Sets multi-level writer
- `Info()`: 信息日志 (关键业务事件) / Info log (key business events)
- `Debug()`: 调试日志 (详细执行过程) / Debug log (detailed execution)
- `Warn()`: 警告日志 (异常但可恢复) / Warning log (abnormal but recoverable)
- `Error()`: 错误日志 (实际错误) / Error log (actual errors)
- `Phase()`: 阶段标题 (同时写入文件) / Phase title (writes to file)
- `Timestamp()`: 带时间戳的消息 (同时写入文件) / Message with timestamp (writes to file)

**日志级别使用原则 / Log Level Usage Principles**:
- **DEBUG**: 详细的执行过程，越多越好 / Detailed execution process, more is better
  - 批次处理进度 / Batch processing progress
  - 文件处理细节 / File processing details
  - hash计算/缓存使用 / Hash calculation/cache usage
  - 虚拟环境排除详情 / Virtual env exclusion details
- **INFO**: 重要的业务事件和里程碑 / Important business events and milestones
  - 阶段开始/结束 / Phase start/end
  - 操作结果总结 / Operation result summary
  - 批量处理完成 / Batch processing complete
- **WARN**: 异常但可恢复的情况 / Abnormal but recoverable situations
  - 降级处理 / Degraded processing
  - 重试成功 / Retry successful
  - **不应该用于正常操作** / Should NOT be used for normal operations
- **ERROR**: 实际的错误和失败 / Actual errors and failures
  - 操作失败 / Operation failed
  - 数据异常 / Data anomaly
  - **不应该用于正常操作** / Should NOT be used for normal operations

**日志格式符号 / Log Format Symbols**:
- `✓` 成功操作 / Successful operation
- `✗` 跳过/排除 / Skip/exclude
- `⚠` 警告 / Warning
- `↳` 详细信息缩进 / Detailed info indent
- `↻` 计算/处理中 / Computing/processing

**性能数据 / Performance Data** (v12.3):
- DEBUG日志: 31,112行 (1.3MB) - 详细的执行过程
- INFO日志: 43行 (2.0KB) - 关键业务事件
- WARN日志: 0行 - 无警告
- ERROR日志: 0行 - 无错误

---

## 主程序 / Main Program

### 8. 主循环控制 / Main Loop Control
**模块名**: main
**功能**: 主循环逻辑，协调各模块工作
**Function**: Main loop logic, coordinates all modules
**路径**: `cmd/git-autosync/main.go`

**主要流程 / Main Flow**:
1. 初始化配置和模块 / Initialize config and modules
2. 确保依赖 / Ensure dependencies
3. 主循环（60秒） / Main loop (60s)
   - 阶段1: 特殊仓库处理 / Phase 1: Special repo processing
   - 阶段1.5: 清理孤儿gitdir / Phase 1.5: Clean orphaned gitdir
   - 阶段2: .gitignore清理 / Phase 2: .gitignore cleanup
   - 阶段3: 常规文件处理 / Phase 3: Regular file processing
   - 提交 / Commit
   - 阶段4: 智能合并 / Phase 4: Intelligent merge

**辅助函数 / Helper Functions**:
- `cleanIgnoredFiles()`: 清理被忽略的文件 / Cleans ignored files
- `isSpecialRepo()`: 检查是否为特殊仓库 / Checks if special repo
- `processDeletedFiles()`: 处理删除的文件 / Processes deleted files
- `processModifiedFiles()`: 处理修改的文件 / Processes modified files

---

## 性能优化点 / Performance Optimization Points

### 1. 并发处理 / Concurrent Processing
- 使用goroutine池（4个worker）/ Uses goroutine pool (4 workers)
- 信号量控制并发数 / Semaphore controls concurrency
- 批量操作减少Git调用 / Batch operations reduce Git calls

### 2. 内存优化 / Memory Optimization
- 流式处理大文件列表 / Streaming processing of large file lists
- 及时释放临时数据 / Timely release of temporary data
- 避免全量加载 / Avoid full loading

### 3. IO优化 / IO Optimization
- 批量Git操作 / Batch Git operations
- 减少文件系统调用 / Reduce filesystem calls
- 使用底层Git命令 / Use low-level Git commands

---

## 设计模式 / Design Patterns

### 1. 单一职责原则 / Single Responsibility Principle
每个模块只负责一项功能
Each module is responsible for only one function

### 2. 依赖注入 / Dependency Injection
通过构造函数注入依赖
Dependencies injected through constructors

### 3. 工厂模式 / Factory Pattern
使用New*()函数创建实例
Use New*() functions to create instances

### 4. 策略模式 / Strategy Pattern
不同场景使用不同处理策略
Different scenarios use different processing strategies

---

## 扩展指南 / Extension Guide

### 添加新的文件处理策略 / Add New File Processing Strategy
1. 在`internal/file/file.go`中添加新方法
   Add new method in `internal/file/file.go`
2. 在主循环中调用
   Call in main loop

### 添加新的合并策略 / Add New Merge Strategy
1. 在`internal/merge/merge.go`中扩展`SmartThreeWayMerge()`
   Extend `SmartThreeWayMerge()` in `internal/merge/merge.go`
2. 添加新的冲突解决规则
   Add new conflict resolution rules

### 添加新的配置项 / Add New Configuration Item
1. 在`internal/config/config.go`中添加字段
   Add field in `internal/config/config.go`
2. 在`DefaultConfig()`中设置默认值
   Set default value in `DefaultConfig()`
3. 在相关模块中使用
   Use in related modules

---

## 测试覆盖 / Test Coverage

### 单元测试 / Unit Tests
- [ ] config模块测试 / config module tests
- [ ] git模块测试 / git module tests
- [ ] file模块测试 / file module tests
- [ ] subrepo模块测试 / subrepo module tests
- [ ] merge模块测试 / merge module tests

### 集成测试 / Integration Tests
- [ ] 完整同步流程测试 / Full sync flow tests
- [ ] 冲突解决测试 / Conflict resolution tests
- [ ] 大文件处理测试 / Large file handling tests

---

## 维护日志 / Maintenance Log

### v12.3.2 (2025-12-02)
- 🐛 **修复Git路径引号解析BUG** / Fixed Git path quote parsing BUG
  - 问题: `git ls-files -s` 对特殊字符路径添加引号和八进制转义，未正确解析
  - Issue: `git ls-files -s` adds quotes and octal escapes for special char paths, not parsed correctly
  - 修复: 新增 `unquoteGitPath()` 和 `isOctalDigit()` 辅助函数
  - Fix: Added `unquoteGitPath()` and `isOctalDigit()` helper functions
  - 效果: 1100+ 个中文路径文件从删除状态恢复正常同步
  - Effect: 1100+ Chinese path files restored from deletion status to normal sync
  - 文档: `docs/fix/20251202-Git路径引号解析BUG修复.md`

### v12.3.1 (2024-11-30)
- ✅ 日志系统全面优化 / Comprehensive logging system optimization
  - 修复Phase()和Timestamp()未写入文件的问题 / Fixed Phase() and Timestamp() not writing to file
  - 修正WARN/ERROR级别滥用 (正常操作改为INFO) / Fixed WARN/ERROR level misuse (normal operations changed to INFO)
  - 增强batchRemoveFiles()的详细日志 / Enhanced batchRemoveFiles() detailed logging
  - 增强collectWorkFiles()虚拟环境排除详情 / Enhanced collectWorkFiles() virtual env exclusion details
  - 增强processWorkFile()文件处理细节 / Enhanced processWorkFile() file processing details
  - 添加logger.go文件头注释 / Added logger.go file header comments
  - 统一日志格式符号 (✓/✗/⚠/↳/↻) / Unified log format symbols
  - DEBUG日志增加380% (6.5K→ 31K行) / DEBUG logs increased by 380%
  - WARN/ERROR日志归零 (5→ 0行) / WARN/ERROR logs reduced to zero

### v12.3 (2024-11-30)
- ✅ 创建统一批量处理框架 / Created unified batch processing framework
- ✅ 实现智能文件分类 / Implemented intelligent file classification
- ✅ 实现动态批次大小 / Implemented dynamic batch sizing
- ✅ 实现性能监控系统 / Implemented performance monitoring system
- ✅ 批量删除性能提升97% / Batch remove performance improved by 97%

### v12.2 (2024-11-30)
- ✅ 完成Shell到GO的完整复刻 / Completed full rewrite from Shell to GO
- ✅ 实现高性能并发处理 / Implemented high-performance concurrent processing
- ✅ 实现智能三路合并 / Implemented intelligent three-way merge
- ✅ 实现虚拟环境过滤 / Implemented virtual environment filtering

---

**最后更新 / Last Updated**: 2025-12-02
**维护者 / Maintainer**: Agent-Gpt-Astra-Pro
