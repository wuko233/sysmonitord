package notifier

import (
	"fmt"
	"sync"
	"sysmonitord/internal/config"
	"sysmonitord/internal/notifier/mail"
	"sysmonitord/pkg/logger"
	"time"

	"go.uber.org/zap"
)

type AlertEvent struct {
	Type    string
	Path    string
	Reason  string
	Details string
}

type Alerter struct {
	cfg       config.NotificationConfig
	mailer    *mail.Mailer
	eventChan chan AlertEvent
	buffer    []AlertEvent
	mu        sync.Mutex
	timer     *time.Timer
	interval  time.Duration
}

func NewAlerter(cfg config.NotificationConfig) *Alerter {
	return &Alerter{
		cfg:       cfg,
		mailer:    mail.NewMailer(cfg.Email),
		eventChan: make(chan AlertEvent, 100),
		buffer:    make([]AlertEvent, 0),
		interval:  30 * time.Second, // Todo: 可配置化
	}
}

func (a *Alerter) Start() {
	go a.loop()
}

func (a *Alerter) PushAlert(event AlertEvent) {
	select {
	case a.eventChan <- event:
		logger.Log.Debug("[notifier] 推送告警事件", zap.String("path", event.Path), zap.String("reason", event.Reason))
	default:
		logger.Log.Warn("[notifier] 告警事件通道已满，丢弃告警", zap.String("path", event.Path), zap.String("reason", event.Reason))
	}
}

func (a *Alerter) loop() {
	a.timer = time.NewTimer(a.interval)

	for {
		select {
		case event := <-a.eventChan:
			a.mu.Lock()
			a.buffer = append(a.buffer, event)
			a.mu.Unlock()
			logger.Log.Debug("[notifier] 收到告警，加入待发送序列", zap.String("path", event.Path))

		case <-a.timer.C:
			logger.Log.Debug("[notifier] 定时检查告警事件，准备发送", zap.Int("count", len(a.buffer)))
			a.mu.Lock()
			if len(a.buffer) > 0 {
				a.sendAlert()
				a.buffer = make([]AlertEvent, 0)
			}
			a.mu.Unlock()

			a.timer.Reset(a.interval)
		}
	}
}

func (a *Alerter) sendAlert() {
	if len(a.buffer) == 0 {
		return
	}

	subject := fmt.Sprintf("【Sysmonitor】新增 %d 个告警", len(a.buffer))
	body := "以下是最近的告警事件：\n\n"

	for _, event := range a.buffer {
		body += fmt.Sprintf("- [%s] %s: %s (%s)\n", event.Type, event.Path, event.Reason, event.Details)
	}

	body += "\n请及时关注系统安全状况。"

	if err := a.mailer.Send(subject, body); err != nil {
		logger.Log.Error("[notifier] 发送告警邮件失败", zap.Error(err))
	} else {
		logger.Log.Debug("[notifier] 告警邮件发送成功", zap.Int("count", len(a.buffer)))
	}
}
