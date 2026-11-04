package component

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	dfcube "github.com/df-mc/dragonfly/server/block/cube"
	dfworld "github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/oomph-ac/oomph/anticheat/player"
	packetcontext "github.com/oomph-ac/oomph/anticheat/player/context"
	oworld "github.com/oomph-ac/oomph/anticheat/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type observationCheck struct {
	count    int
	metadata player.DetectionMetadata
}

func (*observationCheck) Type() string                          { return "Observation" }
func (*observationCheck) SubType() string                       { return "Test" }
func (*observationCheck) Description() string                   { return "Counts real dispatches" }
func (*observationCheck) Punishable() bool                      { return false }
func (d *observationCheck) Metadata() *player.DetectionMetadata { return &d.metadata }
func (d *observationCheck) Detect(packet.Packet)                { d.count++ }

func observationPlayer(t *testing.T) (*player.Player, *AuthoritativeMovementComponent) {
	t.Helper()
	// Not replay mode: no replay guard can conceal a correction attempt.
	p := player.New(slog.New(slog.NewTextHandler(io.Discard, nil)), player.MonitoringState{CurrentTime: time.Now()}, nil)
	p.SetRecoverFunc(func(_ *player.Player, problem any) { t.Errorf("packet processing recovered: %v", problem) })
	Register(p)
	p.Version = protocol.CurrentProtocol
	p.GameMode = packet.GameTypeSurvival
	p.Ready, p.Alive = true, true
	p.RuntimeId = 1
	m := p.Movement().(*AuthoritativeMovementComponent)
	m.ResetTransferState(mgl32.Vec3{2, 64, 2})
	m.SetSize(mgl32.Vec3{.6, 1.8, 1})
	m.SetMovementSpeed(.1)
	m.SetDefaultMovementSpeed(.1)
	m.SetOnGround(true)
	p.World().AddChunk(protocol.ChunkPos{0, 0}, oworld.ChunkInfo{Chunk: chunk.New(oworld.BlockRegistry, dfworld.Overworld.Range())})
	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			p.World().SetBlock(dfcube.Pos{x, 63, z}, block.Stone{}, nil)
		}
	}
	return p, m
}

func observationInput(t *testing.T, p *player.Player, frame uint64, pos mgl32.Vec3) *packet.PlayerAuthInput {
	t.Helper()
	input := &packet.PlayerAuthInput{Tick: frame, Position: pos.Add(mgl32.Vec3{0, 1.62001, 0}), InputData: protocol.NewInputFlags(packet.InputFlagCount), InputMode: packet.InputModeMouse}
	var pk packet.Packet = input
	ctx := packetcontext.NewHandlePacketContext(&pk)
	before := *input
	p.HandleClientPacket(ctx)
	if ctx.Cancelled() {
		t.Error("valid movement was discarded")
	}
	if input.Position != before.Position || input.Delta != before.Delta || input.MoveVector != before.MoveVector {
		t.Error("observation changed forwarded movement")
	}
	return input
}

func TestMovementKeepsOriginalInputAndRunsChecksDuringBurst(t *testing.T) {
	p, _ := observationPlayer(t)
	check := new(observationCheck)
	p.RegisterDetection(check)
	observationInput(t, p, 1, mgl32.Vec3{2, 64, 2})
	input := observationInput(t, p, 2, mgl32.Vec3{2, 75, 2})
	comparison := p.MovementComparison()
	if comparison.ExpectedPosition.Sub(comparison.ObservedPosition).Len() < 1 {
		t.Fatal("prediction disagreement was erased")
	}
	if input.Position.Y() < 76 {
		t.Fatal("client position was replaced with prediction")
	}
	for frame := uint64(3); frame <= 200; frame++ {
		observationInput(t, p, frame, mgl32.Vec3{2, 75, 2})
	}
	if p.InputCount != 200 || check.count != 200 {
		t.Fatalf("processed=%d checks=%d, want 200 each", p.InputCount, check.count)
	}
}

func TestSimulationClampsPrivateImpulseOnly(t *testing.T) {
	p, m := observationPlayer(t)
	observationInput(t, p, 1, mgl32.Vec3{2, 64, 2})
	input := &packet.PlayerAuthInput{Position: mgl32.Vec3{2, 65.62001, 2}, MoveVector: mgl32.Vec2{2, -2}, InputData: protocol.NewInputFlags(packet.InputFlagCount)}
	var pk packet.Packet = input
	p.HandleClientPacket(packetcontext.NewHandlePacketContext(&pk))
	if input.MoveVector != (mgl32.Vec2{2, -2}) {
		t.Fatal("raw input was normalized before detectors")
	}
	if m.Impulse().X() > .98 || m.Impulse().Y() < -.98 {
		t.Fatalf("private impulse was not bounded: %v", m.Impulse())
	}
}

func TestRealTeleportFenceSurvivesACKWithoutRewritingMovement(t *testing.T) {
	p, m := observationPlayer(t)
	observationInput(t, p, 1, mgl32.Vec3{2, 64, 2})
	destination := mgl32.Vec3{10, 64, 10}
	m.ServerUpdate(&packet.MovePlayer{EntityRuntimeID: 1, Mode: packet.MoveModeTeleport, Position: destination.Add(mgl32.Vec3{0, 1.62001, 0})})
	if p.InteractionPosition().Sub(destination).Len() > .001 {
		t.Fatal("server destination not observed")
	}
	m.Teleport(destination, true, false) // What the genuine teleport ACK applies.
	observationInput(t, p, 2, mgl32.Vec3{2, 64, 2})
	observationInput(t, p, 3, mgl32.Vec3{2, 64, 2})
	if p.InteractionPosition().Sub(destination).Len() > .001 {
		t.Fatal("stale client input undid server teleport fence")
	}
	observationInput(t, p, 4, destination)
	observationInput(t, p, 5, destination.Add(mgl32.Vec3{.2, 0, 0}))
	if p.InteractionPosition().X() < 10.1 {
		t.Fatal("reconciled teleport never resumed observation")
	}
}

func TestMiningGroundContextUsesObservedSupport(t *testing.T) {
	p, m := observationPlayer(t)
	observationInput(t, p, 1, mgl32.Vec3{2, 64, 2})
	m.SetPos(mgl32.Vec3{2, 75, 2})
	m.SetOnGround(false)
	if !p.InteractionOnGround() {
		t.Fatal("grounded observed feet inherited airborne prediction penalty")
	}
	observationInput(t, p, 2, mgl32.Vec3{2, 65, 2})
	if p.InteractionOnGround() {
		t.Fatal("actual rising movement was treated as grounded")
	}
}
