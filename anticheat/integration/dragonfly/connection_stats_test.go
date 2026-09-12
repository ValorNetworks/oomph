package dragonfly

import (
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/session"
	"github.com/sandertv/gophertunnel/minecraft"
)

type statsConnection struct {
	session.Conn
	value minecraft.ConnectionStats
}

func (s statsConnection) ConnectionStats() (minecraft.ConnectionStats, bool) { return s.value, true }

func TestSessionConnectionForwardsOptionalStatistics(t *testing.T) {
	if _, ok := (&sessionConn{}).ConnectionStats(); ok {
		t.Fatal("unsupported connection has statistics")
	}
	want := minecraft.ConnectionStats{Count: 8, UpdatedAt: time.Now(), OriginalTransmissions: 9, Retransmissions: 1}
	want.RTT[0] = 12345 * time.Microsecond
	conn := &sessionConn{Conn: statsConnection{value: want}}
	got, ok := conn.ConnectionStats()
	if !ok || got != want {
		t.Fatalf("got=%+v ok=%v", got, ok)
	}
}
