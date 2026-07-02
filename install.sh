#!/bin/bash
# Kunsec 一键安装脚本
# 支持在线安装（自动下载最新 Release）和离线安装（--local）
set -euo pipefail

echo "===================================="
echo "  Kunsec 安装脚本"
echo "===================================="

# ---------------------------------------------------------------------------
# 全局变量
# ---------------------------------------------------------------------------
BASE_URL="https://kunsec.ciyra.com/download"
BIN_NAME="sysmonitord"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/sysmonitord"
DATA_DIR="/var/lib/sysmonitord"
LOG_DIR="/var/log/sysmonitord"
TEMP_DIR="/tmp/kunsec-install-$$"
LOCAL_PKG=""
VERSION=""
ARCH=""
TTY="/dev/tty"

# ---------------------------------------------------------------------------
# 工具函数
# ---------------------------------------------------------------------------

check_root() {
    if [ "$EUID" -ne 0 ]; then
        echo "错误: 请使用 root 用户运行此安装脚本。"
        echo "提示: curl -fsSL $BASE_URL/install.sh | sudo bash"
        exit 1
    fi
}

ensure_tty() {
    if [ ! -r "$TTY" ]; then
        echo "错误: 当前环境没有可交互终端，无法进入配置向导。"
        echo "请在终端中执行，或使用离线安装后手动编辑 $CONFIG_DIR/config.yaml。"
        exit 1
    fi
}

prompt() {
    local __var="$1"
    local __prompt="$2"
    local __value=""
    read -r -p "$__prompt" __value < "$TTY"
    printf -v "$__var" '%s' "$__value"
}

prompt_secret() {
    local __var="$1"
    local __prompt="$2"
    local __value=""
    read -r -s -p "$__prompt" __value < "$TTY"
    echo "" > "$TTY"
    printf -v "$__var" '%s' "$__value"
}

detect_arch() {
    local raw
    raw=$(uname -m)
    case "$raw" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *)
            echo "错误: 不支持的系统架构: $raw"
            echo "Kunsec 目前仅支持 x86_64 (amd64) 和 arm64 架构。"
            exit 1
            ;;
    esac
    echo "检测到系统架构: $ARCH"
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --local)
                if [[ -n "${2:-}" ]]; then
                    LOCAL_PKG="$2"
                    shift 2
                else
                    echo "错误: --local 参数需要指定 tar.gz 文件路径"
                    exit 1
                fi
                ;;
            -h|--help)
                echo "用法:"
                echo "  在线安装: curl -fsSL $BASE_URL/install.sh | sudo bash"
                echo "  离线安装: sudo bash install.sh --local ./sysmonitord-vxxx-linux-amd64.tar.gz"
                exit 0
                ;;
            *)
                echo "未知参数: $1"
                echo "使用 -h 或 --help 查看用法"
                exit 1
                ;;
        esac
    done
}

fetch_latest_version() {
    echo "正在查询最新版本..."
    local url="$BASE_URL/latest"
    if ! VERSION=$(curl -fsSL --connect-timeout 10 "$url" 2>/dev/null); then
        echo "错误: 无法从服务器获取最新版本信息 ($url)"
        echo "请检查网络连接，或使用离线安装模式: --local <path>"
        exit 1
    fi
    VERSION=$(echo "$VERSION" | tr -d '[:space:]')
    echo "最新版本: $VERSION"
}

download_package() {
    local pkg_name="${BIN_NAME}-${VERSION}-linux-${ARCH}.tar.gz"
    local url="$BASE_URL/$pkg_name"
    local dest="$TEMP_DIR/$pkg_name"

    echo "正在下载 $pkg_name ..."
    if ! curl -fsSL --connect-timeout 30 --progress-bar "$url" -o "$dest"; then
        echo "错误: 下载失败 ($url)"
        exit 1
    fi
    echo "下载完成: $dest"
    LOCAL_PKG="$dest"
}

extract_package() {
    echo "正在解压安装包..."
    if ! tar xzf "$LOCAL_PKG" -C "$TEMP_DIR"; then
        echo "错误: 解压失败 ($LOCAL_PKG)"
        exit 1
    fi
    echo "解压完成"
}

install_binary() {
    local src="$TEMP_DIR/$BIN_NAME/$BIN_NAME"
    if [ ! -f "$src" ]; then
        echo "错误: 安装包内未找到可执行文件"
        exit 1
    fi
    echo "正在安装二进制文件..."
    cp "$src" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$BIN_NAME"
    echo "已安装: $INSTALL_DIR/$BIN_NAME"
}

setup_directories() {
    echo "正在创建系统目录..."
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$LOG_DIR"
    echo "目录创建完成:"
    echo "  配置: $CONFIG_DIR"
    echo "  数据: $DATA_DIR"
    echo "  日志: $LOG_DIR"
}

# ---------------------------------------------------------------------------
# 交互式配置向导
# ---------------------------------------------------------------------------

interactive_config() {
    ensure_tty

    echo ""
    echo "===================================="
    echo "  安装配置向导"
    echo "===================================="
    echo "直接按回车将采用方括号内的默认值。"
    echo ""

    # --- 审计服务器 ---
    echo "[1/4] 审计服务器配置 (用于集中接收监控日志)"
    local audit_enabled="y"
    prompt input "启用审计推送? [Y/n]: "
    if [[ "$input" =~ ^[Nn]$ ]]; then audit_enabled="n"; fi

    local audit_server="192.168.1.100"
    local audit_port="9000"
    if [[ "$audit_enabled" == "y" ]]; then
        prompt input "审计服务器地址 [$audit_server]: "
        [[ -n "$input" ]] && audit_server="$input"
        prompt input "审计服务器端口 [$audit_port]: "
        [[ -n "$input" ]] && audit_port="$input"
    fi

    # --- 邮件告警 ---
    echo ""
    echo "[2/4] 邮件告警配置"
    local mail_enabled="y"
    prompt input "启用邮件告警? [Y/n]: "
    if [[ "$input" =~ ^[Nn]$ ]]; then mail_enabled="n"; fi

    local smtp_server="smtp.example.com"
    local smtp_port="465"
    local smtp_user="sysmonitord@example.com"
    local smtp_pass=""
    local mail_recipients="admin@example.com"

    if [[ "$mail_enabled" == "y" ]]; then
        prompt input "SMTP 服务器 [$smtp_server]: "
        [[ -n "$input" ]] && smtp_server="$input"
        prompt input "SMTP 端口 [$smtp_port]: "
        [[ -n "$input" ]] && smtp_port="$input"
        prompt input "SMTP 用户名 [$smtp_user]: "
        [[ -n "$input" ]] && smtp_user="$input"
        prompt_secret smtp_pass "SMTP 密码/授权码: "
        prompt input "收件人邮箱 (多个用逗号分隔) [$mail_recipients]: "
        [[ -n "$input" ]] && mail_recipients="$input"
    fi

    # --- AI 分析 ---
    echo ""
    echo "[3/4] AI 分析配置"
    local ai_enabled="y"
    prompt input "启用 AI 智能分析? [Y/n]: "
    if [[ "$input" =~ ^[Nn]$ ]]; then ai_enabled="n"; fi

    local ai_url="https://api.openai.com/v1/chat/completions"
    local ai_key=""
    local ai_model="gpt-4o-mini"

    if [[ "$ai_enabled" == "y" ]]; then
        prompt input "AI API URL [$ai_url]: "
        [[ -n "$input" ]] && ai_url="$input"
        prompt_secret ai_key "AI API Key: "
        prompt input "AI 模型 [$ai_model]: "
        [[ -n "$input" ]] && ai_model="$input"
    fi

    # --- 扫描配置 ---
    echo ""
    echo "[4/4] 扫描配置"
    local scan_path="/"
    local proc_interval="300"
    prompt input "文件扫描根路径 [$scan_path]: "
    [[ -n "$input" ]] && scan_path="$input"
    prompt input "进程扫描间隔 (秒) [$proc_interval]: "
    [[ -n "$input" ]] && proc_interval="$input"

    echo ""
    echo "配置完成，正在写入 $CONFIG_DIR/config.yaml ..."

    # 构建收件人 YAML 列表
    local recip_yaml=""
    IFS=',' read -ra recip_arr <<< "$mail_recipients"
    for r in "${recip_arr[@]}"; do
        r=$(echo "$r" | xargs)  # trim
        if [[ -n "$r" ]]; then
            recip_yaml="${recip_yaml}      - $r\n"
        fi
    done

    # 生成配置
    cat > "$CONFIG_DIR/config.yaml" <<EOF
server:
  host: "127.0.0.1"
  port: 8080

log:
  level: "info"

audit:
  enabled: $([[ "$audit_enabled" == "y" ]] && echo "true" || echo "false")
  server: "$audit_server"
  port: $audit_port
  buffer_size: 1000

scanner:
  hash:
    algorithm: "xxhash64"
  file:
    include_paths:
      - $scan_path
    exclude_paths:
      # ========== 虚拟/临时文件系统 ==========
      - /proc
      - /sys
      - /dev
      - /tmp
      - /var/tmp
      - /run
      - /mnt
      - /media
    fast_hash: true
    fast_hash_size: 100MB
    fast_hash_chunk: 2MB
  process:
    interval: $proc_interval

storage:
  data_dir: "$DATA_DIR"
  process_system_file: "process_system.data"
  file_system_file: "file_system.data"
  dubious_file_list_file: "dubious_files.data"
  dubious_process_list_file: "dubious_processes.data"

notification:
  interval: 5
  email:
    enabled: $([[ "$mail_enabled" == "y" ]] && echo "true" || echo "false")
    recipients:
$recip_yaml
    smtp:
      server: $smtp_server
      port: $smtp_port
      username: $smtp_user
      password: "$smtp_pass"

script:
  enabled: true
  dir: "$CONFIG_DIR/scripts"
  timeout_ms: 5000
  events:
    system_start:
      - startup.js
    file_change:
      - file-change.js
    dubious_file:
      - dubious-file.js
    dubious_process:
      - dubious-process.js
    error:
      - error.js

ai:
  enabled: $([[ "$ai_enabled" == "y" ]] && echo "true" || echo "false")
  api_url: "$ai_url"
  api_key: "$ai_key"
  model: "$ai_model"
  timeout: 120
  report_dir: "$DATA_DIR/reports"
  max_file_size: 204800
  max_total_size: 2097152
  include_paths:
    - /etc/ssh/sshd_config
    - /etc/sudoers
    - /etc/passwd
    - /etc/group
    - /etc/login.defs
    - /etc/security/limits.conf
    - /etc/sysctl.conf
EOF

    # 若密码/Key 为空，写成空字符串，避免服务启动时因未设置环境变量而失败。
    if [[ -z "$smtp_pass" ]]; then
        sed -i 's/      password: ""/      password: ""/' "$CONFIG_DIR/config.yaml"
    fi
    if [[ -z "$ai_key" ]]; then
        sed -i 's/  api_key: ""/  api_key: ""/' "$CONFIG_DIR/config.yaml"
    fi

    # 设置适当权限
    chmod 600 "$CONFIG_DIR/config.yaml"
    echo "配置文件已写入: $CONFIG_DIR/config.yaml"
}

install_systemd() {
    local svc_src="$TEMP_DIR/$BIN_NAME/${BIN_NAME}.service"
    if [ ! -f "$svc_src" ]; then
        echo "警告: 安装包内未找到 systemd 服务文件，跳过服务安装"
        return
    fi
    echo "正在安装 systemd 服务..."
    cp "$svc_src" /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable "$BIN_NAME"
    systemctl start "$BIN_NAME"

    sleep 2
    if systemctl is-active --quiet "$BIN_NAME"; then
        echo "服务启动成功!"
    else
        echo "警告: 服务启动失败，请检查配置并手动启动:"
        echo "  journalctl -u $BIN_NAME -n 50 --no-pager"
    fi
}

cleanup() {
    if [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi
}

print_success() {
    local ip
    ip=$(hostname -I 2>/dev/null | awk '{print $1}' || echo "127.0.0.1")
    echo ""
    echo "===================================="
    echo "  Kunsec 安装完成!"
    echo "===================================="
    echo ""
    echo "版本: $VERSION"
    echo "架构: $ARCH"
    echo ""
    echo "安装路径:"
    echo "  二进制: $INSTALL_DIR/$BIN_NAME"
    echo "  配置:   $CONFIG_DIR/config.yaml"
    echo "  数据:   $DATA_DIR"
    echo "  日志:   $LOG_DIR"
    echo ""
    echo "服务管理命令:"
    echo "  启动:   systemctl start $BIN_NAME"
    echo "  停止:   systemctl stop $BIN_NAME"
    echo "  重启:   systemctl restart $BIN_NAME"
    echo "  状态:   systemctl status $BIN_NAME"
    echo "  日志:   journalctl -u $BIN_NAME -f"
    echo ""
    echo "如需修改配置，请编辑: $CONFIG_DIR/config.yaml"
    echo "然后执行: systemctl restart $BIN_NAME"
    echo ""
}

# ---------------------------------------------------------------------------
# 主流程
# ---------------------------------------------------------------------------

main() {
    check_root
    parse_args "$@"
    detect_arch

    mkdir -p "$TEMP_DIR"
    trap cleanup EXIT

    if [[ -n "$LOCAL_PKG" ]]; then
        # 离线模式
        echo "离线安装模式"
        if [ ! -f "$LOCAL_PKG" ]; then
            echo "错误: 指定的本地包不存在: $LOCAL_PKG"
            exit 1
        fi
        VERSION="local"
    else
        # 在线模式
        fetch_latest_version
        download_package
    fi

    extract_package
    install_binary
    setup_directories

    # 检查配置是否已存在
    if [ -f "$CONFIG_DIR/config.yaml" ]; then
        echo ""
        ensure_tty
        prompt confirm "配置文件已存在 ($CONFIG_DIR/config.yaml)，是否覆盖并重新配置? [y/N]: "
        if [[ "$confirm" =~ ^[Yy]$ ]]; then
            interactive_config
        else
            echo "保留现有配置文件。"
        fi
    else
        interactive_config
    fi

    install_systemd
    print_success
}

main "$@"
