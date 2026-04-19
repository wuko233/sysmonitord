#!/bin/bash
# sysmonitord 卸载脚本
set -e

echo "正在卸载 sysmonitord..."

# 检测是否为 root 用户
if [ "$EUID" -ne 0 ]; then
    echo "请使用 root 用户运行此卸载脚本。"
    exit 1
fi

# 路径设置
BIN_NAME="sysmonitord"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/sysmonitord"
DATA_DIR="/var/lib/sysmonitord"
LOG_DIR="/var/log/sysmonitord"
SERVICE_FILE="/etc/systemd/system/sysmonitord.service"

# 停止并禁用服务
if systemctl is-active --quiet sysmonitord; then
    echo "正在停止 sysmonitord 服务..."
    systemctl stop sysmonitord
fi

if systemctl is-enabled --quiet sysmonitord 2>/dev/null; then
    echo "正在禁用 sysmonitord 服务..."
    systemctl disable sysmonitord
fi

# 删除 systemd 服务文件
if [ -f "$SERVICE_FILE" ]; then
    echo "正在删除 systemd 服务文件..."
    rm -f "$SERVICE_FILE"
    systemctl daemon-reload
fi

# 删除可执行文件
if [ -f "$INSTALL_DIR/$BIN_NAME" ]; then
    echo "正在删除可执行文件..."
    rm -f "$INSTALL_DIR/$BIN_NAME"
fi

# 询问是否删除数据文件
echo ""
read -p "是否删除所有数据文件（包括配置、数据和日志）？[y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "正在删除数据文件..."
    rm -rf "$CONFIG_DIR"
    rm -rf "$DATA_DIR"
    rm -rf "$LOG_DIR"
    echo "数据文件已删除。"
else
    echo "保留数据文件。"
    echo "配置文件目录: $CONFIG_DIR"
    echo "数据目录: $DATA_DIR"
    echo "日志目录: $LOG_DIR"
fi

echo ""
echo "sysmonitord 卸载完成！"