package mail

import (
	"fmt"
	"net/smtp"
	"sysmonitord/internal/config"
	"sysmonitord/pkg/logger"

	"go.uber.org/zap"
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

	headers := make(map[string]string)
	headers["From"] = m.cfg.SMTP.Username
	headers["To"] = m.cfg.Recipients[0]
	headers["Subject"] = subject

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	auth := smtp.PlainAuth("", m.cfg.SMTP.Username, m.cfg.SMTP.Password, m.cfg.SMTP.Server)
	addr := fmt.Sprintf("%s:%d", m.cfg.SMTP.Server, m.cfg.SMTP.Port)

	logger.Log.Info("[notifier] 发送邮件通知", zap.String("subject", subject), zap.String("to", m.cfg.Recipients[0]))

	err := smtp.SendMail(addr, auth, m.cfg.SMTP.Username, m.cfg.Recipients, []byte(message))
	if err != nil {
		logger.Log.Error("[notifier] 发送邮件失败", zap.Error(err))
		return err
	}

	logger.Log.Info("[notifier] 邮件发送成功")
	return nil
}
