package cd

import "testing"

// gitRepo builds the part of a Fleet GitRepo object the adapter reads.
func gitRepo(commit string, summary map[string]any, conditions ...map[string]any) map[string]any {
	conds := make([]any, 0, len(conditions))
	for _, c := range conditions {
		conds = append(conds, c)
	}
	return map[string]any{"status": map[string]any{
		"commit": commit, "summary": summary, "conditions": conds,
	}}
}

func ready(status string, message string) map[string]any {
	return map[string]any{"type": "Ready", "status": status, "message": message}
}

// The CD abstraction only knows these values; nothing from Fleet's own
// vocabulary may appear in a Status.
var neutralSync = map[string]bool{"SYNCED": true, "SYNCING": true, "OUT_OF_SYNC": true, "UNKNOWN": true}
var neutralHealth = map[string]bool{"HEALTHY": true, "PROGRESSING": true, "DEGRADED": true, "MISSING": true, "UNKNOWN": true}

func TestFleetStatus(t *testing.T) {
	for _, tc := range []struct {
		name         string
		obj          map[string]any
		sync, health string
		revision     string
	}{
		{
			name:   "applied everywhere",
			obj:    gitRepo("abc123", map[string]any{"ready": int64(1), "desiredReady": int64(1)}, ready("True", "")),
			sync:   "SYNCED",
			health: "HEALTHY", revision: "abc123",
		},
		{
			name:   "still rolling out",
			obj:    gitRepo("abc123", map[string]any{"ready": int64(0), "desiredReady": int64(1), "notReady": int64(1)}, ready("False", "waiting")),
			sync:   "SYNCING",
			health: "PROGRESSING", revision: "abc123",
		},
		{
			name:   "apply failed",
			obj:    gitRepo("abc123", map[string]any{"ready": int64(0), "desiredReady": int64(1), "errApplied": int64(1)}, ready("False", "boom")),
			sync:   "OUT_OF_SYNC",
			health: "DEGRADED", revision: "abc123",
		},
		{
			name:   "changed outside git",
			obj:    gitRepo("abc123", map[string]any{"ready": int64(1), "desiredReady": int64(1), "modified": int64(1)}, ready("False", "drift")),
			sync:   "OUT_OF_SYNC",
			health: "PROGRESSING", revision: "abc123",
		},
		{
			name:   "nothing reported yet",
			obj:    gitRepo("", map[string]any{}),
			sync:   "SYNCING",
			health: "PROGRESSING", revision: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := fleetStatus(tc.obj)
			if st.Sync != tc.sync || st.Health != tc.health {
				t.Errorf("got %s/%s, want %s/%s", st.Sync, st.Health, tc.sync, tc.health)
			}
			if st.Revision != tc.revision {
				t.Errorf("revision %q, want %q", st.Revision, tc.revision)
			}
			if !neutralSync[st.Sync] || !neutralHealth[st.Health] {
				t.Errorf("status %s/%s is not part of the neutral vocabulary", st.Sync, st.Health)
			}
		})
	}
}

// A message from Fleet is passed through for the operator to read, but it never
// decides the status.
func TestFleetStatusKeepsMessage(t *testing.T) {
	st := fleetStatus(gitRepo("abc123", map[string]any{"ready": int64(0), "desiredReady": int64(1), "errApplied": int64(1)},
		ready("False", "failed to apply manifests.yaml")))
	if st.Message != "failed to apply manifests.yaml" {
		t.Errorf("message %q", st.Message)
	}
}
