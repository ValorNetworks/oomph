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

func TestMovementAccumulatesSmallPersistentDisagreement(t *testing.T) {
	p, _, d, _ := movementDetector(t)
	observationInput(t, p, 1, mgl32.Vec3{2, 64, 2})
	observationInput(t, p, 2, mgl32.Vec3{2.2, 64, 2})
	if d.Metadata().Violations != 0 {
		t.Fatal("one sub-threshold displacement flagged")
	}
	for frame := uint64(3); frame <= 20; frame++ {
		observationInput(t, p, frame, mgl32.Vec3{2 + float32(frame-1)*.2, 64, 2})
	}
	if d.Metadata().Violations == 0 {
		t.Fatal("rebasing erased repeated displacement error")
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
