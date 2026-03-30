package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	Level     slog.Leveler
	Format    string
	AddSource bool
	Output    string
	Stdout    io.Writer
}

type multiCloser struct {
	closers []io.Closer
}

func (m multiCloser) Close() error {
	var errs []error
	for _, closer := range m.closers {
		if closer == nil {
			continue
		}
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("close logger outputs: %v", errs)
}

func NewLogger(opts Options) (*slog.Logger, io.Closer, error) {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Level == nil {
		opts.Level = slog.LevelInfo
	}
	if opts.Format == "" {
		opts.Format = "json"
	}

	writer := opts.Stdout
	var closers []io.Closer

	if output := strings.TrimSpace(opts.Output); output != "" {
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return nil, nil, fmt.Errorf("create log directory: %w", err)
		}
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("open log file: %w", err)
		}
		writer = io.MultiWriter(opts.Stdout, file)
		closers = append(closers, file)
	}

	handlerOpts := &slog.HandlerOptions{
		Level:     opts.Level,
		AddSource: opts.AddSource,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				if t, ok := attr.Value.Any().(time.Time); ok {
					attr.Value = slog.StringValue(t.Format(time.RFC3339))
				}
			}
			if attr.Key == slog.LevelKey {
				attr.Value = slog.StringValue(strings.ToUpper(attr.Value.String()))
			}
			return attr
		},
	}

	var handler slog.Handler
	if strings.EqualFold(opts.Format, "text") {
		handler = slog.NewTextHandler(writer, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(writer, handlerOpts)
	}

	logger := slog.New(handler)
	if len(closers) == 0 {
		return logger, nil, nil
	}
	return logger, multiCloser{closers: closers}, nil
}

func Component(logger *slog.Logger, component string) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	if strings.TrimSpace(component) == "" {
		return logger
	}
	return logger.With("component", component)
}
