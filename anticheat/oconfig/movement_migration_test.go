package oconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMovementRemovalMigrationActivatesPreservedSettings(t *testing.T) {
	saved := Global
	t.Cleanup(func() { Global = saved })
	path := filepath.Join(t.TempDir(), "oomph.hjson")
	input := `{"version":7, "prefix":"kept", "combat_opts":{"maximum_reach":3.1}, "network_opts":{"max_ack_timeout":42}, "movement_opts":{"correction_threshold":0.5, "persuasion_threshold":0.01, "accept_client_position":true, "position_acception_threshold":0.1, "accept_client_velocity":true, "velocity_acception_threshold":0.1, "limit_all_velocity_threshold":14, "wrapper_owned":"keep"}, "wrapper_root":"preserved"}`
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ParseJSON(path); !errors.Is(err, ErrConfigUpdated) {
		t.Fatalf("migration status=%v", err)
	}
	if Global.Version != 8 || Global.Prefix != "kept" || Global.Combat.MaximumReach != 3.1 || Global.Network.MaxACKTimeout != 42 || Global.Movement.LimitAllVelocityThreshold != 14 {
		t.Fatalf("first load lost effective settings: %#v", Global)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"correction_threshold", "persuasion_threshold", "accept_client_position", "position_acception_threshold", "accept_client_velocity", "velocity_acception_threshold"} {
		if strings.Contains(string(body), key) {
			t.Fatalf("obsolete option survived: %s", key)
		}
	}
	if !strings.Contains(string(body), "wrapper_owned") || !strings.Contains(string(body), "wrapper_root") {
		t.Fatal("migration deleted unrelated fields")
	}
	if err := ParseJSON(path); err != nil {
		t.Fatalf("second load=%v", err)
	}
}
