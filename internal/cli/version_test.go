package cli

import (
	"bytes"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionCmd covers asking folio which release it is: "folio version" prints the version it was
// built as and the platform it was built for, on one line, the way "go version" does. It has no
// use for the account, so it does not create the database.
func TestVersionCmd(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "version")
	cmd.BuildVersion = "1.0.0"

	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())

	assert.Equal(t, "folio version v1.0.0 "+runtime.GOOS+"/"+runtime.GOARCH+"\n", out.String())
	assert.NoFileExists(t, dbPath, "the version must not touch the database of the account")
}

// TestVersionCmd_WithoutARelease covers the builds that the release build did not make, which have
// no version handed to them. One that "go install" made knows the version of the module it was
// built from, and says that. One built from a checkout that knows nothing says so, rather than
// print a version that is not there.
func TestVersionCmd_WithoutARelease(t *testing.T) {
	tests := map[string]struct {
		info     *debug.BuildInfo
		expected string
	}{
		"installed with go install": {
			info:     &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}},
			expected: "v1.2.3",
		},
		"built from a checkout": {
			info:     &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			expected: "devel",
		},
		"a module without a version": {
			info:     &debug.BuildInfo{},
			expected: "devel",
		},
		"no build information": {
			expected: "devel",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd, _ := NewTestingCmd(t, "version")
			cmd.buildInfo = func() (*debug.BuildInfo, bool) { return tt.info, tt.info != nil }

			var out bytes.Buffer
			cmd.SetOut(&out)

			require.NoError(t, cmd.Execute())

			assert.Equal(t, "folio version "+tt.expected+" "+runtime.GOOS+"/"+runtime.GOARCH+"\n", out.String())
		})
	}
}
