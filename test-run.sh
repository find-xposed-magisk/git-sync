#!/bin/bash
# =================================================================================
#  Git Auto-Sync 测试运行脚本 / Git Auto-Sync Test Run Script
# =================================================================================
#  用途 / Purpose: 在当前仓库测试运行一个周期
#                  Test run one cycle in current repository
# =================================================================================

set -e

# 颜色定义 / Color definitions
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${CYAN}==================================================================================${NC}"
echo -e "${CYAN}  Git Auto-Sync 测试运行 / Test Run${NC}"
echo -e "${CYAN}==================================================================================${NC}"

# 获取脚本目录 / Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 检查二进制文件是否存在 / Check if binary exists
BINARY="${SCRIPT_DIR}/git-autosync-static"
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}错误：找不到二进制文件 / Error: Binary not found: $BINARY${NC}"
    echo -e "${YELLOW}请先运行 ./build-static.sh 编译 / Please run ./build-static.sh first${NC}"
    exit 1
fi

# 切换到Git仓库根目录 / Change to git repository root
REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null)
if [ $? -ne 0 ]; then
    echo -e "${RED}错误：当前目录不在Git仓库中 / Error: Not in a git repository${NC}"
    exit 1
fi

echo -e "${GREEN}Git仓库根目录 / Git repository root: ${REPO_ROOT}${NC}"
cd "$REPO_ROOT"

# 显示当前状态 / Display current status
echo -e "${CYAN}==================================================================================${NC}"
echo -e "${CYAN}当前Git状态 / Current Git status:${NC}"
git status --short | head -20
echo -e "${CYAN}==================================================================================${NC}"

# 询问是否继续 / Ask to continue
echo -e "${YELLOW}警告：此操作将会：${NC}"
echo -e "${YELLOW}Warning: This operation will:${NC}"
echo -e "  1. 处理特殊仓库 / Process special repositories"
echo -e "  2. 暂存所有变更 / Stage all changes"
echo -e "  3. 提交变更 / Commit changes"
echo -e "  4. 推送到远程 / Push to remote"
echo -e ""
read -p "是否继续？(y/N) / Continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}已取消 / Cancelled${NC}"
    exit 0
fi

# 运行测试 / Run test
echo -e "${CYAN}==================================================================================${NC}"
echo -e "${CYAN}开始测试运行 / Starting test run...${NC}"
echo -e "${CYAN}==================================================================================${NC}"

# 设置超时 / Set timeout
timeout 120 "$BINARY" &
PID=$!

# 等待一个周期完成（60秒同步间隔 + 60秒处理时间）
# Wait for one cycle to complete (60s sync interval + 60s processing time)
echo -e "${YELLOW}等待一个同步周期完成（最多120秒）/ Waiting for one sync cycle (max 120s)...${NC}"
sleep 70

# 终止进程 / Terminate process
if ps -p $PID > /dev/null; then
    echo -e "${YELLOW}终止测试进程 / Terminating test process...${NC}"
    kill $PID 2>/dev/null || true
    wait $PID 2>/dev/null || true
fi

# 显示结果 / Display results
echo -e "${CYAN}==================================================================================${NC}"
echo -e "${CYAN}测试完成 / Test completed${NC}"
echo -e "${CYAN}==================================================================================${NC}"

echo -e "${GREEN}最新提交 / Latest commit:${NC}"
git log -1 --oneline

echo -e "${GREEN}当前状态 / Current status:${NC}"
git status --short | head -20

echo -e "${CYAN}==================================================================================${NC}"
echo -e "${GREEN}✓ 测试运行完成 / Test run completed${NC}"
echo -e "${CYAN}==================================================================================${NC}"
