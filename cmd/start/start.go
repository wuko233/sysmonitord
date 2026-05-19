package start

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"sysmonitord/internal/config"
	"sysmonitord/internal/event"
	"sysmonitord/internal/monitor/detector"
	"sysmonitord/internal/monitor/timer"
	"sysmonitord/internal/monitor/watcher"
	"sysmonitord/internal/notifier"
	"sysmonitord/internal/scanner/file"
	"sysmonitord/internal/scanner/process"
	"sysmonitord/internal/script"
	"sysmonitord/internal/storage"
	"sysmonitord/pkg/logger"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func NewStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "启动系统监控守护服务",
		Long:  "sysmonitord start 命令用于启动系统监控守护服务，首次启动会进行全量扫描建立白名单。",
		Run: func(cmd *cobra.Command, args []string) {
			logger.Log.Info("正在启动系统监控守护服务...")

			cfg, ok := cmd.Context().Value("config").(*config.Config)
			if !ok {
				logger.Log.Error("无法获取配置")
				os.Exit(1)
			}

			logger.Log.Info("配置文件加载成功",
				zap.String("审计服务器地址", fmt.Sprintf("%s:%d", cfg.Audit.Server, cfg.Audit.Port)),
			)

			storageCfg := &storage.Storage{
				DataDir:           cfg.Storage.DataDir,
				ProcessSystemFile: cfg.Storage.ProcessSystemFile,
				FileSystemFile:    cfg.Storage.FileSystemFile,
			}

			if err := storage.InitDataDir(storageCfg.DataDir); err != nil {
				logger.Log.Error("初始化数据目录失败", zap.Error(err))
				os.Exit(1)
			}

			// ====== 首次启动：进程白名单初始化 ======
			processWhitelistExists := storage.DataFileExists(
				storageCfg.DataDir,
				storageCfg.ProcessSystemFile,
			)

			if !processWhitelistExists {
				logger.Log.Info("未发现进程白名单，开始首次进程扫描")

				startTime := time.Now()

				procs, err := process.ScanAllProcesses(cfg)
				if err != nil {
					logger.Log.Error("扫描进程失败", zap.Error(err))
					os.Exit(1)
				}

				if err := storage.SaveProcessSystem(
					procs,
					storageCfg.DataDir,
					storageCfg.ProcessSystemFile,
				); err != nil {
					logger.Log.Error("保存进程白名单失败", zap.Error(err))
					os.Exit(1)
				}

				logger.Log.Info("首次进程白名单建立完成",
					zap.Int("进程数量", len(procs)),
					zap.Duration("扫描耗时", time.Since(startTime)),
				)
			} else {
				logger.Log.Info("检测到已有进程白名单，跳过首次进程扫描")
			}

			// ====== 首次启动：文件白名单初始化 ======
			fileWhitelistExists := storage.DataFileExists(
				storageCfg.DataDir,
				storageCfg.FileSystemFile,
			)

			if !fileWhitelistExists {
				logger.Log.Info("未发现文件白名单，开始首次文件系统扫描")

				startTime := time.Now()

				fileScanner := file.NewScanner(cfg)

				files, err := fileScanner.Scan()
				if err != nil {
					logger.Log.Error("扫描文件系统失败", zap.Error(err))
					os.Exit(1)
				}

				if err := storage.SaveFileSystem(
					files,
					storageCfg.DataDir,
					storageCfg.FileSystemFile,
				); err != nil {
					logger.Log.Error("保存文件系统白名单失败", zap.Error(err))
					os.Exit(1)
				}

				logger.Log.Info("首次文件系统白名单建立完成",
					zap.Int("文件数量", len(files)),
					zap.Duration("扫描耗时", time.Since(startTime)),
				)
			} else {
				logger.Log.Info("检测到已有文件白名单，跳过首次文件系统扫描")
			}

			// ====== 启动文件监听 ======
			logger.Log.Info("正在启动文件监听...")

			fileMon, err := watcher.NewWatcher(cfg)
			if err != nil {
				logger.Log.Error("启动文件监听失败", zap.Error(err))
				os.Exit(1)
			}

			fileMon.Start()

			// ====== 初始化文件检测器 ======
			fileDetector, err := detector.NewFileDetector(cfg)
			if err != nil {
				logger.Log.Error("初始化文件检测器失败", zap.Error(err))
				os.Exit(1)
			}

			fileEventChan := fileDetector.Events()

			// ====== 启动进程检测定时任务 ======
			procDetector, err := detector.NewProcessDetector(cfg)
			if err != nil {
				logger.Log.Error("初始化进程检测器失败", zap.Error(err))
				os.Exit(1)
			}

			procEventChan := procDetector.Event()
			procScheduler := timer.NewScheduler(time.Duration(cfg.Scanner.Process.Interval)*time.Second, procDetector)
			procScheduler.Start()

			// ====== 启动告警管理器 ======
			alerter := notifier.NewAlerter(cfg.Notification)
			alerter.Start()

			// ====== 初始化脚本管理器 ======
			scriptManager := script.NewManager(cfg.Script)
			if scriptManager.Enabled() {
				logger.Log.Debug("脚本系统已启用",
					zap.String("script_dir", cfg.Script.Dir),
					zap.Int("event_count", len(cfg.Script.Events)),
				)
			}

			logger.Log.Info("系统监控守护服务已启动，正在监控系统变化...")

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

			handleEvent := func(e event.Event) {
				allowDefault := true // 是否默认继续执行

				if scriptManager.Enabled() {
					scriptEvent := script.FromEvent(&e)
					executions := scriptManager.ExecuteEvent(scriptEvent)
					allowDefault = handleScriptExecutions(executions)
				}

				if !allowDefault {
					logger.Log.Debug("脚本已阻止默认事件处理流程",
						zap.String("event_type", string(e.Type)),
						zap.String("source", e.Source),
						zap.String("path", e.Path),
						zap.String("name", e.Name),
					)
					return
				}

				switch e.Type {
				case event.TypeFileChange:
					handleFileChange(e, fileDetector)
				case event.TypeDubiousFile:
					handleDubiousFile(e, alerter)
				case event.TypeDubiousProcess:
					handleDubiousProcess(e, alerter, procDetector)
				case event.TypeError:
					handleError(e)
				case event.TypeSystemStart:
					handleSystemStart(e)
				case event.TypeSystemStop:
					handleSystemStop(e)
				default:
					logger.Log.Warn("未知事件类型", zap.String("type", string(e.Type)))
				}
			}

			eventCh := make(chan event.Event, 100)

			go func() {
				for ev := range eventCh {
					handleEvent(ev)
				}
			}()

			eventCh <- event.Event{
				Time:   time.Now(),
				Type:   event.TypeSystemStart,
				Source: "System",
				Reason: "系统启动",
				Detail: "Sysmonitord 监控守护服务已启动",
			}

			for {
				select {

				case ev := <-fileMon.Events():
					eventCh <- event.Event{
						Time:   time.Now(),
						Type:   event.TypeFileChange,
						Source: "FileWatcher",
						Path:   ev.Path,
						Detail: fmt.Sprintf("文件事件: %s", ev.Op.String()),
						Data:   ev.FileInfo,
					}

				case fileEvent := <-fileEventChan:
					eventCh <- event.Event{
						Time:   time.Now(),
						Type:   event.TypeDubiousFile,
						Source: "FileDetector",
						Path:   fileEvent.Path,
						Reason: "可疑文件事件",
						Detail: fmt.Sprintf("hash=%s, discovered_at=%s", fileEvent.Hash, fileEvent.DiscoveredAt),
						Data:   fileEvent,
					}

				case procEvents := <-procEventChan:
					eventCh <- event.Event{
						Time:   time.Now(),
						Type:   event.TypeDubiousProcess,
						Source: "ProcessDetector",
						PID:    procEvents.PID,
						Name:   procEvents.Name,
						Path:   procEvents.Path,
						Reason: "可疑进程事件",
						Detail: fmt.Sprintf("pid=%d, name=%s, path=%s", procEvents.PID, procEvents.Name, procEvents.Path),
						Data:   procEvents,
					}

				case err := <-fileMon.Errors():
					eventCh <- event.Event{
						Time:   time.Now(),
						Type:   event.TypeError,
						Source: "FileWatcher",
						Reason: "文件监听错误",
						Err:    err,
					}

				case <-quit:
					eventCh <- event.Event{
						Time:   time.Now(),
						Type:   event.TypeSystemStop,
						Source: "System",
						Reason: "系统停止",
						Detail: "收到系统停止信号，正在关闭服务...",
					}

					logger.Log.Info("正在停止系统监控组件...")
					procScheduler.Stop()
					fileMon.Stop()
					logger.Log.Info("系统监控守护服务已停止")
					return
				}
			}

		},
	}

	return cmd
}
