#!/bin/bash

# 图标生成脚本
# 基于 app_icon_1024.png 生成所有需要的尺寸的图标

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 源图标文件
SOURCE_ICON="$PROJECT_ROOT/scripts/icon_1024.png"

# 目标目录
MACOS_ICONSET_DIR="$PROJECT_ROOT/macos/Runner/Assets.xcassets/AppIcon.appiconset"
TRAY_ICON_DIR="$PROJECT_ROOT/images"

# 检查源文件是否存在
if [ ! -f "$SOURCE_ICON" ]; then
    echo -e "${RED}错误: 源图标文件不存在: $SOURCE_ICON${NC}"
    exit 1
fi

# 检查是否安装了 sips (macOS 自带)
if ! command -v sips &> /dev/null; then
    echo -e "${RED}错误: 未找到 sips 命令 (macOS 自带工具)${NC}"
    exit 1
fi

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}开始生成图标...${NC}"
echo -e "${GREEN}========================================${NC}"

# 创建目标目录
mkdir -p "$MACOS_ICONSET_DIR"
mkdir -p "$TRAY_ICON_DIR"

# 生成 macOS 应用图标 (所有需要的尺寸)
echo -e "\n${YELLOW}[1/3] 生成 macOS 应用图标...${NC}"

# 定义所有需要的图标尺寸
ICON_SIZES=("16" "32" "64" "128" "256" "512" "1024")

for size in "${ICON_SIZES[@]}"; do
    filename="app_icon_${size}.png"
    output_path="$MACOS_ICONSET_DIR/$filename"
    
    echo "  生成 ${size}x${size} -> $filename"
    sips -z "$size" "$size" "$SOURCE_ICON" --out "$output_path" > /dev/null 2>&1
done

echo -e "${GREEN}  ✓ macOS 应用图标生成完成${NC}"

# 生成托盘图标 (PNG 格式, 彩色图标使用 64x64 更清晰)
echo -e "\n${YELLOW}[2/3] 生成托盘图标 (PNG)...${NC}"
TRAY_PNG="$TRAY_ICON_DIR/tray_icon.png"
sips -z 64 64 "$SOURCE_ICON" --out "$TRAY_PNG" > /dev/null 2>&1
echo -e "${GREEN}  ✓ 托盘图标 PNG 生成完成: tray_icon.png (64x64)${NC}"

# 生成 Windows ICO 图标 (托盘 + 应用)
echo -e "\n${YELLOW}[3/4] 生成 Windows ICO 图标...${NC}"

TEMP_ICONSET="$PROJECT_ROOT/temp_ico.iconset"
WIN_APP_ICON_DIR="$PROJECT_ROOT/windows/runner/resources"
mkdir -p "$TEMP_ICONSET"
mkdir -p "$WIN_APP_ICON_DIR"

# 为 ICO 生成多个尺寸
ICO_SIZES=("16" "32" "48" "64" "128" "256")
for size in "${ICO_SIZES[@]}"; do
    sips -z "$size" "$size" "$SOURCE_ICON" --out "$TEMP_ICONSET/icon_${size}.png" > /dev/null 2>&1
done

if command -v magick &> /dev/null || command -v convert &> /dev/null; then
    TRAY_ICO="$TRAY_ICON_DIR/tray_icon.ico"
    APP_ICO="$WIN_APP_ICON_DIR/app_icon.ico"
    MAGICK_CMD="magick"
    if ! command -v magick &> /dev/null; then
        MAGICK_CMD="convert"
    fi

    # 托盘图标: 16+32 (系统托盘常用尺寸)
    $MAGICK_CMD "$TEMP_ICONSET/icon_16.png" "$TEMP_ICONSET/icon_32.png" "$TRAY_ICO" 2>&1
    echo -e "${GREEN}  ✓ 托盘图标 ICO: tray_icon.ico${NC}"

    # Windows 应用图标: 包含所有尺寸 (任务栏/Alt-Tab/资源管理器)
    $MAGICK_CMD "$TEMP_ICONSET/icon_16.png" "$TEMP_ICONSET/icon_32.png" \
        "$TEMP_ICONSET/icon_48.png" "$TEMP_ICONSET/icon_64.png" \
        "$TEMP_ICONSET/icon_128.png" "$TEMP_ICONSET/icon_256.png" \
        "$APP_ICO" 2>&1
    echo -e "${GREEN}  ✓ Windows 应用图标: app_icon.ico${NC}"
else
    echo -e "${YELLOW}  ⚠ 未安装 ImageMagick, 跳过 ICO 生成${NC}"
    echo -e "${YELLOW}    命令: brew install imagemagick${NC}"
fi

rm -rf "$TEMP_ICONSET"

# 生成 ICNS 文件 (macOS 标准图标格式)
echo -e "\n${YELLOW}[4/4] 生成 ICNS 文件...${NC}"

ICNS_ICONSET="$PROJECT_ROOT/AppIcon.iconset"
ICNS_OUTPUT="$PROJECT_ROOT/macos/Runner/Assets.xcassets/AppIcon.appiconset/AppIcon.icns"

mkdir -p "$ICNS_ICONSET"

# 为 ICNS 生成所有必需的尺寸
sips -z 16 16     "$SOURCE_ICON" --out "$ICNS_ICONSET/icon_16x16.png" > /dev/null 2>&1
sips -z 32 32     "$SOURCE_ICON" --out "$ICNS_ICONSET/icon_16x16@2x.png" > /dev/null 2>&1
sips -z 32 32     "$SOURCE_ICON" --out "$ICNS_ICONSET/icon_32x32.png" > /dev/null 2>&1
sips -z 64 64     "$SOURCE_ICON" --out "$ICNS_ICONSET/icon_32x32@2x.png" > /dev/null 2>&1
sips -z 128 128   "$SOURCE_ICON" --out "$ICNS_ICONSET/icon_128x128.png" > /dev/null 2>&1
sips -z 256 256   "$SOURCE_ICON" --out "$ICNS_ICONSET/icon_128x128@2x.png" > /dev/null 2>&1
sips -z 256 256   "$SOURCE_ICON" --out "$ICNS_ICONSET/icon_256x256.png" > /dev/null 2>&1
sips -z 512 512   "$SOURCE_ICON" --out "$ICNS_ICONSET/icon_256x256@2x.png" > /dev/null 2>&1
sips -z 512 512   "$SOURCE_ICON" --out "$ICNS_ICONSET/icon_512x512.png" > /dev/null 2>&1
sips -z 1024 1024 "$SOURCE_ICON" --out "$ICNS_ICONSET/icon_512x512@2x.png" > /dev/null 2>&1

# 转换为 ICNS
iconutil -c icns "$ICNS_ICONSET" -o "$ICNS_OUTPUT"

# 清理临时文件
rm -rf "$ICNS_ICONSET"

echo -e "${GREEN}  ✓ ICNS 文件生成完成: AppIcon.icns${NC}"

# 显示生成的文件列表
echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}图标生成完成!${NC}"
echo -e "${GREEN}========================================${NC}"

echo -e "\n${YELLOW}生成的文件:${NC}"
echo "  应用图标目录: $MACOS_ICONSET_DIR"
echo "    - app_icon_16.png (16x16)"
echo "    - app_icon_32.png (32x32)"
echo "    - app_icon_64.png (64x64)"
echo "    - app_icon_128.png (128x128)"
echo "    - app_icon_256.png (256x256)"
echo "    - app_icon_512.png (512x512)"
echo "    - app_icon_1024.png (1024x1024)"
echo "    - AppIcon.icns (macOS 标准格式)"
echo ""
echo "  托盘图标目录: $TRAY_ICON_DIR"
echo "    - tray_icon.png (64x64, 用于 macOS/Linux)"
if [ -f "$TRAY_ICON_DIR/tray_icon.ico" ]; then
    echo "    - tray_icon.ico (16+32, 用于 Windows 托盘)"
fi
echo ""
echo "  Windows 应用图标: $WIN_APP_ICON_DIR"
if [ -f "$WIN_APP_ICON_DIR/app_icon.ico" ]; then
    echo "    - app_icon.ico (16~256, 用于任务栏/Alt-Tab/资源管理器)"
fi

echo -e "\n${GREEN}提示: 现在可以运行构建脚本或重新启动应用以查看新图标${NC}"

exit 0
