#!/bin/bash
# =================================================================================
#  Git Auto-Sync 编译脚本 / Git Auto-Sync Build Script
# =================================================================================
#  用途 / Purpose: 编译GO版本的Git自动同步工具
#                  Build GO version of Git auto-sync tool
# =================================================================================

set -e  # 遇到错误立即退出 / Exit on error

# 颜色定义 / Color definitions
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${CYAN}==================================================================================${NC}"
echo -e "${CYAN}  Git Auto-Sync 编译脚本 / Build Script${NC}"
echo -e "${CYAN}==================================================================================${NC}"

# 检查GO是否已安装 / Check if GO is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误：GO未安装 / Error: GO is not installed${NC}"
    echo -e "${YELLOW}正在安装GO... / Installing GO...${NC}"
    
    if [ "$(id -u)" -ne 0 ]; then
        sudo apt-get update
        sudo apt-get install -y golang-go
    else
        apt-get update
        apt-get install -y golang-go
    fi
fi

# 显示GO版本 / Display GO version
GO_VERSION=$(go version)
echo -e "${GREEN}GO版本 / GO Version: ${GO_VERSION}${NC}"

# 获取项目根目录 / Get project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo -e "${CYAN}项目目录 / Project directory: ${SCRIPT_DIR}${NC}"

# 下载依赖 / Download dependencies
echo -e "${CYAN}下载依赖 / Downloading dependencies...${NC}"
go mod tidy

# 编译 / Build
echo -e "${CYAN}开始编译 / Starting build...${NC}"
OUTPUT_FILE="git-autosync"

# 编译参数说明 / Build parameters explanation:
# -o: 输出文件名 / Output file name
# -ldflags "-s -w": 去除调试信息，减小文件大小 / Strip debug info, reduce file size
go build -ldflags "-s -w" -o "$OUTPUT_FILE" cmd/git-autosync/main.go

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 编译成功 / Build successful${NC}"
    
    # 显示文件信息 / Display file info
    FILE_SIZE=$(du -h "$OUTPUT_FILE" | cut -f1)
    echo -e "${GREEN}输出文件 / Output file: ${OUTPUT_FILE}${NC}"
    echo -e "${GREEN}文件大小 / File size: ${FILE_SIZE}${NC}"
    
    # 添加可执行权限 / Add executable permission
    chmod +x "$OUTPUT_FILE"
    echo -e "${GREEN}已添加可执行权限 / Executable permission added${NC}"
    
    # 显示使用说明 / Display usage instructions
    echo -e "${CYAN}==================================================================================${NC}"
    echo -e "${CYAN}使用方法 / Usage:${NC}"
    echo -e "${YELLOW}  1. 在Git仓库根目录运行 / Run in git repository root:${NC}"
    echo -e "     cd /path/to/your/git/repo"
    echo -e "     ${SCRIPT_DIR}/${OUTPUT_FILE}"
    echo -e ""
    echo -e "${YELLOW}  2. 后台运行 / Run in background:${NC}"
    echo -e "     nohup ${SCRIPT_DIR}/${OUTPUT_FILE} > /tmp/git-autosync.log 2>&1 &"
    echo -e ""
    echo -e "${YELLOW}  3. 查看日志 / View logs:${NC}"
    echo -e "     tail -f /tmp/git-autosync.log"
    echo -e "${CYAN}==================================================================================${NC}"
else
    echo -e "${RED}✗ 编译失败 / Build failed${NC}"
    exit 1
fi
