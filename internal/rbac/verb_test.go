package rbac_test

import (
	"errors"
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/rbac"
)

func TestParseActions(t *testing.T) {
	allVerbs := []string{
		http.MethodHead,
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
		http.MethodConnect,
		http.MethodTrace,
	}
	slices.Sort(allVerbs)

	someVerbs := []string{http.MethodGet, http.MethodPost, http.MethodDelete}
	slices.Sort(someVerbs)

	tests := []struct {
		name     string
		input    []string
		expected []string
		wantErr  error
	}{
		{
			name:     "valid actions",
			input:    []string{"get", "Post", " DeLeTe  "},
			expected: someVerbs,
			wantErr:  nil,
		},
		{
			name:     "wildcard action",
			input:    []string{"*"},
			expected: allVerbs,
			wantErr:  nil,
		},
		{
			name:     "invalid action",
			input:    []string{"Post", "gET", "PUT", "unknown"},
			expected: nil,
			wantErr:  rbac.ErrInvalidVerb,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions, err := rbac.ParseVerbs(tt.input)
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
