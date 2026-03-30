package openclaw

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type runnerStub struct {
	name   string
	args   []string
	output []byte
	err    error
}

func (runner *runnerStub) CombinedOutput(_ context.Context, name string, args ...string) ([]byte, error) {
	runner.name = name
	runner.args = append([]string(nil), args...)
	return append([]byte(nil), runner.output...), runner.err
}

func TestAddTelegramChannelUsesAdminContext(t *testing.T) {
	runner := &runnerStub{output: []byte("configured\n")}
	client := NewClient(Config{BinaryPath: "/custom/openclaw"}, runner)

	output, err := client.AddTelegramChannel(context.Background(), "demo-token")
	if err != nil {
		t.Fatalf("add telegram channel: %v", err)
	}
	if strings.TrimSpace(output) != "configured" {
		t.Fatalf("unexpected output: %q", output)
	}
	if runner.name != "sudo" {
		t.Fatalf("unexpected runner name: %s", runner.name)
	}
	want := []string{
		"-u", "admin",
		"HOME=/home/admin",
		"XDG_CONFIG_HOME=/home/admin/.config",
		"XDG_STATE_HOME=/home/admin/.local/state",
		"XDG_DATA_HOME=/home/admin/.local/share",
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"/custom/openclaw",
		"channels", "add", "--channel", "telegram", "--token", "demo-token",
	}
	if !reflect.DeepEqual(runner.args, want) {
		t.Fatalf("unexpected args:\nwant=%q\ngot=%q", want, runner.args)
	}
}

func TestRunLocalAgentAddsDeliveryFlags(t *testing.T) {
	runner := &runnerStub{output: []byte(`{"status":"completed","text":"ready"}`)}
	client := NewClient(Config{}, runner)

	payload, err := client.RunLocalAgent(context.Background(), "platform status", AgentOptions{
		Channel:        "telegram",
		Deliver:        true,
		ReplyChannel:   "telegram",
		ReplyTo:        "6959476905",
		TimeoutSeconds: 120,
	})
	if err != nil {
		t.Fatalf("run local agent: %v", err)
	}
	if payload["status"] != "completed" || payload["text"] != "ready" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	wantTail := []string{
		"/home/admin/.local/share/pnpm/openclaw",
		"agent", "--local", "--message", "platform status", "--json",
		"--channel", "telegram",
		"--deliver",
		"--reply-channel", "telegram",
		"--reply-to", "6959476905",
		"--timeout", "120",
	}
	gotTail := runner.args[len(runner.args)-len(wantTail):]
	if !reflect.DeepEqual(gotTail, wantTail) {
		t.Fatalf("unexpected agent args:\nwant=%q\ngot=%q", wantTail, gotTail)
	}
}

func TestRunLocalAgentIncludesCommandOutputOnFailure(t *testing.T) {
	runner := &runnerStub{output: []byte("boom"), err: errors.New("exit status 1")}
	client := NewClient(Config{}, runner)

	_, err := client.RunLocalAgent(context.Background(), "platform status", AgentOptions{})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected command output in error, got %v", err)
	}
}
