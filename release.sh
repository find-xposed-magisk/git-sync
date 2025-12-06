#!/bin/bash
# Release script - Reads version from VERSION file and creates git tag
# 发布脚本 - 从 VERSION 文件读取版本号并创建 git tag
#
# Usage / 使用方法:
#   ./release.sh         # Read from VERSION and release
#   ./release.sh v2.1.0  # Override with custom version
#
# Created by: Agent-Gpt-Astra-Pro

set -e

# Colors / 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Get script directory / 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  Git Sync Release Script${NC}"
echo -e "${CYAN}========================================${NC}"

# Determine version / 确定版本号
if [ -n "$1" ]; then
    VERSION="$1"
    echo -e "${YELLOW}Using provided version: ${VERSION}${NC}"
else
    # Read from VERSION file / 从 VERSION 文件读取
    if [ ! -f "VERSION" ]; then
        echo -e "${RED}ERROR: VERSION file not found!${NC}"
        echo -e "${YELLOW}Create VERSION file with content like: v2.1.0${NC}"
        exit 1
    fi
    
    VERSION=$(head -1 VERSION | tr -d '[:space:]')
    echo -e "${YELLOW}Read version from VERSION file: ${VERSION}${NC}"
fi

# Validate version format / 验证版本格式
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$ ]]; then
    echo -e "${RED}ERROR: Invalid version format: ${VERSION}${NC}"
    echo -e "${YELLOW}Expected format: vMAJOR.MINOR.PATCH (e.g., v2.1.0, v2.1.0-beta)${NC}"
    exit 1
fi

echo ""

# Check if tag already exists / 检查标签是否已存在
if git tag -l "$VERSION" | grep -q "$VERSION"; then
    echo -e "${RED}ERROR: Tag ${VERSION} already exists!${NC}"
    echo -e "${YELLOW}To delete and recreate:${NC}"
    echo -e "  git tag -d ${VERSION}"
    echo -e "  git push origin :refs/tags/${VERSION}"
    exit 1
fi

# Check for uncommitted changes / 检查未提交的变更
if ! git diff-index --quiet HEAD --; then
    echo -e "${YELLOW}WARNING: You have uncommitted changes!${NC}"
    git status --short
    echo ""
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Aborted.${NC}"
        exit 1
    fi
fi

# Confirm release / 确认发布
echo -e "${CYAN}Ready to release:${NC}"
echo -e "  Version: ${GREEN}${VERSION}${NC}"
echo -e "  Branch:  $(git branch --show-current)"
echo -e "  Commit:  $(git rev-parse --short HEAD)"
echo ""
read -p "Create tag and trigger release? (y/N) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${RED}Aborted.${NC}"
    exit 0
fi

# Create and push tag / 创建并推送标签
echo ""
echo -e "${CYAN}Creating tag ${VERSION}...${NC}"
git tag "$VERSION"

echo -e "${CYAN}Pushing tag to origin...${NC}"
git push origin "$VERSION"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Release ${VERSION} triggered!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}Check build status:${NC}"
echo -e "  https://github.com/find-xposed-magisk/git-sync/actions"
echo ""
echo -e "${YELLOW}View release when ready:${NC}"
echo -e "  https://github.com/find-xposed-magisk/git-sync/releases/tag/${VERSION}"
echo ""
