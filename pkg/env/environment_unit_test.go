package env

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBranchVersion(t *testing.T) {
	e := Environment{
		Options: Options{},
	}

	for _, tc := range []struct {
		branchName  string
		shouldError bool
		expectedVal int
	}{
		{"release-12.0", false, 12},
		{"release-12", true, 12},
		{"12.0", true, 12},
	} {
		e.Options.Branch = tc.branchName
		ver, err := e.BranchVersion()
		if tc.shouldError {
			require.NoError(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.expectedVal, ver)
		}
	}
}

func TestGetRepoTags(t *testing.T) {
	// TODO
}
