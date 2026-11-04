package player

import (
	"github.com/ethaniccc/float32-cube/cube"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// MovementComponent is an interface for which movement information is stored in the player.
// It is used for updating movement states of the player, and providing them to any other
// component that requires it.
type MovementComponent interface {
	// Client returns the non-authoritative client movement sent to the server.
	Client() NonAuthoritativeMovementInfo

	// Pos returns the position of the movement component.
	Pos() mgl32.Vec3
	// LastPos returns the previous position of the movement component.
	LastPos() mgl32.Vec3
	// SetPos sets the position of the movement component.
	SetPos(pos mgl32.Vec3)

	// SlideOffset returns the slide offset of the player.
	SlideOffset() mgl32.Vec2
	// SetSlideOffset sets the slide offset of the player.
	SetSlideOffset(mgl32.Vec2)

	// Vel returns the velocity of the movement component.
	Vel() mgl32.Vec3
	// LastVel returns the previous velocity of the movement component.
	LastVel() mgl32.Vec3
	// SetVel returns the velocity of the movement component.
	SetVel(vel mgl32.Vec3)

	// Mov returns the velocity of the movement component before friction and
	// gravity are applied to it.
	Mov() mgl32.Vec3
	// LastMov returns the previous processed velocity before friction and gravity
	// of the movement component.
	LastMov() mgl32.Vec3
	// SetMov sets the velocity of the movement component before friction and gravity.
	SetMov(mov mgl32.Vec3)

	// Rotation returns the rotation of the movement component. The X-axis contains
	// the pitch, the Y-axis contains the head-yaw, and the Z-axis contains the yaw.
	Rotation() mgl32.Vec3
	// LastRotation returns the previous rotation of the movement component.
	LastRotation() mgl32.Vec3
	// SetRotation sets the current rotation of the movement component.
	SetRotation(rot mgl32.Vec3)
	// RotationDelta returns the difference from the current and previous rotations of
	// the movement component.
	RotationDelta() mgl32.Vec3

	// SupportingBlockPos is the position of the block that the player is standing on/supported by.
	SupportingBlockPos() *cube.Pos
	// SetSupportingBlockPos sets the position of the block that the player is standing on/supported by.
	SetSupportingBlockPos(pos *cube.Pos)

	// Impulse returns the movement impulse of the movement component. The X-axis contains
	// the forward impulse, and the Y-axis contains the left impulse.
	Impulse() mgl32.Vec2

	// Sprinting returns true if the movement component is sprinting.
	Sprinting() bool
	// SetSprinting sets whether the movement component is sprinting.
	SetSprinting(sprint bool)
	// ServerSprint returns true if the movement component is sprinting according to the server.
	ServerSprint() bool
	// SetServerSprint sets whether the movement component is sprinting according to the server.
	SetServerSprint(sprint bool)
	// PressingSprint returns whether the movement component is holding down the key bound to the sprint action.
	PressingSprint() bool

	// Jumping returns true if the movement component is expecting a jump in the current frame.
	Jumping() bool
	// PressingJump returns true if the movement component is holding down the key bound to the jump action.
	PressingJump() bool
	// JumpDelay returns the number of ticks until the movement component can make another jump.
	JumpDelay() uint64
	// SetJumpDelay sets the number of ticks until the movement component can make another jump.
	SetJumpDelay(ticks uint64)

	// Sneaking returns true if the movement component is currently sneaking.
	Sneaking() bool
	// PressingSneak returns true if the movement component is holding down the key bound to the sneak action.
	PressingSneak() bool
	// SetPressingSneak sets if the movement component is holding down the key bound o the sneak action.
	SetPressingSneak(pressing bool)

	// PenetratedLastFrame returns true if the movement component had penetrated through a block in
	// the previous simulation frame.
	PenetratedLastFrame() bool
	// SetPenetratedLastFrame sets whether the movement component had penetrated through a block
	// in the previous simulation frame.
	SetPenetratedLastFrame(penetrated bool)
	// StuckInCollider returns true if the movement component is stuck in a block that does
	// not support one-way collisions.
	StuckInCollider() bool
	// SetStuckInCollider sets whether the movement component is stuck in a block that does
	// not support one-way collisions.
	SetStuckInCollider(stuck bool)

	// Knockback returns the knockback sent by the server to the movement component.
	Knockback() mgl32.Vec3
	// SetKnockback notifies the movement component of knockback sent by the server.
	SetKnockback(vel mgl32.Vec3)
	// HasKnockback returns true if the movement component needs knockback applied on the next simulation.
	HasKnockback() bool

	// Teleport notifies the movement component of a teleport.
	Teleport(pos mgl32.Vec3, onGround bool, smoothed bool)
	// TeleportPos returns the teleport position sent to the movement component.
	TeleportPos() mgl32.Vec3
	// HasTeleport returns true if the movement component needs a teleport applied on the next simulation.
	HasTeleport() bool
	// TeleportSmoothed returns true if the movement component has a teleport that needs to be smoothed out.
	TeleportSmoothed() bool
	// SetPendingTeleportPos
	SetPendingTeleportPos(mgl32.Vec3)
	// PendingTeleportPos
	PendingTeleportPos() mgl32.Vec3
	// AddPendingTeleport
	AddPendingTeleport()
	// RemovePendingTeleport
	RemovePendingTeleport()
	// PendingTeleports
	PendingTeleports() int
	// RemainingTeleportTicks returns the amount of ticks the teleport still needs to be completed.
	RemainingTeleportTicks() int
	// TicksSinceTeleport returns the amount of ticks since the last teleport was applied.
	TicksSinceTeleport() uint64

	// Size returns the width, height, and scale of the movement component in a Vec2. The X-axis
	// contains the width, the Y-axis contains the height, and the Z-axis contains the scale.
	Size() mgl32.Vec3
	// SetSize sets the size of the movement component.
	SetSize(size mgl32.Vec3)
	// BoundingBox returns the bounding box of the movement component translated to
	// it's current position.
	BoundingBox() cube.BBox
	// ClientBoundingBox returns the bounding box of the movement component translated to
	// the client's current position.
	ClientBoundingBox() cube.BBox

	// Gravity returns the gravity of the movement component.
	Gravity() float32
	// SetGravity sets the gravity of the movement component.
	SetGravity(gravity float32)

	// HasGravity returns true if the movement component is affected by gravity.
	HasGravity() bool
	// SetHasGravity sets if the movement component is affected by gravity.
	SetHasGravity(bool)

	// FallDistance returns the fall distance of the movement component.
	FallDistance() float32
	// SetFallDistance sets the fall distance of the movement component.
	SetFallDistance(fallDistance float32)

	// MovementSpeed returns the movement speed of the movement component.
	MovementSpeed() float32
	// SetMovementSpeed sets the movement speed of the movement component.
	SetMovementSpeed(speed float32)
	// DefaultMovementSpeed return the movement speed the client should default to.
	DefaultMovementSpeed() float32
	// SetDefaultMovementSpeed sets the movement speed the client should default to.
	SetDefaultMovementSpeed(speed float32)

	// AirSpeed returns the movement speed of the movement component while off ground.
	AirSpeed() float32
	// SetAirSpeed sets the movement speed of the movement component while off ground.
	SetAirSpeed(airSpeed float32)

	// JumpHeight returns the jump height of the movement component.
	JumpHeight() float32
	// SetJumpHeight sets the jump height of the movement component.
	SetJumpHeight(height float32)

	// XCollision returns true if the movement component is collided with a block
	// on the x-axis.
	XCollision() bool
	// YCollision returns true if the movement component is collided with a block
	// on the y-axis.
	YCollision() bool
	// ZCollision returns true if the movement component is collided with a block
	// on the z-axis.
	ZCollision() bool
	// SetCollisions sets whether the movement component is colliding with a block
	// on the x, y, or z axies.
	SetCollisions(x, y, z bool)

	// OnGround returns true if the movement component is on the ground.
	OnGround() bool
	// SetOnGround sets whether the movement component is on the ground.
	SetOnGround(onGround bool)

	// Immobile returns true if the movement component is immobile.
	Immobile() bool
	// SetImmobile sets whether the movement component is immobile.
	SetImmobile(immobile bool)

	// NoClip returns true if the movement component has no collisions.
	NoClip() bool
	// SetNoClip sets whether the movement component has no collisions.
	SetNoClip(noClip bool)

	// Gliding returns true if the movement component is gliding.
	Gliding() bool
	// SetGliding sets whether movement component has no collisions.
	SetGliding(gliding bool)
	// GlideBoost returns the amount of ticks the movement component has a gliding boost for.
	GlideBoost() (boostTicks int64)
	// SetGlideBoost sets the amount of ticks the movement component should apply a gliding boost for.
	SetGlideBoost(boostTicks int64)

	// Flying returns true if the movement component is currently flying.
	Flying() bool
	// SetFlying sets whether the movement component is flying.
	SetFlying(fly bool)
	// MayFly returns true if the movement component has the permission to fly.
	MayFly() bool
	// SetMayFly sets whether the movement component has the permission to fly.
	SetMayFly(mayFly bool)
	// TrustFlyStatus returns whether the movement component can trust the fly status sent by the client.
	TrustFlyStatus() bool
	// SetTrustFlyStatus sets whether the movement component can trust the fly status sent by the client.
	SetTrustFlyStatus(bool)
	// JustDisabledFlight returns true if the movement component just disabled flight.
	JustDisabledFlight() bool

	// Update updates the states of the movement component from the given input.
	Update(input *packet.PlayerAuthInput) bool
	// ServerUpdate updates certain states of the movement component based on a packet sent by the remote server.
	ServerUpdate(pk packet.Packet)

	// Reset is a function that resets the current movement of the movement component to the client's non-authoritative movement.
	Reset()
	// ResetTransferState clears movement information belonging to an old backend.
	ResetTransferState(pos mgl32.Vec3)
}

// NonAuthoritativeMovementInfo represents movement information that the player has sent to the server but has not validated/verified.
type NonAuthoritativeMovementInfo interface {
	Pos() mgl32.Vec3
	LastPos() mgl32.Vec3

	Vel() mgl32.Vec3
	LastVel() mgl32.Vec3

	Mov() mgl32.Vec3
	LastMov() mgl32.Vec3

	// ToggledFly returns whether the client has attempted to trigger a fly action.
	ToggledFly() bool
	// SetToggledFly sets whether the client has attempted to trigger a fly action.
	SetToggledFly(bool)

	HorizontalCollision() bool
	VerticalCollision() bool
}

func (p *Player) SetMovement(c MovementComponent) {
	p.movement = c
}

func (p *Player) Movement() MovementComponent {
	return p.movement
}

// handleMovement updates internal prediction without changing the client's input.
func (p *Player) handleMovement(pk *packet.PlayerAuthInput) {
	// Capture the old correction exclusions before simulation consumes impulses.
	exempt := !p.Ready || !p.Alive || p.movement.Immobile() ||
		p.movement.HasTeleport() || p.movement.PendingTeleports() != 0 ||
		p.interaction.teleportPending || p.movement.HasKnockback() ||
		pk.InputData.Load(packet.InputFlagJumpPressedRaw)
	p.SimulationFrame = pk.Tick
	p.ClientTick++
	p.InputCount++
	pos := p.InteractionPosition()
	p.World().CleanChunks(p.WorldUpdater().ChunkRadius(), interactionChunk(pos))
	p.effects.Tick()
	simulated := p.movement.Update(pk)
	p.interaction.comparison = MovementComparison{
		Frame: pk.Tick, ExpectedPosition: p.movement.Pos(), ExpectedVelocity: p.movement.Vel(),
		ObservedPosition: p.movement.Client().Pos(), ObservedVelocity: p.movement.Client().Vel(),
		Eligible: simulated && !exempt,
	}
}
