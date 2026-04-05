package mail

import (
	"fmt"
	"sysmonitord/internal/config"
	"sysmonitord/pkg/logger"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

type Mailer struct {
	cfg config.EmailConfig
}

func NewMailer(cfg config.EmailConfig) *Mailer {
	return &Mailer{cfg: cfg}
}

func (m *Mailer) Send(subject, body string) error {
	if !m.cfg.Enabled {
		logger.Log.Debug("[notifier] 未启用邮件通知，跳过....")
		return nil
	}

	if m.cfg.SMTP.Server == "" || m.cfg.SMTP.Port == 0 {
		logger.Log.Error("[notifier] SMTP配置缺失",
			zap.String("server", m.cfg.SMTP.Server),
			zap.Int("port", m.cfg.SMTP.Port),
		)
		return fmt.Errorf("SMTP配置缺失")
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", m.cfg.SMTP.Username)
	msg.SetHeader("To", m.cfg.Recipients...)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", body)

	d := gomail.NewDialer(
		m.cfg.SMTP.Server,
		m.cfg.SMTP.Port,
		m.cfg.SMTP.Username,
		m.cfg.SMTP.Password,
	)

	if m.cfg.SMTP.Port == 465 {
		d.SSL = true
	}

	if err := d.DialAndSend(msg); err != nil {
		logger.Log.Error("[notifier] 邮件发送失败", zap.Error(err))
		return err
	}
	logger.Log.Info("[notifier] 邮件发送成功", zap.Strings("recipients", m.cfg.Recipients))
	return nil
}
