package live

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"quantlab/internal/exchange/bitget"
)

func TestServiceFlattenSymbolRunsExecdAndReadsBackFlatPosition(t *testing.T) {
	runner := &runnerStub{}
	reader := &readerStub{
		position: bitget.SinglePositionSnapshot{Qty: 0, Mode: "one_way_mode"},
	}
	service := NewService(Config{
		ExecdPath:      "/tmp/quantlab-execd",
			ConfigPath:     "configs/live.yaml",
			StateDBPath:    "var/live-state.db",
		ProductType:    "USDT-FUTURES",
		MarginCoin:     "USDT",
		AllowedSymbols: []string{"MSTRUSDT"},
		Runner:         runner,
		Reader:         reader,
		Now:            func() time.Time { return time.Unix(1710000000, 0).UTC() },
	})

	result, err := service.FlattenSymbol(context.Background(), "MSTRUSDT")
	if err != nil {
		t.Fatalf("flatten symbol: %v", err)
	}
	if !reflect.DeepEqual(runner.args, []string{"-config", "configs/live.yaml", "-state-db", "var/live-state.db", "-flatten-symbol", "MSTRUSDT"}) {
		t.Fatalf("unexpected runner args: %+v", runner.args)
	}
	if reader.symbol != "MSTRUSDT" || reader.productType != "USDT-FUTURES" || reader.marginCoin != "USDT" {
		t.Fatalf("unexpected readback request: %+v", reader)
	}
	if result.Symbol != "MSTRUSDT" || result.Status != "flat" || result.Qty != 0 || !result.CheckedAt.Equal(time.Unix(1710000000, 0).UTC()) {
		t.Fatalf("unexpected flatten result: %+v", result)
	}
}

func TestServiceFlattenSymbolRejectsUnknownSymbol(t *testing.T) {
	service := NewService(Config{
		AllowedSymbols: []string{"MSTRUSDT"},
		Runner:         &runnerStub{},
		Reader:         &readerStub{},
	})

	if _, err := service.FlattenSymbol(context.Background(), "BTCUSDT"); err == nil {
		t.Fatal("expected unknown symbol error")
	}
}

type runnerStub struct {
	path string
	args []string
	err  error
}

func (runner *runnerStub) Run(_ context.Context, execdPath string, args ...string) error {
	runner.path = execdPath
	runner.args = append([]string(nil), args...)
	return runner.err
}

type readerStub struct {
	symbol      string
	productType string
	marginCoin  string
	position    bitget.SinglePositionSnapshot
	err         error
}

func (reader *readerStub) FetchSinglePosition(_ context.Context, symbol, productType, marginCoin string) (bitget.SinglePositionSnapshot, error) {
	reader.symbol = symbol
	reader.productType = productType
	reader.marginCoin = marginCoin
	if reader.err != nil {
		return bitget.SinglePositionSnapshot{}, reader.err
	}
	return reader.position, nil
}

func TestServiceFlattenSymbolReturnsRunnerFailure(t *testing.T) {
	service := NewService(Config{
		ExecdPath:      "/tmp/quantlab-execd",
			ConfigPath:     "configs/live.yaml",
			StateDBPath:    "var/live-state.db",
		ProductType:    "USDT-FUTURES",
		MarginCoin:     "USDT",
		AllowedSymbols: []string{"MSTRUSDT"},
		Runner:         &runnerStub{err: errors.New("execd failed")},
		Reader:         &readerStub{},
	})

	if _, err := service.FlattenSymbol(context.Background(), "MSTRUSDT"); err == nil {
		t.Fatal("expected runner error")
	}
}
