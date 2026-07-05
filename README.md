# SysMonitord

SysMonitord 是一个面向 Linux 服务器的系统安全监控守护程序。它通过首次扫描建立文件和进程白名单，后续持续监控文件变更和新增进程，并将异常对象记录为可疑项，方便管理员集中确认和处置。

## 功能特性

- 白名单基线：首次启动自动扫描文件系统和当前进程，生成可信基线。
- 文件监控：监听配置范围内的文件新增、修改、删除和移动事件。
- 进程监控：按固定周期扫描系统进程，发现不在白名单中的进程。
- 可疑项管理：将可疑文件、可疑进程保存到数据目录，支持交互式确认安全并移入白名单。
- 邮件告警：可配置 SMTP，在发现可疑对象后发送通知。
- 事件脚本：支持按事件类型执行 JavaScript 脚本，便于自定义联动处理。
- AI 安全分析：可读取指定系统/应用配置文件，调用兼容 OpenAI Chat Completions 的接口生成安全建议报告。
- systemd 部署：提供安装、卸载脚本和 systemd service 文件。

## 适用场景

- Linux 服务器文件完整性监控。
- 服务器进程白名单巡检。
- 运维变更后的安全确认。
- 中小规模服务器的轻量级主机安全监控。
- 对系统配置进行 AI 辅助安全审查。

## 项目结构

```text
.
├── cmd/                    # Cobra 命令入口
│   ├── ai/                 # AI 安全分析命令
│   ├── safe/               # 可疑项确认命令
│   ├── start/              # 守护服务启动命令
│   ├── status/             # 状态查看命令
│   └── version/            # 版本信息命令
├── internal/
│   ├── ai/                 # AI Prompt 生成、客户端和报告保存
│   ├── config/             # 配置加载和解析
│   ├── event/              # 事件模型
│   ├── monitor/            # 文件监听、进程定时扫描、异常检测
│   ├── notifier/           # 告警管理和邮件发送
│   ├── scanner/            # 文件、进程和哈希扫描
│   ├── script/             # JavaScript 事件脚本引擎
│   └── storage/            # 白名单和可疑项数据存储
├── pkg/logger/             # 日志封装
├── scripts/                # systemd service 和示例脚本
├── data/                   # 本地开发数据样例
├── config.yaml.example     # 配置文件示例
├── install.sh              # 安装脚本
├── uninstall.sh            # 卸载脚本
└── Makefile                # 构建和打包命令
```

## 快速开始

### 环境要求

- Linux 系统。
- root 权限，文件系统全量扫描、systemd 部署和读取系统配置通常需要 root。
- Go 1.26.1 或兼容版本，用于源码构建。
- systemd，用于以服务方式运行。

### 在线安装

```bash
curl -fsSL https://kunsec.ciyra.com/download/install.sh | sudo bash
```

安装脚本会自动检测架构、下载发布包、安装二进制文件、创建配置/数据/日志目录，并进入交互式配置向导。

默认安装路径：

| 类型 | 路径 |
| --- | --- |
| 可执行文件 | `/usr/local/bin/sysmonitord` |
| 配置目录 | `/etc/sysmonitord` |
| 配置文件 | `/etc/sysmonitord/config.yaml` |
| 数据目录 | `/var/lib/sysmonitord` |
| 日志目录 | `/var/log/sysmonitord` |
| systemd 服务 | `/etc/systemd/system/sysmonitord.service` |

### 离线安装

先准备发布包，例如 `sysmonitord-V0.1.0-linux-amd64.tar.gz`，然后执行：

```bash
sudo bash install.sh --local ./sysmonitord-V0.1.0-linux-amd64.tar.gz
```

### 源码构建

构建 amd64 版本：

```bash
make build
```

构建 arm64 版本：

```bash
make build-arm64
```

构建产物会输出到 `dist/`，发布包会输出到 `release/`。

## 配置说明

SysMonitord 默认按以下顺序查找配置文件：

1. 通过 `--config` 或 `-c` 指定的路径。
2. 当前目录下的 `./config.yaml`。
3. 系统配置路径 `/etc/sysmonitord/config.yaml`。

可以从示例文件复制一份配置：

```bash
cp config.yaml.example config.yaml
```

常用配置项：

| 配置段 | 说明 |
| --- | --- |
| `log` | 日志级别，例如 `info`、`debug`。 |
| `audit` | 审计服务地址、端口和缓冲区大小。 |
| `scanner.hash` | 文件哈希算法，示例配置使用 `xxhash64`。 |
| `scanner.file.include_paths` | 文件扫描和监听的包含路径。 |
| `scanner.file.exclude_paths` | 扫描和监听时排除的路径或通配规则。 |
| `scanner.process.interval` | 进程扫描间隔，单位为秒。 |
| `storage` | 白名单、可疑项等数据文件的保存位置。 |
| `notification.email` | 邮件告警收件人和 SMTP 配置。 |
| `script` | 事件脚本开关、脚本目录、超时时间和事件映射。 |
| `ai` | AI 安全分析接口、模型、报告目录和读取路径。 |

敏感配置建议使用环境变量占位，例如：

```yaml
notification:
  email:
    smtp:
      password: "${SYSMONITORD_SMTP_PASSWORD}"

ai:
  api_key: "${SYSMONITORD_AI_API_KEY}"
```

## 命令使用

查看帮助：

```bash
sysmonitord --help
```

使用指定配置文件：

```bash
sysmonitord -c /etc/sysmonitord/config.yaml <command>
```

### 启动监控

```bash
sudo sysmonitord start
```

首次启动时，如果数据目录中不存在白名单文件，程序会执行初始扫描并生成：

- 文件白名单：`file_system.data`
- 进程白名单：`process_system.data`

之后程序会启动文件监听、进程定时扫描、告警管理器和事件脚本系统。

### 查看状态

```bash
sysmonitord status
```

该命令会显示服务运行时间、数据目录、文件白名单数量、进程白名单数量、可疑文件数量和可疑进程数量。

### 确认可疑项安全

```bash
sudo sysmonitord safe
```

该命令会进入交互式界面，可以逐个或批量确认可疑文件/可疑进程。确认后的对象会被移入对应白名单，并从可疑列表中删除。

### AI 安全分析

```bash
sudo sysmonitord ai
```

AI 分析会读取 `ai.include_paths` 中配置的系统和应用配置文件，并发送到配置的 AI 接口。执行前程序会要求输入 `yes` 确认风险。

注意事项：

- 被读取的配置文件可能包含密码、Token、密钥或数据库连接串。
- 文件路径和文件内容会作为 Prompt 明文发送给 AI 服务。
- SysMonitord 不保存原始配置内容，只保存 AI 返回的报告。
- AI 报告仅供参考，修改系统配置前请先备份并由管理员复核。

### 查看版本

```bash
sysmonitord version
```

## systemd 管理

安装后可以使用 systemd 管理服务：

```bash
sudo systemctl start sysmonitord
sudo systemctl status sysmonitord
sudo systemctl enable sysmonitord
```

查看日志：

```bash
journalctl -u sysmonitord -f
```

重启服务：

```bash
sudo systemctl restart sysmonitord
```

停止服务：

```bash
sudo systemctl stop sysmonitord
```

## 数据文件

默认数据目录为 `/var/lib/sysmonitord`，主要数据文件包括：

| 文件 | 说明 |
| --- | --- |
| `file_system.data` | 文件白名单数据。 |
| `process_system.data` | 进程白名单数据。 |
| `dubious_files.data` | 可疑文件列表。 |
| `dubious_processes.data` | 可疑进程列表。 |
| `reports/` | AI 安全分析报告目录。 |

请谨慎删除白名单数据。删除后再次启动会重新进行首次扫描，当前系统状态会被重新视为基线。

## 事件脚本

配置示例：

```yaml
script:
  enabled: true
  dir: "/etc/sysmonitord/scripts"
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
```

事件脚本适合用于自定义通知、审计转发、阻断默认处理流程或与现有运维平台集成。脚本运行超时时间由 `script.timeout_ms` 控制。

## 卸载

```bash
sudo bash uninstall.sh
```

卸载脚本会停止并禁用 systemd 服务，删除二进制文件和 service 文件。执行过程中会询问是否同时删除配置、数据和日志目录。

## 开发

常用命令：

```bash
go test ./...
make build
make clean
```

带版本号构建：

```bash
make build VERSION=V0.1.0
```

## 安全建议

- 首次建立白名单前，请确认服务器处于可信、干净状态。
- 生产环境中建议先缩小 `scanner.file.include_paths` 范围，避免首次全盘扫描耗时过长。
- 将 `/proc`、`/sys`、`/dev`、缓存目录、日志目录、构建目录等高频变化路径加入 `exclude_paths`。
- SMTP 密码和 AI API Key 不建议明文写入版本库。
- 执行 `safe` 确认可疑项前，应先确认变更来源是否可信。
- 使用 AI 分析前，应确认配置文件内容允许发送到目标 AI 服务。

## License

本项目使用 `GPL-v2.0` 许可证。
