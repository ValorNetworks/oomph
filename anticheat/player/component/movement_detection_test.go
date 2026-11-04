package component

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	dfcube "github.com/df-mc/dragonfly/server/block/cube"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/oomph-ac/oomph/anticheat/player"
	packetcontext "github.com/oomph-ac/oomph/anticheat/player/context"
	"github.com/oomph-ac/oomph/anticheat/player/detection"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func movementDetector(t *testing.T) (*player.Player, *AuthoritativeMovementComponent, *detection.MovementA, *bytes.Buffer) {
	t.Helper()
	p, m := observationPlayer(t)
	output := new(bytes.Buffer)
	p.SetLog(slog.New(slog.NewTextHandler(output, nil)))
	p.Opts().Movement.PredictionThreshold = 1.2
	m.ticksSinceKb = 1000
	d := detection.New_MovementA(p)
	p.RegisterDetection(d)
	return p, m, d, output
}

func TestMovementDetectionLogsWithoutChangingInputOrSendingCorrection(t *testing.T) {
	p, _, d, output := movementDetector(t)
	for frame := uint64(1); frame <= 30; frame++ {
		observationInput(t, p, frame, mgl32.Vec3{2, 64, 2})
	}
	if d.Metadata().Violations != 0 {
		t.Fatal("stationary supported player flagged")
	}
	observationInput(t, p, 31, mgl32.Vec3{2, 75, 2})
	if d.Metadata().Violations == 0 || !strings.Contains(output.String(), "failed detection") || !strings.Contains(output.String(), "dtc=Movement") {
		t.Fatalf("movement disagreement did not produce a normal console flag: %s", output)
	}
	if p.Closed || d.Punishable() {
		t.Fatal("detection-only movement can disconnect a player")
	}
}

func TestConstantPredictionOffsetIsNotCountedRepeatedly(t *testing.T) {
	p, _, d, _ := movementDetector(t)
	for frame := uint64(1); frame <= 1000; frame++ {
		observationInput(t, p, frame, mgl32.Vec3{2.2, 64, 2})
		comparison := p.MovementComparison()
		if !comparison.Eligible {
			t.Fatalf("frame %d did not exercise supported prediction", frame)
		}
		offset := comparison.ObservedPosition.Sub(comparison.ExpectedPosition).Len()
		if offset < .19 || offset > .21 {
			t.Fatalf("prediction was restarted from observations: %v", comparison)
		}
	}
	if d.Metadata().Violations != 0 {
		t.Fatal("one tolerated position offset became a running total")
	}
}

func TestGrowingPositionDeviationStillCrossesTheOriginalThreshold(t *testing.T) {
	p, _, d, _ := movementDetector(t)
	observationInput(t, p, 1, mgl32.Vec3{2, 64, 2})
	for frame := uint64(2); frame <= 6; frame++ {
		observationInput(t, p, frame, mgl32.Vec3{2 + float32(frame-1)*.2, 64, 2})
		if d.Metadata().Violations != 0 {
			t.Fatal("position difference below tolerance flagged")
		}
	}
	observationInput(t, p, 7, mgl32.Vec3{3.5, 64, 2})
	if d.Metadata().Violations != 1 {
		t.Fatal("continuous prediction did not catch growing deviation")
	}
}

func TestOutlyingPositionCannotDiscardPredictionGeometry(t *testing.T) {
	p, _, d, _ := movementDetector(t)
	p.WorldUpdater().SetChunkRadius(1)
	observationInput(t, p, 1, mgl32.Vec3{2, 64, 2})
	observationInput(t, p, 2, mgl32.Vec3{160, 64, 2})
	if !p.MovementComparison().Eligible || d.Metadata().Violations != 1 {
		t.Fatal("reported position evicted the geometry before prediction could check it")
	}
}

type predictionCaptureCheck struct {
	observationCheck
	p        *player.Player
	position mgl32.Vec3
}

func (d *predictionCaptureCheck) Detect(packet.Packet) { d.position = d.p.Movement().Pos() }

func TestFlagReconcilesOnlyInternalPositionAfterAllChecks(t *testing.T) {
	p, m, d, _ := movementDetector(t)
	capture := &predictionCaptureCheck{p: p}
	p.RegisterDetection(capture)
	observationInput(t, p, 1, mgl32.Vec3{2, 64, 2})
	m.ServerUpdate(&packet.SetActorMotion{EntityRuntimeID: 1, Velocity: mgl32.Vec3{.4, .4, 0}})
	a := p.ACKs().(*ACKComponent)
	queued := len(a.currentBatch.acks)
	m.SetVel(mgl32.Vec3{.2, 0, 0})
	observationInput(t, p, 2, mgl32.Vec3{6, 64, 2})
	comparison := p.MovementComparison()
	if d.Metadata().Violations != 1 {
		t.Fatal("large disagreement did not flag")
	}
	if m.Pos() != comparison.ObservedPosition {
		t.Fatal("internal reference was not reconciled after the flag")
	}
	if capture.position != comparison.ExpectedPosition {
		t.Fatal("internal reconciliation changed evidence before other checks ran")
	}
	if comparison.ExpectedPosition == comparison.ObservedPosition {
		t.Fatal("prediction evidence was erased")
	}
	if m.Vel() != comparison.ExpectedVelocity {
		t.Fatal("internal position reconciliation replaced expected velocity")
	}
	if len(a.currentBatch.acks) != queued {
		t.Fatal("internal reconciliation discarded queued server state")
	}
	for frame := uint64(3); frame <= 8; frame++ {
		observationInput(t, p, frame, m.Pos())
	}
	if d.Metadata().Violations != 1 {
		t.Fatal("one old offset produced repeated flags after reconciliation")
	}
}

func TestMovementRetainsRawJumpExemption(t *testing.T) {
	p, _, d, _ := movementDetector(t)
	observationInput(t, p, 1, mgl32.Vec3{2, 64, 2})
	input := &packet.PlayerAuthInput{Tick: 2, Position: mgl32.Vec3{2, 76.62001, 2}, InputData: protocol.NewInputFlags(packet.InputFlagCount)}
	input.InputData.Set(packet.InputFlagJumpPressedRaw)
	var pk packet.Packet = input
	p.HandleClientPacket(packetcontext.NewHandlePacketContext(&pk))
	if p.MovementComparison().Eligible || d.Metadata().Violations != 0 {
		t.Fatal("old raw-jump exclusion was lost")
	}
}

func TestMovementSkipsUnavailableSimulationAndRealServerMotion(t *testing.T) {
	for name, prepare := range map[string]func(*player.Player, *AuthoritativeMovementComponent){
		"immobile": func(_ *player.Player, m *AuthoritativeMovementComponent) { m.SetImmobile(true) },
		"teleport": func(_ *player.Player, m *AuthoritativeMovementComponent) {
			m.Teleport(mgl32.Vec3{10, 64, 10}, true, false)
		},
		"pending_teleport": func(_ *player.Player, m *AuthoritativeMovementComponent) { m.AddPendingTeleport() },
		"knockback":        func(_ *player.Player, m *AuthoritativeMovementComponent) { m.SetKnockback(mgl32.Vec3{.4, .4, 0}) },
		"missing_chunks":   func(p *player.Player, _ *AuthoritativeMovementComponent) { p.World().PurgeChunks() },
		"unsupported_liquid": func(p *player.Player, _ *AuthoritativeMovementComponent) {
			p.World().SetBlock(dfcube.Pos{2, 64, 2}, block.Water{Depth: 8}, nil)
		},
	} {
		t.Run(name, func(t *testing.T) {
			p, m, d, _ := movementDetector(t)
			observationInput(t, p, 1, mgl32.Vec3{2, 64, 2})
			prepare(p, m)
			observationInput(t, p, 2, mgl32.Vec3{2, 75, 2})
			if p.MovementComparison().Eligible || d.Metadata().Violations != 0 {
				t.Fatal("exempt/unavailable movement was flagged")
			}
		})
	}
}
