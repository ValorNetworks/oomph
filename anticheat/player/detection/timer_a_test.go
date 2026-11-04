package detection

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/oomph-ac/oomph/anticheat/player"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func timerPlayer(t *testing.T) (*player.Player, *TimerA, *bytes.Buffer) {
	t.Helper()
	output := new(bytes.Buffer)
	p := player.New(slog.New(slog.NewTextHandler(output, nil)), player.MonitoringState{
		IsReplay: true, CurrentTime: time.Unix(1000, 0),
	}, nil)
	p.Ready, p.Alive = true, true
	return p, New_TimerA(p), output
}

func timerInput(p *player.Player, d *TimerA, after time.Duration) {
	p.SetTime(p.Time().Add(after))
	d.Detect(&packet.PlayerAuthInput{})
}

func TestTimerAcceptsNormalSchedulingJitterAndHighPingBatches(t *testing.T) {
	p, d, _ := timerPlayer(t)
	p.StackLatency = 400 * time.Millisecond
	for range 100 {
		timerInput(p, d, 49*time.Millisecond)
		timerInput(p, d, 51*time.Millisecond)
	}
	for range 100 {
		timerInput(p, d, 200*time.Millisecond)
		for range 3 {
			timerInput(p, d, 0)
		}
	}
	if d.Metadata().Violations != 0 {
		t.Fatal("normal paced input or batching flagged")
	}
}

func TestTimerFlagsFastInputThroughNormalConsolePathAndRecovers(t *testing.T) {
	p, d, output := timerPlayer(t)
	for range 240 {
		timerInput(p, d, 25*time.Millisecond)
	}
	if d.Metadata().Violations == 0 || !strings.Contains(output.String(), "failed detection") || !strings.Contains(output.String(), "dtc=Timer") {
		t.Fatalf("fast input did not generate ordinary flags: %s", output)
	}
	violations := d.Metadata().Violations
	for range 100 {
		timerInput(p, d, 50*time.Millisecond)
	}
	if d.Metadata().Violations != violations {
		t.Fatal("normal input inherited permanent timer debt")
	}
	if p.Closed || d.Punishable() {
		t.Fatal("detection-only timer can disconnect a player")
	}
}

func TestTimerAllowsStallCatchupButExpiresUnusedCredit(t *testing.T) {
	p, d, _ := timerPlayer(t)
	for range 20 {
		timerInput(p, d, 50*time.Millisecond)
	}
	timerInput(p, d, 5*time.Second)
	for range 99 {
		timerInput(p, d, 0)
	}
	if d.Metadata().Violations != 0 {
		t.Fatal("legitimate queued inputs after a stall flagged")
	}
	// An unused pause cannot bank an unlimited future speed advantage.
	timerInput(p, d, 10*time.Second)
	for range 300 {
		timerInput(p, d, 25*time.Millisecond)
	}
	if d.Metadata().Violations == 0 {
		t.Fatal("unused catch-up allowance never expired")
	}
	d.ResetBackend()
	if !d.last.IsZero() || d.credit != 0 || d.metadata.Buffer != 0 {
		t.Fatal("backend transfer retained input debt")
	}
}
