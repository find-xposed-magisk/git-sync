// Package logger / 日志记录器包
// Module: Multi-Level Logging System / 多级日志系统
// Function: Provides structured logging with file rotation and level filtering
//           提供结构化日志，支持文件轮转和级别过滤
// Author: Agent-Gpt-Astra-Pro
// Dependencies: os, io, sync, time, fmt, path/filepath
//
// Features / 特性:
// - Four log levels: DEBUG, INFO, WARN, ERROR / 四个日志级别
// - Colored terminal output / 彩色终端输出
// - File rotation (size-based) / 文件轮转 (基于大小)
// - Multi-level file writers / 分级文件写入器
// - Thread-safe / 线程安全

package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ANSI颜色代码 / ANSI color codes
const (
	ColorGreen  = "\033[0;32m"
	ColorCyan   = "\033[0;36m"
	ColorYellow = "\033[1;33m"
	ColorRed    = "\033[0;31m"
	ColorReset  = "\033[0m"
)

// LogLevel 日志级别
// Log level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// Logger 日志记录器
// Logger for logging messages
type Logger struct {
	enableColor bool
	level       LogLevel
	output      io.Writer
	multiWriter *MultiLevelWriter // 分级日志写入器 / Multi-level writer
	mu          sync.Mutex
}

// NewLogger 创建新的日志记录器
// Creates a new logger
func NewLogger(enableColor bool) *Logger {
	return &Logger{
		enableColor: enableColor,
		level:       INFO, // 默认INFO级别 / Default INFO level
		output:      os.Stdout,
	}
}

// SetLevel 设置日志级别
// Sets log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetOutput 设置输出目标
// Sets output target
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// SetMultiLevelWriter 设置分级日志写入器
// Sets multi-level log writer
func (l *Logger) SetMultiLevelWriter(w *MultiLevelWriter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.multiWriter = w
}

// colorize 为文本添加颜色
// Adds color to text
func (l *Logger) colorize(color, text string) string {
	if !l.enableColor {
		return text
	}
	return color + text + ColorReset
}

// log 通用日志输出方法
// Generic log output method
func (l *Logger) log(level LogLevel, levelStr, color, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// 检查日志级别
	// Check log level
	if level < l.level {
		return
	}
	
	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	
	// 输出到终端
	// Output to terminal
	var logLine string
	if l.enableColor && l.output == os.Stdout {
		logLine = fmt.Sprintf("%s [%s] %s\n",
			l.colorize(ColorCyan, "["+timestamp+"]"),
			levelStr,
			l.colorize(color, msg))
	} else {
		logLine = fmt.Sprintf("[%s] [%s] %s\n", timestamp, levelStr, msg)
	}
	
	fmt.Fprint(l.output, logLine)
	
	// 写入分级日志文件
	// Write to level-specific log file
	if l.multiWriter != nil {
		// 不带颜色的纯文本日志
		// Plain text log without color
		plainLog := fmt.Sprintf("[%s] [%s] %s\n", timestamp, levelStr, msg)
		l.multiWriter.WriteWithLevel(level, []byte(plainLog))
	}
}

// Debug 输出调试日志
// Outputs debug log
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, "DEBUG", ColorCyan, format, args...)
}

// Info 输出信息日志
// Outputs info log
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, "INFO ", ColorGreen, format, args...)
}

// Warn 输出警告日志
// Outputs warning log
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, "WARN ", ColorYellow, format, args...)
}

// Error 输出错误日志
// Outputs error log
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, "ERROR", ColorRed, format, args...)
}

// Phase 输出阶段标题
// Outputs phase title
func (l *Logger) Phase(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	msg := fmt.Sprintf(format, args...)
	
	// 终端输出 (带颜色)
	// Terminal output (with color)
	fmt.Println(l.colorize(ColorCyan, "--- "+msg+" ---"))
	
	// 文件输出 (纯文本)
	// File output (plain text)
	if l.multiWriter != nil {
		timestamp := time.Now().Format("15:04:05.000")
		plainLog := fmt.Sprintf("[%s] [PHASE] --- %s ---\n", timestamp, msg)
		l.multiWriter.WriteWithLevel(INFO, []byte(plainLog))
	}
}

// Timestamp 输出带时间戳的消息
// Outputs message with timestamp
func (l *Logger) Timestamp(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	
	// 终端输出 (带颜色)
	// Terminal output (with color)
	fmt.Println(l.colorize(ColorGreen, fmt.Sprintf("[%s] %s", timestamp, msg)))
	
	// 文件输出 (纯文本)
	// File output (plain text)
	if l.multiWriter != nil {
		plainLog := fmt.Sprintf("[%s] [CYCLE] %s\n", timestamp, msg)
		l.multiWriter.WriteWithLevel(INFO, []byte(plainLog))
	}
}

// RotatingFileWriter 日志轮转写入器
// Rotating file writer for logs
type RotatingFileWriter struct {
	filePath    string
	maxSize     int64 // 最大文件大小（字节）/ Max file size in bytes
	maxBackups  int   // 最大备份数量 / Max number of backups
	currentFile *os.File
	currentSize int64
	mu          sync.Mutex
}

// NewRotatingFileWriter 创建日志轮转写入器
// Creates a new rotating file writer
func NewRotatingFileWriter(filePath string, maxSizeMB int, maxBackups int) (*RotatingFileWriter, error) {
	w := &RotatingFileWriter{
		filePath:   filePath,
		maxSize:    int64(maxSizeMB) * 1024 * 1024,
		maxBackups: maxBackups,
	}
	
	// 创建日志目录
	// Create log directory
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}
	
	// 打开或创建日志文件
	// Open or create log file
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}
	
	w.currentFile = file
	
	// 获取当前文件大小
	// Get current file size
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat log file: %v", err)
	}
	w.currentSize = info.Size()
	
	return w, nil
}

// Write 实现io.Writer接口
// Implements io.Writer interface
func (w *RotatingFileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	// 检查是否需要轮转
	// Check if rotation is needed
	if w.currentSize+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	
	// 写入数据
	// Write data
	n, err = w.currentFile.Write(p)
	w.currentSize += int64(n)
	return n, err
}

// rotate 轮转日志文件
// Rotates log file
func (w *RotatingFileWriter) rotate() error {
	// 关闭当前文件
	// Close current file
	if w.currentFile != nil {
		w.currentFile.Close()
	}
	
	// 轮转备份文件
	// Rotate backup files
	for i := w.maxBackups - 1; i > 0; i-- {
		oldPath := fmt.Sprintf("%s.%d", w.filePath, i)
		newPath := fmt.Sprintf("%s.%d", w.filePath, i+1)
		
		if _, err := os.Stat(oldPath); err == nil {
			os.Rename(oldPath, newPath)
		}
	}
	
	// 重命名当前文件
	// Rename current file
	if _, err := os.Stat(w.filePath); err == nil {
		os.Rename(w.filePath, w.filePath+".1")
	}
	
	// 创建新文件
	// Create new file
	file, err := os.OpenFile(w.filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %v", err)
	}
	
	w.currentFile = file
	w.currentSize = 0
	
	return nil
}

// Close 关闭日志文件
// Closes log file
func (w *RotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.currentFile != nil {
		return w.currentFile.Close()
	}
	return nil
}

// MultiLevelWriter 多级别日志写入器
// Multi-level log writer
type MultiLevelWriter struct {
	debugWriter io.Writer
	infoWriter  io.Writer
	warnWriter  io.Writer
	errorWriter io.Writer
	currentLevel LogLevel
	mu sync.Mutex
}

// NewMultiLevelWriter 创建多级别日志写入器
// Creates a new multi-level log writer
func NewMultiLevelWriter(logDir string, maxSizeMB, maxBackups int) (*MultiLevelWriter, error) {
	// 创建日志目录
	// Create log directory
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}
	
	// 创建各级别日志文件写入器
	// Create writers for each log level
	debugWriter, err := NewRotatingFileWriter(filepath.Join(logDir, "debug.log"), maxSizeMB, maxBackups)
	if err != nil {
		return nil, fmt.Errorf("failed to create debug writer: %v", err)
	}
	
	infoWriter, err := NewRotatingFileWriter(filepath.Join(logDir, "info.log"), maxSizeMB, maxBackups)
	if err != nil {
		return nil, fmt.Errorf("failed to create info writer: %v", err)
	}
	
	warnWriter, err := NewRotatingFileWriter(filepath.Join(logDir, "warn.log"), maxSizeMB, maxBackups)
	if err != nil {
		return nil, fmt.Errorf("failed to create warn writer: %v", err)
	}
	
	errorWriter, err := NewRotatingFileWriter(filepath.Join(logDir, "error.log"), maxSizeMB, maxBackups)
	if err != nil {
		return nil, fmt.Errorf("failed to create error writer: %v", err)
	}
	
	return &MultiLevelWriter{
		debugWriter: debugWriter,
		infoWriter:  infoWriter,
		warnWriter:  warnWriter,
		errorWriter: errorWriter,
	}, nil
}

// WriteWithLevel 根据级别写入日志
// Writes log based on level
func (m *MultiLevelWriter) WriteWithLevel(level LogLevel, p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 根据级别写入到对应的文件
	// Write to corresponding file based on level
	switch level {
	case DEBUG:
		if m.debugWriter != nil {
			m.debugWriter.Write(p)
		}
	case INFO:
		if m.infoWriter != nil {
			m.infoWriter.Write(p)
		}
	case WARN:
		if m.warnWriter != nil {
			m.warnWriter.Write(p)
		}
	case ERROR:
		if m.errorWriter != nil {
			m.errorWriter.Write(p)
		}
	}
	
	return len(p), nil
}

// Close 关闭所有日志文件
// Closes all log files
func (m *MultiLevelWriter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if closer, ok := m.debugWriter.(io.Closer); ok {
		closer.Close()
	}
	if closer, ok := m.infoWriter.(io.Closer); ok {
		closer.Close()
	}
	if closer, ok := m.warnWriter.(io.Closer); ok {
		closer.Close()
	}
	if closer, ok := m.errorWriter.(io.Closer); ok {
		closer.Close()
	}
	
	return nil
}
