package env_test

import (
	"testing"

	"github.com/puerco/vtrelease/pkg/env"
	"github.com/puerco/vtrelease/pkg/env/envfakes"
	"github.com/stretchr/testify/require"
)

func TestLastVersion(t *testing.T) {
	for _, tc := range []struct {
		expectedVersion string
		branch          string
		tags            []string
		shouldError     bool
	}{
		// Standard
		{"v12.1.3", "release-12.0", []string{"v12.1.1", "v12.1.2", "v12.1.3"}, false},
		// Different order of tags
		{"v12.1.3", "release-12.0", []string{"v12.1.3", "v12.1.1", "v12.1.2"}, false},
		// Tags  from other branches
		{"v12.1.1", "release-12.0", []string{"v13.1.3", "v12.1.1", "v11.1.2"}, false},
		// Tags  from other branches
		{"v12.0.0", "release-12.0", []string{}, false},
		// Malformed branch
		{"v12.0.0", "release-12", []string{}, true},
	} {
		sut := env.Environment{
			Options: env.Options{Branch: tc.branch},
		}
		fake := &envfakes.FakeImplementation{}

		fake.GetRepoTagsReturns(tc.tags, nil)
		sut.SetImplementation(fake)
		ver, err := sut.LastVersion()

		if tc.shouldError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.expectedVersion, ver)
		}
	}
}
