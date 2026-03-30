package qq

import (
	"fmt"
	"log/slog"
)

type logHandler struct {
	logger *slog.Logger
	name   string
}

func NewLogHandler(logger *slog.Logger, name string) *logHandler {
	return &logHandler{logger: logger, name: name}
}

func (l *logHandler) Info(v ...any) {
	l.logger.Info(formatMessage(v...), "name", l.name)
}

func (l *logHandler) Error(v ...any) {
	l.logger.Error(formatMessage(v...), "name", l.name)
}

func (l *logHandler) Debug(v ...any) {
	l.logger.Debug(formatMessage(v...), "name", l.name)
}

func (l *logHandler) Warn(v ...any) {
	l.logger.Warn(formatMessage(v...), "name", l.name)
}

func (l *logHandler) Debugf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Debug(msg, "name", l.name)
}

func (l *logHandler) Infof(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Info(msg, "name", l.name)
}

func (l *logHandler) Warnf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Warn(msg, "name", l.name)
}

func (l *logHandler) Errorf(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	l.logger.Error(msg, "name", l.name)
}

func (l *logHandler) Sync() error {
	return nil
}

func formatMessage(v ...any) string {
	if len(v) == 0 {
		return ""
	}
	msg, ok := v[0].(string)
	if !ok {
		return fmt.Sprint(v...)
	}
	if len(v) == 1 {
		return msg
	}
	args := make([]any, len(v)-1)
	copy(args, v[1:])
	return msg + " " + fmt.Sprint(args...)
}
