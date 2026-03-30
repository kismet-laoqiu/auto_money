package openclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type CommandRunner interface {
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

type Config struct {
	User       string
	BinaryPath string
	HomeDir    string
	ConfigDir  string
	StateDir   string
	DataDir    string
	PathEnv    string
}

type AgentOptions struct {
	Agent          string
	SessionID      string
	Channel        string
	Deliver        bool
	ReplyChannel   string
	ReplyTo        string
	TimeoutSeconds int
}

type Client struct {
	cfg    Config
	runner CommandRunner
}

func NewClient(cfg Config, runner CommandRunner) *Client {
	if cfg.User == "" {
		cfg.User = "admin"
	}
	if cfg.BinaryPath == "" {
		cfg.BinaryPath = "/home/admin/.local/share/pnpm/openclaw"
	}
	if cfg.HomeDir == "" {
		cfg.HomeDir = "/home/admin"
	}
	if cfg.ConfigDir == "" {
		cfg.ConfigDir = "/home/admin/.config"
	}
	if cfg.StateDir == "" {
		cfg.StateDir = "/home/admin/.local/state"
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "/home/admin/.local/share"
	}
	if cfg.PathEnv == "" {
		cfg.PathEnv = "/usr/local/bin:/usr/bin:/bin"
	}
	if runner == nil {
		runner = execRunner{}
	}
	return &Client{cfg: cfg, runner: runner}
}

func (client *Client) ChannelsList(ctx context.Context) (string, error) {
	out, err := client.run(ctx, "channels", "list")
	return strings.TrimSpace(string(out)), err
}

func (client *Client) AddTelegramChannel(ctx context.Context, token string) (string, error) {
	out, err := client.run(ctx, "channels", "add", "--channel", "telegram", "--token", token)
	return strings.TrimSpace(string(out)), err
}

func (client *Client) RunLocalAgent(ctx context.Context, message string, options AgentOptions) (map[string]any, error) {
	args := []string{"agent", "--local", "--message", message, "--json"}
	if options.Agent != "" {
		args = append(args, "--agent", options.Agent)
	}
	if options.SessionID != "" {
		args = append(args, "--session-id", options.SessionID)
	}
	if options.Channel != "" {
		args = append(args, "--channel", options.Channel)
	}
	if options.Deliver {
		args = append(args, "--deliver")
	}
	if options.ReplyChannel != "" {
		args = append(args, "--reply-channel", options.ReplyChannel)
	}
	if options.ReplyTo != "" {
		args = append(args, "--reply-to", options.ReplyTo)
	}
	if options.TimeoutSeconds > 0 {
		args = append(args, "--timeout", strconv.Itoa(options.TimeoutSeconds))
	}
	return client.runJSON(ctx, args...)
}

func (client *Client) SendTelegramMessage(ctx context.Context, target, message string) (map[string]any, error) {
	return client.runJSON(ctx, "message", "send", "--channel", "telegram", "--target", target, "--message", message, "--json")
}

func (client *Client) runJSON(ctx context.Context, args ...string) (map[string]any, error) {
	out, err := client.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, fmt.Errorf("decode openclaw json: %w output=%s", err, strings.TrimSpace(string(out)))
	}
	return payload, nil
}

func (client *Client) run(ctx context.Context, args ...string) ([]byte, error) {
	sudoArgs := []string{
		"-u", client.cfg.User,
		"HOME=" + client.cfg.HomeDir,
		"XDG_CONFIG_HOME=" + client.cfg.ConfigDir,
		"XDG_STATE_HOME=" + client.cfg.StateDir,
		"XDG_DATA_HOME=" + client.cfg.DataDir,
		"PATH=" + client.cfg.PathEnv,
		client.cfg.BinaryPath,
	}
	sudoArgs = append(sudoArgs, args...)
	out, err := client.runner.CombinedOutput(ctx, "sudo", sudoArgs...)
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			return out, fmt.Errorf("openclaw command failed: %w", err)
		}
		return out, fmt.Errorf("openclaw command failed: %s: %w", message, err)
	}
	return out, nil
}
