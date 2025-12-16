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

	tcs := []struct {
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
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			actions, err := rbac.ParseVerbs(tc.input)
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("ParseActions() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !reflect.DeepEqual(actions, tc.expected) {
				t.Errorf("ParseActions() got = %v, want %v", actions, tc.expected)
				return
			}
		})
	}
}
