package detection

import (
	"math"

	"github.com/oomph-ac/oomph/anticheat/oconfig"
	"github.com/oomph-ac/oomph/anticheat/player"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// MovementA reports the disagreement that previously triggered correction.
// It owns evidence only: it never moves the player or changes prediction.
type MovementA struct {
	p        *player.Player
	metadata player.DetectionMetadata
}

func New_MovementA(p *player.Player) *MovementA {
	return &MovementA{p: p, metadata: player.DetectionMetadata{FailBuffer: 1, MaxBuffer: 1}}
}

func (*MovementA) Type() string    { return TypeMovement }
func (*MovementA) SubType() string { return "A" }
func (*MovementA) Description() string {
	return "Reports movement that disagrees with supported physics prediction."
}
func (*MovementA) Punishable() bool                      { return false }
func (d *MovementA) Metadata() *player.DetectionMetadata { return &d.metadata }

func (d *MovementA) Detect(pk packet.Packet) {
	input, ok := pk.(*packet.PlayerAuthInput)
	if !ok {
		return
	}
	comparison := d.p.MovementComparison()
	if !comparison.Eligible || comparison.Frame != input.Tick {
		d.ResetBackend()
		return
	}
	threshold := d.p.Opts().Movement.PredictionThreshold
	if threshold <= 0 || math.IsNaN(float64(threshold)) || math.IsInf(float64(threshold), 0) {
		threshold = oconfig.DefaultConfig.Movement.PredictionThreshold
	}
	// The predicted position is continuous across inputs. Its distance from the
	// reported position already represents deviation; do not sum that distance.
	if offset := comparison.ObservedPosition.Sub(comparison.ExpectedPosition).Len(); offset > threshold {
		d.p.FailDetection(d, "offset", offset, "threshold", threshold, "frame", comparison.Frame)
		d.p.RequestPredictionRebase(comparison.Frame)
	} else {
		d.p.PassDetection(d, .25)
	}
}

func (d *MovementA) ResetBackend() {
	d.metadata.Buffer = 0
}
