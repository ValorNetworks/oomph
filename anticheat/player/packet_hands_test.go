package player

import (
	"testing"

	"github.com/oomph-ac/oomph/anticheat/player/context"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type handTestInventory struct {
	InventoryComponent
	slot int32
}

func (i *handTestInventory) SetHeldSlot(slot int32) { i.slot = slot }

type handTestWorldUpdater struct{ WorldUpdaterComponent }

func (*handTestWorldUpdater) ValidateInteraction(*packet.InventoryTransaction) bool { return true }
func (*handTestWorldUpdater) AttemptItemInteractionWithBlock(*packet.InventoryTransaction) bool {
	return true
}

type handTestCombat struct {
	CombatComponent
	attacks int
}

func (c *handTestCombat) Attack(*packet.InventoryTransaction) { c.attacks++ }

type handTestClicks struct {
	ClicksComponent
	attacks int
}

func (c *handTestClicks) HandleAttack(*protocol.UseItemOnEntityTransactionData) { c.attacks++ }

func TestNonMainHandReleaseDoesNotResetConsumption(t *testing.T) {
	for _, tc := range []struct {
		name string
		hand byte
	}{
		{"offhand", protocol.HandSlotOffHand}, {"invalid", 255},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inv := &handTestInventory{slot: 2}
			p := &Player{inventory: inv, worldUpdater: &handTestWorldUpdater{}, StartUseConsumableTick: 40}
			var pk packet.Packet = &packet.InventoryTransaction{TransactionData: &protocol.ReleaseItemTransactionData{
				Hand: tc.hand, HotBarSlot: 0,
			}}
			ctx := context.NewHandlePacketContext(&pk)
			p.HandleClientPacket(ctx)
			if p.StartUseConsumableTick != 40 || inv.slot != 2 {
				t.Fatalf("non-main-hand release changed main-hand state: consumption start=%d, held slot=%d", p.StartUseConsumableTick, inv.slot)
			}
			if ctx.Cancelled() {
				t.Fatal("backend must receive the packet to resync or reject its hand")
			}
		})
	}
}

func TestNonMainHandEntityAttackDoesNotQueueCombat(t *testing.T) {
	for _, tc := range []struct {
		name string
		hand byte
	}{
		{"offhand", protocol.HandSlotOffHand}, {"invalid", 255},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inv, combat, clicks := &handTestInventory{slot: 2}, &handTestCombat{}, &handTestClicks{}
			p := &Player{inventory: inv, combat: combat, clicks: clicks, worldUpdater: &handTestWorldUpdater{}, opts: &Opts{}, GameMode: packet.GameTypeSurvival}
			var pk packet.Packet = &packet.InventoryTransaction{TransactionData: &protocol.UseItemOnEntityTransactionData{
				Hand: tc.hand, HotBarSlot: 0, ActionType: protocol.UseItemOnEntityActionAttack,
			}}
			ctx := context.NewHandlePacketContext(&pk)
			p.HandleClientPacket(ctx)
			if inv.slot != 2 || combat.attacks != 0 || clicks.attacks != 0 {
				t.Fatalf("non-main-hand interaction treated as main-hand attack: slot=%d, combat=%d, clicks=%d", inv.slot, combat.attacks, clicks.attacks)
			}
			if ctx.Cancelled() {
				t.Fatal("backend must receive the packet to resync or reject its hand")
			}
		})
	}
}

func TestMainHandStillReleasesAndAttacks(t *testing.T) {
	inv, combat, clicks := &handTestInventory{slot: 2}, &handTestCombat{}, &handTestClicks{}
	p := &Player{inventory: inv, combat: combat, clicks: clicks, worldUpdater: &handTestWorldUpdater{}, opts: &Opts{}, GameMode: packet.GameTypeSurvival, StartUseConsumableTick: 40}
	var pk packet.Packet = &packet.InventoryTransaction{TransactionData: &protocol.ReleaseItemTransactionData{
		Hand: protocol.HandSlotMainHand, HotBarSlot: 3,
	}}
	p.HandleClientPacket(context.NewHandlePacketContext(&pk))
	if inv.slot != 3 || p.StartUseConsumableTick != 0 {
		t.Fatalf("main-hand release ignored: slot=%d, consumption start=%d", inv.slot, p.StartUseConsumableTick)
	}
	pk = &packet.InventoryTransaction{TransactionData: &protocol.UseItemOnEntityTransactionData{
		Hand: protocol.HandSlotMainHand, HotBarSlot: 4, ActionType: protocol.UseItemOnEntityActionAttack,
	}}
	ctx := context.NewHandlePacketContext(&pk)
	p.HandleClientPacket(ctx)
	if inv.slot != 4 || combat.attacks != 1 || clicks.attacks != 1 || !ctx.Cancelled() {
		t.Fatalf("main-hand combat changed: slot=%d, combat=%d, clicks=%d, queued=%t", inv.slot, combat.attacks, clicks.attacks, ctx.Cancelled())
	}
}
