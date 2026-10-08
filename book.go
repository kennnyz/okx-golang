package okx

import (
	"errors"
	"sort"
)

// ErrBookOutOfSync means an update did not continue the local book's
// sequence. Call Resync on the subscription to receive a fresh snapshot.
var ErrBookOutOfSync = errors.New("okx: order book out of sync")

// Book is a local order book maintained from BookUpdate pushes. It is not
// safe for concurrent use.
type Book struct {
	InstID string
	Asks   []BookLevel
	Bids   []BookLevel
	SeqID  int64
	Ts     Time
	synced bool
}

// Apply applies a snapshot or an incremental update. An update whose
// PrevSeqID does not match the book's SeqID returns ErrBookOutOfSync once;
// later updates are ignored until the next snapshot.
func (b *Book) Apply(u BookUpdate) error {
	if u.IsSnapshot() {
		b.InstID = u.InstID
		b.Asks = append(b.Asks[:0], u.Asks...)
		b.Bids = append(b.Bids[:0], u.Bids...)
		b.SeqID, b.Ts, b.synced = u.SeqID, u.Ts, true
		return nil
	}
	if !b.synced {
		return nil
	}
	if u.PrevSeqID != b.SeqID {
		b.synced = false
		return ErrBookOutOfSync
	}
	b.Asks = merge(b.Asks, u.Asks, func(a, b float64) bool { return a < b })
	b.Bids = merge(b.Bids, u.Bids, func(a, b float64) bool { return a > b })
	b.SeqID, b.Ts = u.SeqID, u.Ts
	return nil
}

// Synced reports whether the book reflects the latest snapshot plus every
// update since.
func (b *Book) Synced() bool { return b.synced }

// BestBid returns the highest bid.
func (b *Book) BestBid() (BookLevel, bool) {
	if len(b.Bids) == 0 {
		return BookLevel{}, false
	}
	return b.Bids[0], true
}

// BestAsk returns the lowest ask.
func (b *Book) BestAsk() (BookLevel, bool) {
	if len(b.Asks) == 0 {
		return BookLevel{}, false
	}
	return b.Asks[0], true
}

// merge applies changed levels to a side sorted by before. A size of zero
// removes the level.
func merge(side, changes []BookLevel, before func(a, b float64) bool) []BookLevel {
	for _, ch := range changes {
		px := ch.Px.Float64()
		i := sort.Search(len(side), func(i int) bool { return !before(side[i].Px.Float64(), px) })
		found := i < len(side) && side[i].Px.Float64() == px
		switch {
		case ch.Sz.IsZero():
			if found {
				side = append(side[:i], side[i+1:]...)
			}
		case found:
			side[i] = ch
		default:
			side = append(side, BookLevel{})
			copy(side[i+1:], side[i:])
			side[i] = ch
		}
	}
	return side
}
