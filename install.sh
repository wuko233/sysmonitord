#!/bin/bash
# sysmonitord 安装脚本
set -e
echo "正在安装 sysmonitord..."

# 检测是否为 root 用户
if [ "$EUID" -ne 0 ]; then
    echo "请使用 root 用户运行此安装脚本。"
    exit 1
fi

# 路径设置
BIN_NAME="sysmonitord"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/sysmonitord"
DATA_DIR="/var/lib/sysmonitord"
LOG_DIR="/var/log/sysmonitord"

# 检查当前目录下是否存在编译好的文件
if [ ! -f "./$BIN_NAME" ]; then
    echo "错误: 在当前目录未找到编译好的 $BIN_NAME 文件"
    echo "请确保编译好的 $BIN_NAME 文件存在于当前目录"
    exit 1
fi

# 创建目录
echo "正在创建目录..."
mkdir -p "$CONFIG_DIR"
mkdir -p "$DATA_DIR"
mkdir -p "$LOG_DIR"

# 复制文件
echo "正在复制文件..."
cp "./$BIN_NAME" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/$BIN_NAME"

# 初始化配置文件
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    echo "==> 初始化配置文件..."
    if [ -f "./config.yaml.example" ]; then
        cp ./config.yaml.example $CONFIG_DIR/config.yaml
    else
        echo "警告: 未找到 config.yaml.example 文件，请手动创建配置文件"
    fi
else
    echo "==> 配置文件已存在，跳过覆盖..."
fi

# 安装systemd服务
if [ -f "./scripts/sysmonitord.service" ]; then
    echo "正在安装 systemd 服务..."
    cp ./scripts/sysmonitord.service /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable sysmonitord
else
    echo "警告: 未找到 systemd 服务文件，跳过服务安装"
fi

echo ""
echo "安装完成！"
echo ""
echo "配置文件路径: $CONFIG_DIR/config.yaml"
echo "数据目录: $DATA_DIR"
echo "日志目录: $LOG_DIR"
echo ""
echo "您可以使用以下命令来管理 sysmonitord 服务："
echo "启动: systemctl start sysmonitord"
echo "停止: systemctl stop sysmonitord"
echo "重启: systemctl restart sysmonitord"
echo "查看状态: systemctl status sysmonitord"
echo "查看日志: journalctl -u sysmonitord -f"