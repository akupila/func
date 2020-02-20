package cloudformation

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func equalAsJSON(t *testing.T, got interface{}, wantJSON string) {
	t.Helper()

	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}

	equalJSON(t, string(gotJSON), wantJSON)
}

func equalJSON(t *testing.T, got, want string) {
	t.Helper()

	gotMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(got), &gotMap); err != nil {
		t.Fatalf("Unmarshal got: %v", err)
	}

	wantMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(want), &wantMap); err != nil {
		t.Fatalf("Unmarshal want: %v", err)
	}

	if diff := cmp.Diff(gotMap, wantMap); diff != "" {
		t.Errorf("Diff (-got +want)\n%s", diff)
	}
}
