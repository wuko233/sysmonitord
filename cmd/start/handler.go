package start

import (
	"fmt"
	"sysmonitord/internal/event"
	"sysmonitord/internal/monitor/detector"
	"sysmonitord/internal/notifier"
	"sysmonitord/internal/script"
	"sysmonitord/internal/storage"
	"sysmonitord/pkg/logger"

	"go.uber.org/zap"
)

func handleFileChange(e event.Event, fileDetector *detector.FileDetector) {
	logger.Log.Debug("文件系统事件",
		zap.String("source", e.Source),
		zap.String("path", e.Path),
		zap.String("detail", e.Detail),
	)

	fileDetector.HandleEvent(e.Path, e.Detail)
}

func handleDubiousFile(e event.Event, alerter *notifier.Alerter) {
	logger.Log.Warn("可疑文件事件",
		zap.String("path", e.Path),
		zap.String("reason", e.Reason),
		zap.String("detail", e.Detail),
	)

	alerter.PushAlert(notifier.AlertEvent{
		Type:    "File",
		Path:    e.Path,
		Reason:  "可疑文件事件",
		Details: e.Detail,
	})
}

func handleDubiousProcess(e event.Event, alerter *notifier.Alerter, procDetector *detector.ProcessDetector) {
	logger.Log.Info("可疑进程事件",
		zap.Int32("pid", e.PID),
		zap.String("name", e.Name),
		zap.String("path", e.Path),
	)

	procInfo, ok := e.Data.(storage.DubiousProcessInfo)
	if !ok {
		logger.Log.Error("可疑进程事件数据类型错误",
			zap.Any("data", e.Data),
		)
		return
	}
	procDetector.HandleDubiousProcesses(procInfo)

	alerter.PushAlert(notifier.AlertEvent{
		Type:    "Process",
		Path:    e.Path,
		Reason:  "可疑进程",
		Details: fmt.Sprintf("pid=%d, name=%s", e.PID, e.Name),
	})
}

func handleError(e event.Event) {
	logger.Log.Error("错误事件",
		zap.String("source", e.Source),
		zap.String("reason", e.Reason),
		zap.String("detail", e.Detail),
		zap.Error(e.Err),
	)
}

func handleSystemStart(e event.Event) {
	logger.Log.Info("处理系统启动事件",
		zap.String("source", e.Source),
		zap.String("detail", e.Detail),
	)
}

func handleSystemStop(e event.Event) {
	logger.Log.Info("处理系统停止事件",
		zap.String("source", e.Source),
		zap.String("detail", e.Detail),
	)
}

func handleScriptExecutions(executions []script.ScriptExecution) bool {
	allowDefault := true

	for _, execution := range executions {
		if !execution.Success {
			logger.Log.Warn("脚本执行失败",
				zap.String("script", execution.Script),
				zap.String("event_id", execution.EventID),
				zap.String("event_type", execution.EventType),
				zap.Int64("cost_ms", execution.CostMS),
				zap.String("error", execution.Error),
			)
			continue
		}

		logger.Log.Info("脚本执行成功",
			zap.String("script", execution.Script),
			zap.String("event_id", execution.EventID),
			zap.String("event_type", execution.EventType),
			zap.String("action", execution.Result.Action),
			zap.String("level", execution.Result.Level),
			zap.String("message", execution.Result.Message),
			zap.Bool("allow_default", execution.Result.AllowDefault),
			zap.Int64("cost_ms", execution.CostMS),
		)

		if !execution.Result.AllowDefault {
			allowDefault = false
		}
	}
	return allowDefault
}
