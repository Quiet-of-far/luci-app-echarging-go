#!/bin/bash
set -e

# 默认在当前目录查找名为 openwrt-sdk-* 的目录
SDK_DIR=""
if [ -n "$1" ]; then
    SDK_DIR="$1"
else
    SDK_DIR=$(find . -maxdepth 1 -type d -name "openwrt-sdk-*" | head -n 1)
fi

if [ -z "$SDK_DIR" ] || [ ! -d "$SDK_DIR" ]; then
    echo "错误: 未找到 OpenWrt SDK 目录。"
    echo "请将 OpenWrt SDK 解压到当前目录下，或者通过参数指定路径。"
    echo "用法: $0 [/path/to/openwrt-sdk]"
    exit 1
fi

SDK_DIR=$(realpath "$SDK_DIR")
echo "=> 使用 OpenWrt SDK 路径: $SDK_DIR"

# 从 SDK 中解析目标架构，以便交叉编译 Go 二进制文件
TARGET_DIR=$(find "$SDK_DIR/staging_dir" -maxdepth 1 -type d -name "target-*" | head -n 1)
if [ -z "$TARGET_DIR" ]; then
    echo "警告: 无法在 SDK 中找到 target 目录，将使用 amd64 架构编译。"
    TARGET_ARCH="x86_64"
else
    TARGET_ARCH=$(basename "$TARGET_DIR" | sed 's/target-//g' | cut -d'_' -f1,2)
    echo "=> 检测到 SDK 目标架构: $TARGET_ARCH"
fi

ORIG_DIR=$(pwd)
export GOOS=linux
export CGO_ENABLED=0

case "$TARGET_ARCH" in
    *x86_64*)
        export GOARCH=amd64
        ;;
    *i386* | *i486* | *i686*)
        export GOARCH=386
        ;;
    *aarch64*)
        export GOARCH=arm64
        ;;
    *arm*)
        export GOARCH=arm
        ;;
    *mipsel*)
        export GOARCH=mipsle
        export GOMIPS=softfloat
        ;;
    *mips*)
        export GOARCH=mips
        export GOMIPS=softfloat
        ;;
    *)
        echo "警告: 未知架构格式，将默认使用 amd64 编译。"
        export GOARCH=amd64
        ;;
esac

echo "=> 开始编译 Go 二进制文件 (GOOS=$GOOS GOARCH=$GOARCH)..."
go build -trimpath -ldflags="-s -w" -o luci-app-echarging/files/usr/bin/echarging .

echo "=> 链接项目到 SDK 的 package 目录..."
rm -rf "$SDK_DIR/package/luci-app-echarging"
ln -sf "$ORIG_DIR/luci-app-echarging" "$SDK_DIR/package/luci-app-echarging"

echo "=> 开始使用 SDK 编译 ipk 插件包..."
cd "$SDK_DIR"

make package/luci-app-echarging/compile V=s

echo "=> 检查生成的 ipk 文件..."
IPK_FILE=$(find bin/packages/ -name "luci-app-echarging_*.ipk" | head -n 1)

if [ -n "$IPK_FILE" ]; then
    cd "$ORIG_DIR"
    cp "$SDK_DIR/$IPK_FILE" .
    echo -e "\n================================================="
    echo "✅ 编译成功！成品 ipk 文件已复制到项目根目录："
    echo "$(pwd)/$(basename "$IPK_FILE")"
    echo "================================================="
else
    echo "❌ 编译失败：未找到生成的 ipk 文件。"
    exit 1
fi
