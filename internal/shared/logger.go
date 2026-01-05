package shared

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

func NewLogger(cfg Config) (*slog.Logger, func(), error) {
	dir := filepath.Dir(cfg.LogPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, nil, err
	}

	file, err := os.OpenFile(
		cfg.LogPath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return nil, nil, err
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	multiWriter := io.MultiWriter(os.Stdout, file)

	handler := slog.NewJSONHandler(multiWriter, opts)
	logger := slog.New(handler)

	cleanup := func() {
		_ = file.Close()
	}

	return logger, cleanup, nil
}
