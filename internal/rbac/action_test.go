package rbac_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/rbac"
)

func TestParseActions(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []rbac.Action
		wantErr  error
	}{
		{
			name:     "valid actions",
			input:    []string{"pull", "push", "delete"},
			expected: []rbac.Action{rbac.ActionPull, rbac.ActionPush, rbac.ActionDelete},
			wantErr:  nil,
		},
		{
			name:     "wildcard action",
			input:    []string{"*"},
			expected: []rbac.Action{rbac.ActionPull, rbac.ActionPush, rbac.ActionDelete},
			wantErr:  nil,
		},
		{
			name:     "invalid action",
			input:    []string{"pull", "push", "delete", "unknown"},
			expected: nil,
			wantErr:  rbac.ErrActionInvalid,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions, err := rbac.ParseActions(tt.input)
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("ParseActions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(actions, tt.expected) {
				t.Errorf("ParseActions() got = %v, want %v", actions, tt.expected)
				return
			}
		})
	}
}
