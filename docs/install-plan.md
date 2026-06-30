# sysmonitord 一键安装脚本实施方案

## 一、实施目标

1. 用户通过 `curl` 获取脚本并执行，即可完成全流程安装。
2. 脚本支持**在线模式**（自动下载最新 Release）和**离线模式**（使用本地 tar.gz 包）。
3. 安装过程中通过交互式向导配置核心参数（审计服务器、告警邮箱、AI 等）。
4. 安装完成后自动注册并启动 systemd 服务。

---

## 二、托管服务器目录结构

在服务器`https://kunsec.ciyra.com/download/`上部署以下文件：

```
https://kunsec.ciyra.com/download/
├── install.sh                              # 一键安装脚本
├── latest                                  # 最新版本号
├── sysmonitord-v1.2.0-linux-amd64.tar.gz   # amd64 架构二进制包
├── sysmonitord-v1.2.0-linux-arm64.tar.gz   # arm64 架构二进制包
└── checksums.txt                           # 各包的 sha256sum
```

---

## 三、Release 包 (tar.gz) 内容规范 (未来)

每个 `tar.gz` 解压后应包含：

```
sysmonitord/                  # 包内目录
├── sysmonitord               # 可执行二进制
├── sysmonitord.service       # systemd 服务文件
└── config.yaml.example       # 配置模板
```

---

## 四、一键安装脚本 (`install.sh`) 设计

脚本将完全重写现有的 `install.sh`，并存放在仓库根目录，最终上传到服务器。

### 4.1 脚本全局变量

- `BASE_URL="https://kunsec.ciyra.com/download/"`
- `INSTALL_DIR="/usr/local/bin"`
- `CONFIG_DIR="/etc/sysmonitord"`
- `DATA_DIR="/var/lib/sysmonitord"`
- `LOG_DIR="/var/log/sysmonitord"`
- `TEMP_DIR="/tmp/sysmonitord-install-$$"`

### 4.2 命令行参数

```bash
curl -fsSL https://kunsec.ciyra.com/download/install.sh | sudo bash

# 离线安装
sudo bash install.sh --local ./sysmonitord-vxxx-linux-amd64.tar.gz
```

### 4.3 交互式配置向导设计

将基于 `config.yaml.example` 进行交互式询问，使用 `read -p` 进行标准输入读取。为提升体验，脚本将先显示当前设置概览，允许用户逐项修改。

**交互流程示例**：

```text
====================================
  sysmonitord 安装配置向导
====================================

请输入审计服务器配置 (用于集中接收审计日志):
审计服务器地址 [192.168.1.100]: 10.0.0.5
审计服务器端口 [9000]:

请输入邮件告警配置:
SMTP 服务器 [smtp.example.com]: smtp.qq.com
SMTP 端口 [465]:
发件人邮箱 [sysmonitord@example.com]: monitor@yourdomain.com
发件人密码/授权码: ********
收件人邮箱 (多个用逗号分隔) [admin@example.com]: admin@yourdomain.com,security@yourdomain.com

请输入 AI 分析配置:
AI API URL [https://api.openai.com/v1/chat/completions]:
AI API Key: sk-xxxxxxxx
AI 模型 [gpt-4o-mini]:

请输入文件扫描配置:
扫描根路径 [/]:
进程扫描间隔 (秒) [300]:

====================================
配置完成，正在写入 /etc/sysmonitord/config.yaml ...
====================================
```

**脚本内部处理逻辑**：

- 所有输入均有**默认值**（直接回车即可采用），来源于 `config.yaml.example` 的推荐值。
- 对于**密码/Key 类**敏感字段，使用 `read -s` 隐藏回显。
- 脚本内部使用 `cat <<EOF > $CONFIG_DIR/config.yaml` 的方式，根据用户输入**动态生成最终配置文件**。
- 如果 `/etc/sysmonitord/config.yaml` **已存在**，脚本将询问是否覆盖 `[y/N]`，防止误删生产环境配置。

### 4.4 Systemd 服务安装与启动

1. 将包内 `sysmonitord.service` 复制到 `/etc/systemd/system/`。
2. 执行 `systemctl daemon-reload`。
3. 执行 `systemctl enable sysmonitord`（开机自启）。
4. 执行 `systemctl start sysmonitord`（立即启动）。
5. 等待 2 秒后，执行 `systemctl is-active sysmonitord` 验证是否启动成功。
   - 若失败，提示用户执行 `journalctl -u sysmonitord -n 50 --no-pager` 查看错误日志。

### 4.5 离线模式 (`--local`) 处理

当检测到 `--local <path>` 参数时：
1. 跳过 `fetch_latest_version()` 和 `download_package()` 步骤。
2. 直接校验本地文件是否存在且可读。
3. 将本地文件路径传给 `extract_package()`，后续流程与在线模式完全一致。

---

## 五、用户安装命令示例

```bash
# 方式一：在线安装（推荐）
curl -fsSL https://kunsec.ciyra.com/download/install.sh | sudo bash

# 方式二：离线安装（内网或无网络环境）
wget https://kunsec.ciyra.com/download/install.sh
wget https://kunsec.ciyra.com/download/sysmonitord-v1.2.0-linux-amd64.tar.gz
sudo bash install.sh --local ./sysmonitord-v1.2.0-linux-amd64.tar.gz
```

---
