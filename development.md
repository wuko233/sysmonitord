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
    case 2: // 进程安全
        appendToProcessSystem(process_dubious.data)
        clearFile("process_dubious.data")
    case 3: // 全部安全
        appendToFileSystem(file_dubious.data)
        appendToProcessSystem(process_dubious.data)
        clearDubiousFiles()
    }
}
```

### 3.4 配置文件设计

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
      password: ${SMTP_PASSWORD}  # 从环境变量读取
  webhook:
    enabled: false
    url: https://your-webhook.com/alert

# AI判断配置
ai:
  enabled: true
  api_url: https://api.openai.com/v1/chat/completions
  model: gpt-3.5-turbo
  threshold: 0.85  # 置信度阈值

# 审计服务器
audit:
  enabled: true
  product: kunas
  server: 10.2.x.x
  port: 8080

# 扫描配置
scanner:
  file:
    exclude_paths:
      - /proc/
      - /sys/
      - /dev/
      - /tmp/     # 临时目录暂不监控（待优化）
    max_file_size: 100MB  # 超过此大小的文件不计算hash
    hash_algorithm: sha256
  process:
    scan_interval: 30  # 秒
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

### Phase 4: 通知与AI（第7-8周）
- [ ] 邮件发送模块
- [ ] AI判断集成（OpenAI API）
- [ ] 审计服务器上报
- [ ] 告警模板

### Phase 5: 交互与安装（第9周）
- [ ] safe命令交互界面
- [ ] status命令状态展示
- [ ] 安装脚本
- [ ] systemd服务配置

### Phase 6: 测试与优化（第10周）
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
    // 跳过超大文件
    if file.Size() > 100*1024*1024 {
        return false
    }
    // 跳过特定类型
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

### 5.2 AI判断集成

```go
type AIAnalyzer struct {
    client *openai.Client
    model  string
}

func (a *AIAnalyzer) Analyze(changes []Change) (bool, string) {
    prompt := buildPrompt(changes)
    response := a.client.ChatCompletion(prompt)
    
    // 解析AI响应
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

# 部署新代码
git pull
systemctl restart myapp

# 自动确认文件安全
sysmonitord safe --auto --file /opt/myapp/config/*.conf
sysmonitord safe --auto --process myapp
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
echo "Please edit /etc/sysmonitord/config.yaml to configure email and AI settings"
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

### 8.2 集成测试
- 完整扫描流程
- 监控触发机制
- 邮件发送
- AI判断集成

### 8.3 性能测试
- 10万文件扫描时间
- 内存占用
- CPU使用率
- 磁盘I/O

### 8.4 稳定性测试
- 7x24小时运行
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

3. **增强监控**
   - /tmp目录执行监控
   - 批量文件修改检测
   - Bash子进程检测
   - 系统日志关联分析

4. **安全增强**
   - 进程hook保障
   - 自我保护机制
   - 完整性校验

5. **智能告警**
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
- 示例：`[scanner] 添加文件哈希计算并发支持`

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
- [ ] TROUBLESHOOTING.md - 故障排查
- [ ] DEVELOPMENT.md - 开发指南

---

## 附录：命令速查

```bash
# 启动监控
sysmonitord start

# 查看状态
sysmonitord status

# 安全确认
sysmonitord safe

# 自动确认（CI/CD用）
sysmonitord safe --auto --file /path/to/file
sysmonitord safe --auto --process process_name

# 查看日志
journalctl -u sysmonitord -f
tail -f /var/log/sysmonitord/monitor.log

# 重启服务
systemctl restart sysmonitord
```

---

**文档版本**: v1.0  
**最后更新**: 2026-03-28  