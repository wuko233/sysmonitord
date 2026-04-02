package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Log  *zap.Logger
	atom zap.AtomicLevel
)

func InitLogger() {
	atom = zap.NewAtomicLevelAt(zapcore.InfoLevel)

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), atom),
	)
	Log = zap.New(core, zap.AddCaller())
}

func SetLogLevel(levelStr string) {
	if atom == (zap.AtomicLevel{}) {
		return
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		Log.Warn("无效的日志级别，使用默认级别 info", zap.String("输入级别", levelStr))
		atom.SetLevel(zapcore.InfoLevel)
	} else {
		atom.SetLevel(level)
	}

}

func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
