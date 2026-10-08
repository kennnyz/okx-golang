package okx

import (
	"errors"
	"testing"
)

func lv(px, sz string) BookLevel { return BookLevel{Px: Number(px), Sz: Number(sz)} }

func TestBookApply(t *testing.T) {
	var b Book
	if err := b.Apply(BookUpdate{Action: "update", OrderBook: OrderBook{SeqID: 1}}); err != nil || b.Synced() {
		t.Fatalf("update before snapshot: %v, synced %v", err, b.Synced())
	}
	err := b.Apply(BookUpdate{InstID: "BTC-USDT", Action: "snapshot", OrderBook: OrderBook{
		Asks:  []BookLevel{lv("101", "1"), lv("102", "1"), lv("104", "1")},
		Bids:  []BookLevel{lv("100", "1"), lv("99", "1")},
		SeqID: 10, PrevSeqID: -1, Ts: TimeFromMillis(1),
	}})
	if err != nil {
		t.Fatal(err)
	}
	err = b.Apply(BookUpdate{Action: "update", OrderBook: OrderBook{
		Asks:  []BookLevel{lv("101", "0"), lv("103", "2"), lv("104", "5"), lv("105", "0")},
		Bids:  []BookLevel{lv("100.5", "3"), lv("98", "1")},
		SeqID: 11, PrevSeqID: 10, Ts: TimeFromMillis(2),
	}})
	if err != nil {
		t.Fatal(err)
	}
	assertSide(t, "asks", b.Asks, "102:1", "103:2", "104:5")
	assertSide(t, "bids", b.Bids, "100.5:3", "100:1", "99:1", "98:1")

	if ask, _ := b.BestAsk(); ask.Px != "102" {
		t.Fatalf("best ask %v", ask.Px)
	}
	if bid, _ := b.BestBid(); bid.Px != "100.5" {
		t.Fatalf("best bid %v", bid.Px)
	}
	if err := b.Apply(BookUpdate{Action: "update", OrderBook: OrderBook{SeqID: 11, PrevSeqID: 11, Ts: TimeFromMillis(3)}}); err != nil {
		t.Fatalf("heartbeat update: %v", err)
	}
	if err := b.Apply(BookUpdate{Action: "update", OrderBook: OrderBook{SeqID: 13, PrevSeqID: 12}}); !errors.Is(err, ErrBookOutOfSync) || b.Synced() {
		t.Fatalf("gap not detected: %v", err)
	}
	if err := b.Apply(BookUpdate{Action: "update", OrderBook: OrderBook{SeqID: 14, PrevSeqID: 13}}); err != nil || b.SeqID != 11 {
		t.Fatalf("update while out of sync: %v, seq %d", err, b.SeqID)
	}
	if err := b.Apply(BookUpdate{Action: "snapshot", OrderBook: OrderBook{SeqID: 20}}); err != nil || !b.Synced() {
		t.Fatalf("snapshot after gap: %v", err)
	}
}

func assertSide(t *testing.T, name string, side []BookLevel, want ...string) {
	t.Helper()
	if len(side) != len(want) {
		t.Fatalf("%s = %v, want %v", name, side, want)
	}
	for i, l := range side {
		if got := string(l.Px) + ":" + string(l.Sz); got != want[i] {
			t.Fatalf("%s[%d] = %s, want %s", name, i, got, want[i])
		}
	}
}
