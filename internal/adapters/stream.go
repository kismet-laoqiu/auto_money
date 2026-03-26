package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"quantlab/internal/config"
	"quantlab/internal/core"
)

type bitgetSubscribe struct {
	Op   string              `json:"op"`
	Args []map[string]string `json:"args"`
}

type bitgetMessage struct {
	Event  string     `json:"event"`
	Action string     `json:"action"`
	Code   string     `json:"code"`
	Msg    string     `json:"msg"`
	Arg    bitgetArg  `json:"arg"`
	Data   [][]string `json:"data"`
}

type bitgetArg struct {
	InstType string `json:"instType"`
	Channel  string `json:"channel"`
	InstID   string `json:"instId"`
}

type streamCache struct {
	Bars      []core.Bar
	LastAlert string
}

const (
	defaultBitgetStreamURL = "wss://ws.bitget.com/v2/ws/public"
	streamPingInterval     = 30 * time.Second
	streamReadTimeout      = 90 * time.Second
	streamMaxBars          = 1500
)

func RunBitgetStream(ctx context.Context, cfg config.Config) error {
	backoff := 2 * time.Second
	for {
		err := runBitgetStreamSession(ctx, cfg)
		if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil
		}
		fmt.Fprintf(os.Stderr, "bitget stream reconnecting after error: %v\n", err)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func runBitgetStreamSession(ctx context.Context, cfg config.Config) error {
	url := cfg.Stream.BitgetURL
	if url == "" {
		url = defaultBitgetStreamURL
	}
	interval := cfg.Stream.Interval
	if interval == "" {
		interval = "1m"
	}
	if len(cfg.Stream.Symbols) == 0 {
		return fmt.Errorf("stream symbols empty")
	}

	dialer := websocket.Dialer{HandshakeTimeout: 15 * time.Second}
	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))

	conn.SetReadLimit(1 << 20)
	_ = conn.SetReadDeadline(time.Now().Add(streamReadTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(streamReadTimeout))
	})

	client := NewClient()
	notifier := NewNotifier(cfg.Notify)
	caches := map[string]*streamCache{}
	for _, symbol := range cfg.Stream.Symbols {
		bars, err := client.FetchBars(ctx, config.DatasetConfig{
			Provider: "bitget",
			Symbol:   symbol,
			Interval: interval,
			Limit:    cfg.Stream.SnapshotLimit,
		})
		if err != nil {
			return err
		}
		caches[symbol] = &streamCache{Bars: bars}
	}

	args := make([]map[string]string, 0, len(cfg.Stream.Symbols))
	channel := bitgetCandleChannel(interval)
	for _, symbol := range cfg.Stream.Symbols {
		args = append(args, map[string]string{
			"instType": "SPOT",
			"channel":  channel,
			"instId":   symbol,
		})
	}
	if err := conn.WriteJSON(bitgetSubscribe{Op: "subscribe", Args: args}); err != nil {
		return err
	}

	readDone := make(chan struct{})
	defer close(readDone)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-readDone:
		}
	}()

	pingTicker := time.NewTicker(streamPingInterval)
	defer pingTicker.Stop()
	pingErr := make(chan error, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-pingTicker.C:
				if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
					select {
					case pingErr <- err:
					default:
					}
					return
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-pingErr:
			if ctx.Err() != nil {
				return nil
			}
			return err
		default:
		}

		_, payload, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		_ = conn.SetReadDeadline(time.Now().Add(streamReadTimeout))
		if string(payload) == "pong" {
			continue
		}

		var message bitgetMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			continue
		}
		if message.Event == "error" {
			return fmt.Errorf("bitget websocket error code=%s msg=%s", message.Code, message.Msg)
		}
		if len(message.Data) == 0 {
			continue
		}

		cache := caches[message.Arg.InstID]
		if cache == nil {
			continue
		}
		for _, row := range message.Data {
			bar, err := parseBitgetStreamBar(row)
			if err != nil {
				continue
			}
			closedIndex, advanced := updateStreamCache(cache, bar)
			if !advanced || closedIndex < 0 {
				continue
			}
			signal := core.EvaluateSignal(cache.Bars[:closedIndex+1], closedIndex, cfg.Strategy)
			if signal.Side == core.Flat {
				continue
			}
			closedBar := cache.Bars[closedIndex]
			alertKey := fmt.Sprintf("%s|%s|%s", message.Arg.InstID, signal.Side, closedBar.Time.UTC().Format(time.RFC3339))
			if cache.LastAlert == alertKey {
				continue
			}
			text := formatStreamAlert(message.Arg.InstID, interval, closedBar, signal)
			if err := notifier.Notify(ctx, text); err != nil {
				fmt.Fprintf(os.Stderr, "notify failed symbol=%s bar=%s err=%v\n", message.Arg.InstID, closedBar.Time.UTC().Format(time.RFC3339), err)
				continue
			}
			cache.LastAlert = alertKey
		}
	}
}

func parseBitgetStreamBar(row []string) (core.Bar, error) {
	if len(row) < 5 {
		return core.Bar{}, fmt.Errorf("bitget row too short")
	}
	ts, err := asInt64(row[0])
	if err != nil {
		return core.Bar{}, err
	}
	openValue, _ := asFloat64(row[1])
	highValue, _ := asFloat64(row[2])
	lowValue, _ := asFloat64(row[3])
	closeValue, _ := asFloat64(row[4])
	volumeValue := 0.0
	if len(row) > 5 {
		volumeValue, _ = asFloat64(row[5])
	}
	return core.Bar{Time: time.UnixMilli(ts).UTC(), Open: openValue, High: highValue, Low: lowValue, Close: closeValue, Volume: volumeValue}, nil
}

func updateStreamCache(cache *streamCache, bar core.Bar) (int, bool) {
	if len(cache.Bars) == 0 {
		cache.Bars = append(cache.Bars, bar)
		return -1, false
	}
	last := cache.Bars[len(cache.Bars)-1]
	if bar.Time.Equal(last.Time) {
		cache.Bars[len(cache.Bars)-1] = bar
		return -1, false
	}
	if bar.Time.Before(last.Time) {
		return -1, false
	}
	cache.Bars = append(cache.Bars, bar)
	if len(cache.Bars) > streamMaxBars {
		cache.Bars = cache.Bars[len(cache.Bars)-streamMaxBars:]
	}
	return len(cache.Bars) - 2, true
}

func formatStreamAlert(symbol, interval string, bar core.Bar, signal core.Signal) string {
	return strings.Join([]string{
		"[quant-lab]",
		fmt.Sprintf("symbol=%s interval=%s side=%s score=%.2f", symbol, interval, signal.Side, signal.Score),
		fmt.Sprintf("bar=%s close=%.4f", bar.Time.UTC().Format(time.RFC3339), bar.Close),
		fmt.Sprintf("entry=%.4f stop=%.4f target=%.4f", signal.Entry, signal.Stop, signal.Target),
		fmt.Sprintf("reasons=%s", core.JoinReasons(signal.Reasons)),
	}, "\n")
}

func bitgetCandleChannel(interval string) string {
	switch strings.ToLower(interval) {
	case "1m":
		return "candle1m"
	case "5m":
		return "candle5m"
	case "15m":
		return "candle15m"
	case "30m":
		return "candle30m"
	case "1h":
		return "candle1H"
	case "4h":
		return "candle4H"
	case "6h":
		return "candle6H"
	case "12h":
		return "candle12H"
	case "1d":
		return "candle1D"
	default:
		return "candle1m"
	}
}
