package start

import (
	"fmt"
	"os"
	"sysmonitord/internal/config"
	"sysmonitord/internal/monitor/watcher"
	"sysmonitord/internal/scanner/file"
	"sysmonitord/internal/scanner/process"
	"sysmonitord/internal/storage"
	"sysmonitord/pkg/logger"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动系统监控守护服务",
	Long:  "sysmonitord start 命令用于启动系统监控守护服务，首次启动会进行全量扫描建立白名单。",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Log.Info("正在启动系统监控守护服务...")

		cfg, err := config.LoadConfig("./config.yaml")
		if err != nil {
			logger.Log.Error("加载配置文件失败", zap.Error(err))
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

		// ====== 进程扫描和存储 ======

		startTime := time.Now()
		procs, err := process.ScanAllProcesses(cfg)

		logger.Log.Info("进程扫描完成",
			zap.Int("进程数量", len(procs)),
			zap.Duration("扫描耗时", time.Since(startTime)),
		)

		if err != nil {
			logger.Log.Error("扫描进程失败", zap.Error(err))
			os.Exit(1)
		} else {
			if err := storage.SaveProcessSystem(procs, storageCfg.DataDir, storageCfg.ProcessSystemFile); err != nil {
				logger.Log.Error("保存进程白名单失败", zap.Error(err))
			}
		}

		// ====== 文件扫描和存储 ======
		logger.Log.Info("正在扫描文件系统...")

		startTime = time.Now()
		fileScanner := file.NewScanner(cfg)

		files, err := fileScanner.Scan()
		if err != nil {
			logger.Log.Error("扫描文件系统失败", zap.Error(err))
			os.Exit(1)
		} else {
			if err := storage.SaveFileSystem(files, storageCfg.DataDir, storageCfg.FileSystemFile); err != nil {
				logger.Log.Error("保存文件系统白名单失败", zap.Error(err))
				os.Exit(1)
			}
		}

		duration := time.Since(startTime)
		logger.Log.Info("文件系统扫描完成",
			zap.Int("文件数量", len(files)),
			zap.Duration("扫描耗时", duration),
		)

		// ====== 启动文件监听 ======
		logger.Log.Info("正在启动文件监听...")

		mon, err := watcher.NewWatcher(cfg.Scanner.File.IncludePaths, cfg.Scanner.File.ExcludePaths)
		if err != nil {
			logger.Log.Error("启动文件监听失败", zap.Error(err))
			os.Exit(1)
		}

		mon.Start()

		logger.Log.Info("系统监控守护服务已启动，正在监控文件系统变化...")

		for {
			select {

			case event := <-mon.Events():
				logger.Log.Info("文件系统事件",
					zap.String("path", event.Path),
					zap.String("op", event.Op.String()),
				)

				if event.FileInfo != nil {
					// Todo: 处理文件系统事件，例如更新白名单、触发告警等
					logger.Log.Debug("文件详情", zap.Int64("size", event.FileInfo.Size()))
				}

			case err := <-mon.Errors():
				logger.Log.Error("文件监听错误", zap.Error(err))

				// case <-quit:
				// 	logger.Log.Info("正在停止系统监控守护服务...")
				// 	mon.Stop()
				// 	logger.Log.Info("系统监控守护服务已停止")
				// 	return

			}
		}

	},
}
