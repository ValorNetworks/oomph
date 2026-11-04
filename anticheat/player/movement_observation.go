package player

import (
	"math"

	"github.com/ethaniccc/float32-cube/cube"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/oomph-ac/oomph/anticheat/game"
	"github.com/oomph-ac/oomph/anticheat/utils"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// movementObservation belongs to the serialized packet-processing path. It is
// independent of prediction: predictions never replace forwarded client input.
type movementObservation struct {
	position, previous        mgl32.Vec3
	initialized               bool
	teleportPending           bool
	teleportPosition          mgl32.Vec3
	comparison                MovementComparison
	predictionRebaseRequested bool
}

// MovementComparison preserves prediction before any internal reconciliation.
// It is evidence, not a legality verdict.
type MovementComparison struct {
	Frame                              uint64
	ExpectedPosition, ExpectedVelocity mgl32.Vec3
	ObservedPosition, ObservedVelocity mgl32.Vec3
	Eligible                           bool
}

// MovementComparison returns a value snapshot. Callers use the same processing
// ownership as Movement(); it is not an asynchronous player-state API.
func (p *Player) MovementComparison() MovementComparison { return p.interaction.comparison }

// RequestPredictionRebase asks the serialized packet path to realign internal
// position after all checks see this frame. It sends no packet and retains
// expected velocity, server impulses, abilities and acknowledgement state.
func (p *Player) RequestPredictionRebase(frame uint64) {
	if comparison := p.interaction.comparison; comparison.Eligible && comparison.Frame == frame {
		p.interaction.predictionRebaseRequested = true
	}
}

func (p *Player) finishMovementPrediction() {
	if !p.interaction.predictionRebaseRequested {
		return
	}
	p.interaction.predictionRebaseRequested = false
	if !p.interaction.comparison.Eligible || p.interaction.teleportPending ||
		p.movement.HasTeleport() || p.movement.PendingTeleports() != 0 ||
		p.movement.HasKnockback() || p.movement.Immobile() {
		return
	}
	position := p.InteractionPosition()
	p.movement.SetPos(position)
	p.movement.SetSupportingBlockPos(nil)
	if grounded, known := p.InteractionGroundedAt(position); known && !p.movement.Flying() {
		p.movement.SetOnGround(grounded)
	}
}

func (p *Player) resetInteractionMovement(pos mgl32.Vec3) {
	p.interaction = movementObservation{position: pos, previous: pos, initialized: true}
}

// ObserveServerTeleport records a real outgoing server move under the packet
// processing lock. It never sends a packet. The fence survives the stack ACK
// until client input reaches the destination, matching native DF reconciliation.
func (p *Player) ObserveServerTeleport(pos mgl32.Vec3) {
	if !finiteVector(pos) {
		return
	}
	p.interaction.position, p.interaction.previous = pos, pos
	p.interaction.initialized = true
	p.interaction.teleportPending, p.interaction.teleportPosition = true, pos
}

func (p *Player) observeInteractionMovement(input *packet.PlayerAuthInput) bool {
	if !finiteVector(input.Position) || !finiteVector(input.Delta) ||
		!finiteVector(mgl32.Vec3{input.Pitch, input.HeadYaw, input.Yaw}) ||
		!finiteVector(mgl32.Vec3{input.MoveVector.X(), input.MoveVector.Y(), 0}) {
		return false
	}
	position := input.Position.Sub(mgl32.Vec3{0, game.DefaultPlayerHeightOffset, 0})
	state := &p.interaction
	if !state.initialized {
		p.resetInteractionMovement(p.movement.Pos())
	}
	if state.teleportPending {
		if position.Sub(state.teleportPosition).Len() > 1 {
			return true
		}
		state.teleportPending = false
	}
	if p.movement.Immobile() {
		return true
	}
	state.previous, state.position = state.position, position
	return true
}

// InteractionPosition is the observed backend-input position, fenced by actual
// server moves/immobility. It does not assert that DF accepted every core event.
func (p *Player) HasInteractionObservation() bool { return p.interaction.initialized }

func (p *Player) InteractionPosition() mgl32.Vec3 {
	if p.interaction.initialized {
		return p.interaction.position
	}
	return p.movement.Client().Pos()
}

func (p *Player) LastInteractionPosition() mgl32.Vec3 {
	if p.interaction.initialized {
		return p.interaction.previous
	}
	return p.movement.Client().LastPos()
}

// InteractionBoundingBox retains the movement component's dimension/pose
// convention, but places the box at the interaction rather than predicted feet.
func (p *Player) InteractionBoundingBox() cube.BBox {
	return p.movement.BoundingBox().Translate(p.InteractionPosition().Sub(p.movement.Pos()))
}

// InteractionOnGround tests support at the observed location for mining context.
// It never copies the client's collision flag into authoritative prediction.
func (p *Player) InteractionOnGround() bool {
	if p.movement.Flying() || p.InteractionPosition().Y()-p.LastInteractionPosition().Y() > 1e-5 {
		return false
	}
	grounded, known := p.InteractionGroundedAt(p.InteractionPosition())
	return grounded || !known
}

// InteractionGroundedAt returns support and whether the needed columns exist.
// Prediction uses previous-frame support, while mining uses the current frame.
func (p *Player) InteractionGroundedAt(position mgl32.Vec3) (bool, bool) {
	box := p.movement.BoundingBox().Translate(position.Sub(p.movement.Pos()))
	min, max := box.Min(), box.Max()
	support := cube.Box(min.X(), min.Y()-0.001, min.Z(), max.X(), min.Y(), max.Z())
	// Unknown collision data cannot establish an airborne mining penalty.
	if p.World().Chunk(interactionChunk(min)) == nil || p.World().Chunk(interactionChunk(max)) == nil {
		return false, false
	}
	return utils.HasNearbyBBoxes(support, p.World()), true
}

func finiteVector(v mgl32.Vec3) bool {
	for _, value := range v {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return false
		}
	}
	return true
}

func interactionChunk(pos mgl32.Vec3) protocol.ChunkPos {
	return protocol.ChunkPos{int32(math.Floor(float64(pos.X()))) >> 4, int32(math.Floor(float64(pos.Z()))) >> 4}
}
