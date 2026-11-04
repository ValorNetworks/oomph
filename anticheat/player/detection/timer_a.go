package detection

import (
	"time"

	"github.com/oomph-ac/oomph/anticheat/player"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const inputTickDuration = time.Second / player.TicksPerSecond

// TimerA observes the 20-inputs/second allowance that used to discard packets.
// Use elapsed time instead of ServerTick: scheduling jitter must not mint ticks.
type TimerA struct {
	p            *player.Player
	metadata     player.DetectionMetadata
	last         time.Time
	catchupUntil time.Time
	credit       time.Duration
}

func New_TimerA(p *player.Player) *TimerA {
	return &TimerA{p: p, metadata: player.DetectionMetadata{FailBuffer: 3, MaxBuffer: 3}}
}

func (*TimerA) Type() string    { return TypeTimer }
func (*TimerA) SubType() string { return "A" }
func (*TimerA) Description() string {
	return "Reports sustained input processing above the elapsed-time allowance."
}
func (*TimerA) Punishable() bool                      { return false }
func (d *TimerA) Metadata() *player.DetectionMetadata { return &d.metadata }

func (d *TimerA) Detect(pk packet.Packet) {
	if _, ok := pk.(*packet.PlayerAuthInput); !ok {
		return
	}
	if !d.p.Ready || !d.p.Alive {
		d.ResetBackend()
		return
	}
	now := d.p.Time()
	// Stable latency shifts arrival time without changing the rate. Allow one
	// second of batching, or two measured round trips plus scheduling slack.
	burst := max(time.Second, min(2*d.p.StackLatency+250*time.Millisecond, 5*time.Second))
	if d.last.IsZero() {
		d.last, d.credit = now, burst
	}
	elapsed := now.Sub(d.last)
	d.last = now
	if elapsed < 0 {
		d.ResetBackend()
		return
	}
	if elapsed > burst {
		// A stalled network/server can deliver queued legitimate inputs together.
		// Temporarily fund that exact gap, then expire unused catch-up credit.
		d.credit = burst + elapsed
		d.catchupUntil = now.Add(burst)
		d.metadata.Buffer = 0
	} else {
		d.credit += elapsed
	}
	if !now.Before(d.catchupUntil) {
		d.credit = min(d.credit, burst)
	}
	d.credit -= inputTickDuration
	if d.credit < 0 {
		d.p.FailDetection(d, "excess_ms", float64(-d.credit)/float64(time.Millisecond))
		// Do not leave permanent debt after the input rate returns to normal.
		d.credit = 0
	} else {
		d.p.PassDetection(d, .25)
	}
}

func (d *TimerA) ResetBackend() {
	d.last, d.catchupUntil = time.Time{}, time.Time{}
	d.credit = 0
	d.metadata.Buffer = 0
}
