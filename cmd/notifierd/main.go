package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"quantlab/internal/config"
	"quantlab/internal/platform/notifier"
	sqlitepkg "quantlab/internal/store/sqlite"
)

func main() {
	fs := flag.NewFlagSet("notifierd", flag.ContinueOnError)
	configPath := fs.String("config", "configs/live.yaml", "config file")
	stateDBPath := fs.String("state-db", "", "override sqlite state db path")
	healthAddr := fs.String("health-addr", "", "optional health listen address")
	once := fs.Bool("once", false, "process currently available events and exit")
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *stateDBPath == "" {
		*stateDBPath = cfg.Live.Runtime.StateDBPath
	}
	store, err := sqlitepkg.NewStore(*stateDBPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	service := notifier.New(notifier.Config{
		DingTalkWebhook:   os.Getenv("DINGTALK_WEBHOOK"),
		DingTalkSecret:    os.Getenv("DINGTALK_SECRET"),
		TelegramBotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramAlertChat: firstNonEmpty(os.Getenv("TELEGRAM_ALERT_CHAT_ID"), os.Getenv("TELEGRAM_COMMAND_CHAT_ID")),
	})
	runtime := notifier.NewRuntime(notifier.RuntimeConfig{
		Store:        store,
		Service:      service,
		PollInterval: time.Second,
	})
	commandRuntime := notifier.NewCommandRuntime(notifier.CommandRuntimeConfig{
		StateStore:      store,
		Service:         service,
		PlatformBaseURL: firstNonEmpty(os.Getenv("PLATFORM_ADDR"), "http://127.0.0.1:18080"),
		AllowedChatID:   os.Getenv("TELEGRAM_COMMAND_CHAT_ID"),
		AllowedUserIDs:  parseEnvList(os.Getenv("TELEGRAM_ADMIN_USER_IDS")),
		ConfigDir:       filepath.Clean(filepath.Dir(*configPath)),
	})
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if *once {
		if err := runtime.ProcessAvailable(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := commandRuntime.ProcessAvailable(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	errCh := make(chan error, 3)
	if *healthAddr != "" {
		server := &http.Server{Addr: *healthAddr, Handler: newHealthHandler(runtime, commandRuntime)}
		go func() {
			<-ctx.Done()
			_ = server.Shutdown(context.Background())
		}()
		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}()
	}
	go func() {
		err := runtime.Run(ctx)
		if err == context.Canceled {
			err = nil
		}
		errCh <- err
	}()
	if commandRuntime.Enabled() {
		go func() {
			err := commandRuntime.Run(ctx)
			if err == context.Canceled {
				err = nil
			}
			errCh <- err
		}()
	}
	if err := <-errCh; err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newHealthHandler(runtime *notifier.Runtime, commandRuntime *notifier.CommandRuntime) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		payload := map[string]any{"ok": true}
		if runtime != nil {
			payload["notifier"] = runtime.Health()
		}
		if commandRuntime != nil {
			payload["telegram_command"] = commandRuntime.Health()
		}
		writeJSON(writer, http.StatusOK, payload)
	})
	return mux
}

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func parseEnvList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.Trim(field, `"'`)
		if field != "" {
			values = append(values, field)
		}
	}
	return values
}
