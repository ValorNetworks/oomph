package player

import (
	"math"

	"github.com/chewxy/math32"
	"github.com/ethaniccc/float32-cube/cube"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/oomph-ac/oomph/anticheat/game"
	"github.com/oomph-ac/oomph/anticheat/utils"
	"github.com/oomph-ac/oomph/anticheat/world"
)

func correctionThreshold(base, knockback float32, grace, taper int, sinceKnockback uint64) float32 {
	if grace <= 0 || knockback <= base {
		return base
	}
	if sinceKnockback <= uint64(grace) {
		return knockback
	}
	afterGrace := sinceKnockback - uint64(grace)
	if taper <= 0 || afterGrace >= uint64(taper) {
		return base
	}
	return knockback - (knockback-base)*float32(afterGrace)/float32(taper)
}

// knockbackTaperPull eases the prediction toward the client after knockback
// when the two agree on velocity.
func knockbackTaperPull(positionDelta mgl32.Vec3, velocityDrift float32, grace, taper int, sinceKnockback uint64) (mgl32.Vec3, bool) {
	const (
		maxVelocityDrift  = 0.15
		maxHorizontalStep = 0.06
		maxVerticalStep   = 0.04
	)
	if grace <= 0 || taper <= 0 || sinceKnockback <= uint64(grace) ||
		sinceKnockback-uint64(grace) > uint64(taper) || velocityDrift > maxVelocityDrift {
		return mgl32.Vec3{}, false
	}
	return mgl32.Vec3{
		game.ClampFloat(positionDelta[0], -maxHorizontalStep, maxHorizontalStep),
		game.ClampFloat(positionDelta[1], -maxVerticalStep, maxVerticalStep),
		game.ClampFloat(positionDelta[2], -maxHorizontalStep, maxHorizontalStep),
	}, true
}

// visiblePosition chooses the outgoing location without changing the prediction.
func visiblePosition(predicted, client mgl32.Vec3, tolerance float32) mgl32.Vec3 {
	const maxOffset float32 = 0.4
	if tolerance < 0 {
		return predicted
	}
	if tolerance == 0 {
		tolerance = 0.15
	}
	delta := client.Sub(predicted)
	distance := delta.Len()
	if distance <= min(tolerance, maxOffset) {
		return client
	}
	offset := maxOffset
	if distance > tolerance {
		offset = min(distance-tolerance, maxOffset)
	}
	return predicted.Add(delta.Mul(offset / distance))
}

// safePositionAdjustment clips a straight-line offset against the world. If
// the starting box overlaps a block, simulation handles the separation.
func safePositionAdjustment(box cube.BBox, from, target mgl32.Vec3, w *world.World) mgl32.Vec3 {
	delta := target.Sub(from)
	for _, axis := range delta {
		if math.IsNaN(float64(axis)) || math.IsInf(float64(axis), 0) {
			return from
		}
	}
	distance := delta.Len()
	if distance < 1e-7 {
		return from
	}
	// Restore the full hitbox for the position sent to Dragonfly.
	box = box.GrowVec3(mgl32.Vec3{1e-4, 0, 1e-4})
	sweep := box.Extend(delta)
	if !w.Known(sweep.Grow(1)) {
		return from
	}
	obstacles := utils.NearbyBBoxes(sweep, w)
	limit := float32(1)
	for _, obstacle := range obstacles {
		if obstacle.IntersectsWith(box) {
			return from
		}
		entry, exit := float32(-1e30), float32(1e30)
		for i, speed := range delta {
			if math32.Abs(speed) < 1e-10 {
				if box.Max()[i] <= obstacle.Min()[i] || box.Min()[i] >= obstacle.Max()[i] {
					entry = exit
					break
				}
				continue
			}
			start := (obstacle.Min()[i] - box.Max()[i]) / speed
			end := (obstacle.Max()[i] - box.Min()[i]) / speed
			entry = max(entry, min(start, end))
			exit = min(exit, max(start, end))
		}
		if entry >= 0 && entry < exit && entry <= limit {
			limit = max(0, entry-1e-5/distance)
		}
	}
	dest := from.Add(delta.Mul(limit))
	destBox := box.Translate(dest.Sub(from))
	for _, obstacle := range obstacles {
		if obstacle.IntersectsWith(destBox) {
			return from
		}
	}
	return dest
}
