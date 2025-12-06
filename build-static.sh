#!/bin/bash
# =================================================================================
#  Git Auto-Sync 静态编译脚本 / Git Auto-Sync Static Build Script
# =================================================================================
#  用途 / Purpose: 编译静态链接的GO二进制文件，无运行时依赖
#                  Build statically linked GO binary with no runtime dependencies
# =================================================================================

set -e  # 遇到错误立即退出 / Exit on error

# 颜色定义 / Color definitions
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${CYAN}==================================================================================${NC}"
echo -e "${CYAN}  Git Auto-Sync 静态编译脚本 / Static Build Script${NC}"
echo -e "${CYAN}==================================================================================${NC}"

# 检查GO是否已安装 / Check if GO is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误：GO未安装 / Error: GO is not installed${NC}"
    exit 1
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

# 静态编译 / Static build
echo -e "${CYAN}开始静态编译 / Starting static build...${NC}"
OUTPUT_FILE="git-autosync-static"

# 静态编译参数说明 / Static build parameters explanation:
# CGO_ENABLED=0: 禁用CGO，避免动态链接C库 / Disable CGO to avoid dynamic linking C libraries
# -ldflags "-s -w -extldflags '-static'": 
#   -s: 去除符号表 / Strip symbol table
#   -w: 去除DWARF调试信息 / Strip DWARF debug info
#   -extldflags '-static': 静态链接 / Static linking
# -a: 强制重新编译所有包 / Force rebuild all packages
# -installsuffix cgo: 使用不同的安装后缀 / Use different install suffix

echo -e "${YELLOW}编译选项 / Build options:${NC}"
echo -e "  - CGO_ENABLED=0 (禁用CGO / Disable CGO)"
echo -e "  - 静态链接 / Static linking"
echo -e "  - 去除调试信息 / Strip debug info"
echo -e "  - 强制重新编译 / Force rebuild"

CGO_ENABLED=0 go build \
    -a \
    -installsuffix cgo \
    -ldflags "-s -w -extldflags '-static'" \
    -o "$OUTPUT_FILE" \
    cmd/git-autosync/main.go

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 静态编译成功 / Static build successful${NC}"
    
    # 显示文件信息 / Display file info
    FILE_SIZE=$(du -h "$OUTPUT_FILE" | cut -f1)
    echo -e "${GREEN}输出文件 / Output file: ${OUTPUT_FILE}${NC}"
    echo -e "${GREEN}文件大小 / File size: ${FILE_SIZE}${NC}"
    
    # 添加可执行权限 / Add executable permission
    chmod +x "$OUTPUT_FILE"
    echo -e "${GREEN}已添加可执行权限 / Executable permission added${NC}"
    
    # 检查是否为静态链接 / Check if statically linked
    echo -e "${CYAN}==================================================================================${NC}"
    echo -e "${CYAN}依赖检查 / Dependency check:${NC}"
    
    if command -v ldd &> /dev/null; then
        echo -e "${YELLOW}运行 ldd 检查动态库依赖 / Running ldd to check dynamic library dependencies:${NC}"
        ldd "$OUTPUT_FILE" 2>&1 || echo -e "${GREEN}✓ 完全静态链接，无动态库依赖 / Fully statically linked, no dynamic library dependencies${NC}"
    fi
    
    if command -v file &> /dev/null; then
        echo -e "${YELLOW}文件类型 / File type:${NC}"
        file "$OUTPUT_FILE"
    fi
    
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
    echo -e ""
    echo -e "${GREEN}✓ 此二进制文件可在任何Linux系统上运行，无需额外依赖${NC}"
    echo -e "${GREEN}✓ This binary can run on any Linux system without additional dependencies${NC}"
    echo -e "${CYAN}==================================================================================${NC}"
else
    echo -e "${RED}✗ 静态编译失败 / Static build failed${NC}"
    exit 1
fi
