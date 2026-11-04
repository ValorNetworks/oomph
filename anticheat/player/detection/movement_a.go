package detection

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/oomph-ac/oomph/anticheat/oconfig"
	"github.com/oomph-ac/oomph/anticheat/player"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// MovementA reports the disagreement that previously triggered correction.
// It owns evidence only: it never moves the player or changes prediction.
type MovementA struct {
	p        *player.Player
	metadata player.DetectionMetadata
	residual mgl32.Vec3
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
	// Prediction rebases its position each input. Integrate signed displacement
	// error so persistent sub-threshold speed changes are not erased by rebasing.
	d.residual = d.residual.Add(comparison.ObservedPosition.Sub(comparison.ExpectedPosition))
	if offset := d.residual.Len(); offset > threshold {
		d.p.FailDetection(d, "offset", offset, "threshold", threshold, "frame", comparison.Frame)
		// A flag replaces the old correction boundary. Discard only detector
		// evidence; expected velocity, observed input and gameplay stay intact.
		d.residual = mgl32.Vec3{}
	} else {
		d.p.PassDetection(d, .25)
	}
}

func (d *MovementA) ResetBackend() {
	d.residual = mgl32.Vec3{}
	d.metadata.Buffer = 0
}
