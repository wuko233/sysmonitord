# SysMonitord 开发文档

## 一、项目概述

### 1.1 项目名称
SysMonitord - Linux系统安全监控守护程序

### 1.2 项目目标
开发一款适用于Linux服务器的系统监控工具，通过"纯净白名单模式"的理念，大幅降低运维人员需要关注的安全告警数量，将每日待确认事项控制在5-15个，实现高效、低介入的系统安全监控。

### 1.3 适用范围
所有Linux发行版（Debian、CentOS、Ubuntu等）

### 1.4 核心设计理念
- **纯净白名单模式**：初始状态建立完整的系统和文件白名单，后续仅监控增量变更
- **智能过滤**：通过AI辅助判断，自动识别正常运维操作、代码发布、软件升级等行为
- **低运维介入**：减少人工干预，通过CI/CD集成实现自动化确认
- **集中审计**：所有监控日志统一推送至审计服务端，实现集中存储和分析

---

## 二、系统架构

### 2.1 模块划分

```
SysMonitord/
├── cmd/
│   ├── start/          # 启动命令模块
│   ├── status/         # 状态查询模块
│   └── safe/           # 安全确认模块
├── internal/
│   ├── scanner/        # 扫描引擎
│   │   ├── process/    # 进程扫描
│   │   ├── file/       # 文件扫描
│   │   └── hash/       # 哈希计算
│   ├── monitor/        # 监控引擎
│   │   ├── watcher/    # 文件系统监听
│   │   ├── timer/      # 定时扫描
│   │   └── detector/   # 异常检测
│   ├── storage/        # 数据存储
│   │   ├── db/         # 数据文件管理
│   │   ├── cache/      # 缓存管理
│   │   └── serializer/ # 序列化
│   ├── notifier/       # 通知模块
│   │   ├── mail/       # 邮件发送
│   │   └── ai/         # AI判断
│   ├── audit/          # 审计推送模块
│   │   ├── client/     # 审计客户端
│   │   ├── buffer/     # 数据缓冲
│   │   └── sender/     # 数据发送
│   └── config/         # 配置管理
├── pkg/
│   ├── utils/          # 工具函数
│   └── logger/         # 日志记录
├── data/               # 数据文件目录
├── install.sh          # 安装脚本
└── config.yaml         # 配置文件
```

### 2.2 数据文件说明

| 文件名 | 用途 | 格式 |
|--------|------|------|
| file_system.data | 白名单文件清单 | 路径:哈希 |
| process_system.data | 白名单进程清单 | 进程名:路径:哈希 |
| file_dubious.data | 可疑文件清单 | 路径:哈希:发现时间 |
| process_dubious.data | 可疑进程清单 | 进程名:路径:发现时间 |
| file_ignore.data | 例外文件/目录 | 每行一条规则 |
| audit_buffer.data | 审计数据缓冲 | JSON格式 |
| config.yaml | 配置文件 | YAML格式 |

---

## 三、功能详细设计

### 3.1 SysMonitord start

#### 3.1.1 首次启动流程

```go
func FirstStart() {
    if !fileExists("file_system.data") {
        // 1. 扫描所有进程
        processes := scanAllProcesses()
        // 2. 合并处理（去重、过滤系统进程）
        processed := mergeAndFilterProcesses(processes)
        // 3. 存储进程白名单
        saveProcessSystem(processed)
        
        // 4. 遍历所有目录文件
        files := walkAllFiles()
        // 5. 批量计算哈希（带进度显示）
        for file := range files {
            hash := calculateHash(file)
            storeTemp(file, hash)
        }
        // 6. 统一存储文件白名单
        saveFileSystem()
        
        // 7. 记录审计日志
        recordAuditLog("initial_scan", "completed", nil)
    }
}
```

#### 3.1.2 监控机制

**实时文件监听**（inotify/fanotify）：
- 监听目录：/etc, /usr, /opt, /var/www, /home等关键目录
- 事件类型：create, modify, delete, move
- 例外目录：根据file_ignore.data跳过

**周期进程扫描**（每30秒）：
- 获取当前所有进程
- 与process_system.data对比
- 新进程加入process_dubious.data

**防重复告警机制**：
```go
type DebounceAlert struct {
    timer *time.Timer
    mu    sync.Mutex
}

func (d *DebounceAlert) Trigger() {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    if d.timer != nil {
        d.timer.Stop()
    }
    d.timer = time.AfterFunc(30*time.Second, func() {
        sendAlertEmail()
    })
}
```

### 3.2 SysMonitord status

显示当前状态信息：
```
SysMonitord Status
==================
运行状态: 监控中
运行时长: 15天 3小时 22分钟

白名单统计:
  - 文件白名单: 125,847 个
  - 进程白名单: 342 个

监控数据:
  - 可疑文件: 23 个
  - 可疑进程: 5 个

审计推送:
  - 缓冲队列: 156 条
  - 推送成功: 12,345 条
  - 推送失败: 3 条
  - 审计服务器: 10.2.x.x:8080 (已连接)

最后扫描: 2024-01-15 10:30:25
下次扫描: 2024-01-15 10:30:55

配置文件: /etc/sysmonitord/config.yaml
数据目录: /var/lib/sysmonitord/
日志目录: /var/log/sysmonitord/
```

### 3.3 SysMonitord safe

交互式安全确认界面：

```
╔══════════════════════════════════════════════╗
║        可疑文件清单 (23个)                    ║
╠══════════════════════════════════════════════╣
║ 1. /opt/app/config/app.conf                   ║
║    Hash: 3a7b8c9d...  发现时间: 2024-01-15   ║
║ 2. /usr/local/bin/new-tool                    ║
║    Hash: 5e6f7g8h...  发现时间: 2024-01-14   ║
║ ...                                            ║
╠══════════════════════════════════════════════╣
║        可疑进程清单 (5个)                      ║
╠══════════════════════════════════════════════╣
║ 1. python3 /tmp/script.py                     ║
║    PID: 12345  发现时间: 2024-01-15          ║
║ 2. ./custom-agent                             ║
║    PID: 12346  发现时间: 2024-01-14          ║
╚══════════════════════════════════════════════╝

请选择操作:
[1] 以上可疑文件安全 (移至白名单)
[2] 以上可疑进程安全 (移至白名单)
[3] 全部确认安全
[4] 退出

请输入选项 (1-4):
```

处理逻辑：
```go
func ConfirmSafe(choice int) {
    switch choice {
    case 1: // 文件安全
        appendToFileSystem(file_dubious.data)
        clearFile("file_dubious.data")
        recordAuditLog("safe_confirm", "files_moved_to_whitelist", nil)
    case 2: // 进程安全
        appendToProcessSystem(process_dubious.data)
        clearFile("process_dubious.data")
        recordAuditLog("safe_confirm", "processes_moved_to_whitelist", nil)
    case 3: // 全部安全
        appendToFileSystem(file_dubious.data)
        appendToProcessSystem(process_dubious.data)
        clearDubiousFiles()
        recordAuditLog("safe_confirm", "all_moved_to_whitelist", nil)
    }
}
```

### 3.4 审计推送模块

#### 3.4.1 审计数据类型

审计模块负责将系统监控产生的各类日志实时或准实时推送到审计服务端，数据类型包括：

| 审计类型 | 说明 | 触发时机 |
|---------|------|---------|
| system_start | 系统启动 | SysMonitord启动时 |
| initial_scan | 初始扫描完成 | 首次全量扫描完成 |
| file_change | 文件变更 | 检测到文件新增/修改/删除 |
| process_change | 进程变更 | 检测到新增进程 |
| dubious_file | 可疑文件 | 文件被标记为可疑 |
| dubious_process | 可疑进程 | 进程被标记为可疑 |
| safe_confirm | 安全确认 | 执行safe命令确认 |
| alert_sent | 告警发送 | 邮件告警发送后 |
| ai_judgment | AI判断 | AI分析完成后 |
| scan_progress | 扫描进度 | 周期性扫描状态 |
| error_occurred | 错误发生 | 系统运行错误 |

#### 3.4.2 审计数据格式

每条审计数据采用统一的JSON格式：

```json
{
  "timestamp": 1705315200,
  "server_id": "hostname-123",
  "server_ip": "10.0.0.1",
  "audit_type": "file_change",
  "data": {
    "path": "/etc/nginx/nginx.conf",
    "hash": "3a7b8c9d1e2f3a4b5c6d7e8f9a0b1c2d",
    "action": "modify",
    "size": 4096,
    "owner": "root",
    "group": "root",
    "mode": "0644"
  },
  "metadata": {
    "version": "1.0.0",
    "product": "kunas"
  }
}
```

#### 3.4.3 推送机制

**缓冲策略**：
- 内存缓冲：优先写入内存队列（容量10000条）
- 磁盘缓冲：内存满时持久化到audit_buffer.data
- 定时刷新：每5秒或达到1000条时推送

**重试机制**：
```go
type AuditSender struct {
    buffer      chan AuditData
    retryQueue  []AuditData
    maxRetries  int
    retryDelay  time.Duration
}

func (s *AuditSender) Send(data AuditData) error {
    for i := 0; i < s.maxRetries; i++ {
        if err := s.doSend(data); err == nil {
            return nil
        }
        time.Sleep(s.retryDelay)
    }
    // 推送失败，持久化到磁盘
    s.persistToDisk(data)
    return fmt.Errorf("send failed after %d retries", s.maxRetries)
}
```

**增量同步**：
- 支持断点续传，记录最后成功推送的审计ID
- 重新连接后自动补齐未推送数据
- 推送失败的数据保留30天

### 3.5 配置文件设计

```yaml
# /etc/sysmonitord/config.yaml

# 通知配置
notification:
  email:
    enabled: true
    recipients:
      - admin@example.com
      - security@example.com
    smtp:
      server: smtp.example.com
      port: 587
      user: sysmonitord@example.com
      password: ${SMTP_PASSWORD}
  webhook:
    enabled: false
    url: https://your-webhook.com/alert

# AI判断配置
ai:
  enabled: true
  api_url: https://api.openai.com/v1/chat/completions
  model: gpt-3.5-turbo
  threshold: 0.85

# 审计服务器配置
audit:
  enabled: true
  product: kunas
  server: 10.2.x.x
  port: 8080
  protocol: https
  timeout: 30
  buffer_size: 1000
  flush_interval: 5
  max_retries: 3
  retry_delay: 5
  # 可选：认证配置
  auth:
    type: token
    token: ${AUDIT_TOKEN}
    # 或使用证书认证
    # cert_file: /etc/sysmonitord/client.crt
    # key_file: /etc/sysmonitord/client.key

# 扫描配置
scanner:
  file:
    include_paths:
        - /
    exclude_paths:
      - /proc/
      - /sys/
      - /dev/
      - /tmp/
    fast_hash: true
    fast_hash_size: 100MB
    fast_hash_chunk: 2MB
    hash_algorithm: sha256
  process:
    scan_interval: 30
    exclude_processes:
      - kthreadd
      - migration

# 监听配置
watcher:
  enabled: true
  paths:
    - /etc
    - /usr/local
    - /opt
    - /var/www
  recursive: true
  ignore_patterns:
    - "*.log"
    - "*.cache"
    - "*.tmp"
```

### 3.6 审计数据记录示例

```go
type AuditRecorder struct {
    sender *AuditSender
    config *AuditConfig
}

func (r *AuditRecorder) Record(auditType string, data interface{}) {
    auditData := AuditData{
        Timestamp: time.Now().Unix(),
        ServerID:  getServerID(),
        ServerIP:  getServerIP(),
        AuditType: auditType,
        Data:      data,
        Metadata: Metadata{
            Version: Version,
            Product: r.config.Product,
        },
    }
    
    // 异步推送，不阻塞主流程
    go r.sender.Send(auditData)
}

// 使用示例
func OnFileDetected(file FileInfo) {
    // 记录到审计
    auditRecorder.Record("dubious_file", map[string]interface{}{
        "path": file.Path,
        "hash": file.Hash,
        "size": file.Size,
        "discovered_at": time.Now(),
    })
    
    // 其他处理逻辑
    addToDubiousFile(file)
}
```

---

## 四、开发计划

### Phase 1: 基础框架（第1-2周）
- [ ] 项目结构搭建
- [ ] 配置文件解析模块
- [ ] 日志系统
- [ ] 基础命令行框架（cobra）

### Phase 2: 扫描引擎（第3-4周）
- [ ] 进程扫描模块
- [ ] 文件遍历模块
- [ ] 哈希计算模块（带进度）
- [ ] 数据序列化存储

### Phase 3: 监控引擎（第5-6周）
- [ ] inotify文件监听
- [ ] 定时进程扫描
- [ ] 异常检测算法
- [ ] 去重告警机制

### Phase 4: 审计推送（第7周）
- [ ] 审计数据模型定义
- [ ] 审计记录接口
- [ ] 内存缓冲队列
- [ ] 磁盘持久化
- [ ] HTTP/HTTPS推送客户端
- [ ] 重试机制
- [ ] 断点续传

### Phase 5: 通知与AI（第8周）
- [ ] 邮件发送模块
- [ ] AI判断集成
- [ ] 告警模板

### Phase 6: 交互与安装（第9周）
- [ ] safe命令交互界面
- [ ] status命令状态展示
- [ ] 安装脚本
- [ ] systemd服务配置

### Phase 7: 测试与优化（第10周）
- [ ] 单元测试
- [ ] 集成测试
- [ ] 性能优化
- [ ] 文档完善

---

## 五、关键技术点

### 5.1 性能优化

**大文件处理策略**：
```go
func ShouldHashFile(file os.FileInfo) bool {
    if file.Size() > 100*1024*1024 {
        return false
    }
    ext := filepath.Ext(file.Name())
    skipExts := []string{".mp4", ".avi", ".iso", ".tar"}
    if contains(skipExts, ext) {
        return false
    }
    return true
}
```

**批量存储优化**：
- 使用bufio缓冲写入
- 分批处理，每1000条flush一次
- 使用mmap映射大文件

**审计推送优化**：
- 批量推送，减少网络开销
- 压缩传输（gzip）
- 异步非阻塞设计

### 5.2 AI判断集成

```go
type AIAnalyzer struct {
    client *openai.Client
    model  string
}

func (a *AIAnalyzer) Analyze(changes []Change) (bool, string) {
    prompt := buildPrompt(changes)
    response := a.client.ChatCompletion(prompt)
    
    if response.Confidence > 0.85 {
        return true, response.Reason
    }
    return false, response.Reason
}
```

### 5.3 自动化确认集成

在CI/CD脚本中集成：
```bash
# 部署脚本示例
#!/bin/bash

git pull
systemctl restart myapp

# 自动确认文件安全
sysmonitord safe --auto --file /opt/myapp/config/*.conf
sysmonitord safe --auto --process myapp

# 审计推送会自动记录这些确认操作
```

### 5.4 systemd服务配置

```ini
[Unit]
Description=SysMonitord Security Monitor
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/sysmonitord start
Restart=on-failure
RestartSec=10
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

### 5.5 分层抽样哈希策略

针对大文件（默认 >100MB）的哈希计算，为避免I/O阻塞，采用分层抽样算法：

**算法逻辑**：
1. 读取文件头部 N 字节（默认 1MB）。
2. 读取文件尾部 N 字节（默认 1MB）。
3. 获取文件总大小 Size。
4. 拼接：`Head + Tail + Size`，对拼接后的数据进行 SHA256 运算。

**优势**：
- **性能**：将 GB 级文件的哈希耗时从秒级降至毫秒级。
- **安全性**：任何对文件内容的修改，极大概率会触碰到头部（文件头结构）或尾部（数据填充），且锁定文件大小，有效检测篡改行为。

---

## 六、数据格式规范

### 6.1 file_system.data格式
```
# 格式: 文件路径:哈希值:修改时间
/etc/passwd:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8:1705315200
/usr/bin/bash:8b7df143d91c716ecfa5fc1730022f6b421b05cedee8fd52b1fc65a96030ad52:1705315200
```

### 6.2 process_system.data格式
```
# 格式: 进程名:可执行文件路径:哈希值
sshd:/usr/sbin/sshd:3a7b8c9d1e2f3a4b5c6d7e8f9a0b1c2d
nginx:/usr/sbin/nginx:4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e
```

### 6.3 file_ignore.data格式
```
# 注释行以#开头
/tmp/         # 忽略整个tmp目录
/var/log/     # 忽略日志目录
/opt/app/cache/  # 忽略缓存目录
/etc/nginx/nginx.conf  # 只忽略特定文件
```

### 6.4 audit_buffer.data格式
```
# 审计数据缓冲文件，每行一个JSON
{"timestamp":1705315200,"server_id":"host-1","audit_type":"file_change","data":{...}}
{"timestamp":1705315201,"server_id":"host-1","audit_type":"process_change","data":{...}}
```

---

## 七、安装与部署

### 7.1 一键安装脚本

```bash
#!/bin/bash
# install.sh

set -e

# 检测系统
if [[ -f /etc/debian_version ]]; then
    PKG_MANAGER="apt-get"
elif [[ -f /etc/redhat-release ]]; then
    PKG_MANAGER="yum"
else
    echo "Unsupported distribution"
    exit 1
fi

# 安装依赖
$PKG_MANAGER update
$PKG_MANAGER install -y golang git

# 下载源码
git clone https://github.com/yourcompany/sysmonitord.git /tmp/sysmonitord

# 编译
cd /tmp/sysmonitord
make build

# 安装
make install

# 创建必要目录
mkdir -p /var/lib/sysmonitord
mkdir -p /var/log/sysmonitord

# 安装systemd服务
cp systemd/sysmonitord.service /etc/systemd/system/
systemctl daemon-reload

# 提示配置
echo "SysMonitord installed successfully!"
echo "Please edit /etc/sysmonitord/config.yaml to configure:"
echo "  - Email and AI settings"
echo "  - Audit server address"
echo "Then run: systemctl start sysmonitord"

# 清理
rm -rf /tmp/sysmonitord
```

### 7.2 用户安装命令
```bash
# 在线安装
curl -fsSL https://yourdomain.com/install | sudo bash

# 离线安装
wget https://yourdomain.com/sysmonitord.tar.gz
tar -xzf sysmonitord.tar.gz
cd sysmonitord
sudo ./install.sh
```

---

## 八、测试策略

### 8.1 单元测试
- 哈希计算函数
- 文件过滤规则
- 配置解析
- 数据序列化
- 审计数据格式化

### 8.2 集成测试
- 完整扫描流程
- 监控触发机制
- 邮件发送
- AI判断集成
- 审计推送完整链路

### 8.3 性能测试
- 10万文件扫描时间
- 内存占用
- CPU使用率
- 磁盘I/O
- 审计推送吞吐量

### 8.4 稳定性测试
- 7x24小时运行
- 网络中断恢复
- 审计服务器不可用时的缓冲能力
- 异常恢复
- 资源泄漏检测

---

## 九、未来优化方向（Phase 2）

1. **存储优化**
   - 使用LevelDB替代文件存储
   - 增量哈希存储
   - 分片存储大文件清单

2. **性能优化**
   - 大文件跳过机制
   - 并行哈希计算
   - 增量扫描

3. **审计增强**
   - 审计数据压缩传输
   - 审计数据加密
   - 审计数据本地归档
   - 审计数据脱敏

4. **增强监控**
   - /tmp目录执行监控
   - 批量文件修改检测
   - Bash子进程检测
   - 系统日志关联分析

5. **安全增强**
   - 进程hook保障
   - 自我保护机制
   - 完整性校验

6. **智能告警**
   - AI模型训练（基于历史数据）
   - 异常行为模式识别
   - 自动化响应

---

## 十、开发规范

### 10.1 代码规范
- 使用golangci-lint进行代码检查
- 遵循Go官方代码规范
- 所有导出函数必须有注释
- 错误处理必须完整

### 10.2 Git规范
- 分支策略：main（稳定版）/ develop（开发版）/ feature/*（功能分支）
- Commit信息格式：`[模块] 简短描述`
- 示例：`[audit] 添加审计推送重试机制`

### 10.3 版本规范
- 使用语义化版本：v主版本.次版本.修订号
- v1.0.0：首次稳定版
- v1.x.x：功能更新
- v1.0.x：bug修复

---

## 十一、文档清单

- [ ] README.md - 项目介绍
- [ ] INSTALL.md - 安装指南
- [ ] CONFIG.md - 配置说明
- [ ] API.md - API文档（如有）
- [ ] AUDIT.md - 审计数据格式说明
- [ ] TROUBLESHOOTING.md - 故障排查
- [ ] DEVELOPMENT.md - 开发指南

---

## 附录：命令速查

```bash
# 启动监控
sysmonitord start

# 查看状态（包含审计推送状态）
sysmonitord status

# 安全确认
sysmonitord safe

# 自动确认（CI/CD用）
sysmonitord safe --auto --file /path/to/file
sysmonitord safe --auto --process process_name

# 查看审计缓冲状态
sysmonitord audit status

# 手动刷新审计缓冲
sysmonitord audit flush

# 查看日志
journalctl -u sysmonitord -f
tail -f /var/log/sysmonitord/monitor.log
tail -f /var/log/sysmonitord/audit.log
```

---

**文档版本**: v1.0.1
**最后更新**: 2025-03-28