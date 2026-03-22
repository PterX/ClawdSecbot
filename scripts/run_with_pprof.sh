#!/bin/bash
# run_with_pprof.sh — 构建并以 pprof 性能分析模式运行应用
#
# 用法:
#   ./scripts/run_with_pprof.sh              # 使用默认端口 6060
#   ./scripts/run_with_pprof.sh 9090         # 使用自定义端口 9090
#   BOTSEC_PPROF_PORT=8080 ./scripts/run_with_pprof.sh  # 通过环境变量指定端口
#
# pprof 常用命令:
#   go tool pprof http://127.0.0.1:6060/debug/pprof/heap          # 堆内存分析
#   go tool pprof http://127.0.0.1:6060/debug/pprof/profile?seconds=30  # CPU 分析 (30秒采样)
#   go tool pprof http://127.0.0.1:6060/debug/pprof/goroutine     # goroutine 分析
#   go tool pprof http://127.0.0.1:6060/debug/pprof/allocs        # 内存分配分析
#   go tool pprof http://127.0.0.1:6060/debug/pprof/block         # 阻塞分析
#   go tool pprof http://127.0.0.1:6060/debug/pprof/mutex         # 互斥锁竞争分析
#
# 浏览器查看:
#   open http://127.0.0.1:6060/debug/pprof/
#
set -e

PROJECT_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." &> /dev/null && pwd )"
cd "$PROJECT_ROOT"

# 确定 pprof 端口: 命令行参数 > 环境变量 > 默认值 6060
PPROF_PORT="${1:-${BOTSEC_PPROF_PORT:-6060}}"

echo "============================================"
echo "  BotSecManager — pprof 性能分析模式"
echo "============================================"
echo ""

# Step 1: 构建 Go 插件
echo "[1/2] 构建 Go 插件..."
"$PROJECT_ROOT/scripts/build_openclaw_plugin.sh"
echo ""

# Step 2: 检测操作系统并启动 Flutter 应用（带 pprof）
echo "[2/2] 启动 Flutter 应用（pprof 端口: $PPROF_PORT）..."
echo ""
echo "  pprof 地址: http://127.0.0.1:${PPROF_PORT}/debug/pprof/"
echo ""
echo "  常用分析命令:"
echo "    go tool pprof http://127.0.0.1:${PPROF_PORT}/debug/pprof/heap"
echo "    go tool pprof http://127.0.0.1:${PPROF_PORT}/debug/pprof/profile?seconds=30"
echo "    go tool pprof http://127.0.0.1:${PPROF_PORT}/debug/pprof/goroutine"
echo ""
echo "============================================"
echo ""

# 自动检测操作系统
if [[ "$OSTYPE" == "darwin"* ]]; then
    FLUTTER_DEVICE="macos"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    FLUTTER_DEVICE="linux"
elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
    FLUTTER_DEVICE="windows"
else
    echo "警告: 未能自动识别操作系统 ($OSTYPE)，尝试使用 linux"
    FLUTTER_DEVICE="linux"
fi

echo "检测到操作系统: $FLUTTER_DEVICE"
echo ""

# Linux 开发模式: 安装 .desktop 文件和多尺寸图标，使 GNOME 能正确显示应用名、
# 窗口图标和托盘图标。托盘需要小尺寸（22-48px），窗口/Dock 需要大尺寸。
if [[ "$FLUTTER_DEVICE" == "linux" ]]; then
    DESKTOP_DIR="$HOME/.local/share/applications"
    ICON_BASE="$HOME/.local/share/icons/hicolor"
    DESKTOP_FILE="$DESKTOP_DIR/com.clawdsecbot.guard.desktop"
    SOURCE_ICON="$PROJECT_ROOT/scripts/icon_1024.png"
    TRAY_ICON="$PROJECT_ROOT/images/tray_icon.png"

    if [ -f "$SOURCE_ICON" ]; then
        ICON_SIZES=(16 22 24 32 48 64 128 256)
        for SIZE in "${ICON_SIZES[@]}"; do
            mkdir -p "$ICON_BASE/${SIZE}x${SIZE}/apps"
        done

        if command -v convert &> /dev/null; then
            for SIZE in "${ICON_SIZES[@]}"; do
                convert "$SOURCE_ICON" -resize "${SIZE}x${SIZE}" \
                    "$ICON_BASE/${SIZE}x${SIZE}/apps/clawdsecbot.png"
            done
        else
            # 无 ImageMagick 时: 大尺寸用源图标，小尺寸用 64x64 托盘图标
            for SIZE in 128 256; do
                cp "$SOURCE_ICON" "$ICON_BASE/${SIZE}x${SIZE}/apps/clawdsecbot.png"
            done
            if [ -f "$TRAY_ICON" ]; then
                for SIZE in 16 22 24 32 48 64; do
                    cp "$TRAY_ICON" "$ICON_BASE/${SIZE}x${SIZE}/apps/clawdsecbot.png"
                done
            fi
        fi

        # 更新图标缓存，使 GNOME/GTK 立即识别新图标
        gtk-update-icon-cache -f -t "$ICON_BASE" 2>/dev/null || true

        mkdir -p "$DESKTOP_DIR"
        cat > "$DESKTOP_FILE" << DESKTOP_EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=ClawdSecbot
Comment=AI Bot Security Manager (Dev)
Icon=clawdsecbot
Terminal=false
Categories=Utility;Security;
StartupWMClass=com.clawdsecbot.guard
NoDisplay=true
DESKTOP_EOF
        echo "已安装开发用 .desktop 文件和多尺寸图标"
    fi
fi

export BOTSEC_PPROF_PORT="$PPROF_PORT"
# 禁用 Flutter 自动更新检查，避免网络问题
export FLUTTER_STORAGE_BASE_URL="https://storage.flutter-io.cn"
export PUB_HOSTED_URL="https://pub.flutter-io.cn"
export FLUTTER_GIT_URL="https://gitee.com/mirrors/flutter.git"
# 跳过 git fetch 操作
export FLUTTER_ALREADY_LOCKED=true

# 首次运行或缓存被清理时，必须先生成 package_config 和插件符号链接
if [ ! -f "$PROJECT_ROOT/.dart_tool/package_config.json" ]; then
    echo "Bootstrap Flutter dependencies: package_config.json is missing."
    flutter pub get
fi

exec flutter run -d "$FLUTTER_DEVICE" --no-pub
