package player

import (
	"math"
	"testing"

	"github.com/df-mc/dragonfly/server/block"
	dfcube "github.com/df-mc/dragonfly/server/block/cube"
	dfworld "github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/ethaniccc/float32-cube/cube"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/oomph-ac/oomph/anticheat/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestKnockbackCorrectionThreshold(t *testing.T) {
	for age, want := range map[uint64]float32{0: 10, 60: 10, 110: 5.25, 160: 0.5, 161: 0.5, math.MaxUint64: 0.5} {
		if got := correctionThreshold(0.5, 10, 60, 100, age); got != want {
			t.Errorf("threshold after %d ticks = %v, want %v", age, got, want)
		}
	}
	delta := mgl32.Vec3{1, -1, 1}
	if pull, ok := knockbackTaperPull(delta, 0.1, 60, 100, 61); !ok || pull != (mgl32.Vec3{0.06, -0.04, 0.06}) {
		t.Errorf("taper pull = %v, enabled = %v", pull, ok)
	}
	if _, ok := knockbackTaperPull(delta, 0.2, 60, 100, 61); ok {
		t.Error("taper pull allowed despite excessive velocity drift")
	}
}

func TestVisiblePositionStaysNearPrediction(t *testing.T) {
	prediction := mgl32.Vec3{0, 2, 0}
	for _, tc := range []struct {
		clientX, tolerance, wantX float32
	}{
		{0.1, 0.15, 0.1},
		{0.3, 0.15, 0.15},
		{1, 0.15, 0.4},
		{1, 2, 0.4},
		{0.1, -1, 0},
	} {
		got := visiblePosition(prediction, mgl32.Vec3{tc.clientX, 2, 0}, tc.tolerance)
		if !got.ApproxEqual(mgl32.Vec3{tc.wantX, 2, 0}) {
			t.Errorf("client x %v with tolerance %v = %v, want %v", tc.clientX, tc.tolerance, got.X(), tc.wantX)
		}
	}
}

func TestPositionAdjustmentRequiresKnownTerrainAndStopsAtWalls(t *testing.T) {
	world.BlockRegistry.Finalize()
	terrain := world.New(nil)
	column := chunk.New(world.BlockRegistry, dfworld.Overworld.Range())
	columnPos := protocol.ChunkPos{0, 0}
	terrain.AddPartialChunk(columnPos, column)
	from := mgl32.Vec3{2.5, 68, 2.5}
	target := mgl32.Vec3{2.9, 68, 2.5}
	box := cube.Box(2.2001, 68, 2.2001, 2.7999, 69.8, 2.7999)
	if got := safePositionAdjustment(box, from, target, terrain); got != from {
		t.Fatalf("unreceived subchunk allowed adjustment to %v", got)
	}
	terrain.MarkSubChunkKnown(columnPos, byte(column.SubIndex(68)))
	if got := safePositionAdjustment(box, from, target, terrain); !got.ApproxEqual(target) {
		t.Fatalf("clear space stopped adjustment at %v", got)
	}
	terrain.SetBlock(dfcube.Pos{3, 68, 2}, block.Stone{}, nil)
	if got := safePositionAdjustment(box, from, target, terrain); got.X() < 2.69 || got.X() > 2.7 {
		t.Fatalf("wall adjustment stopped at %v, want x near 2.7", got)
	}
}
