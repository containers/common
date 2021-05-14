package libimage

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImport(t *testing.T) {
	runtime, cleanup := testNewRuntime(t)
	defer cleanup()
	ctx := context.Background()

	importOptions := &ImportOptions{}
	importOptions.Writer = os.Stdout

	imported, err := runtime.Import(ctx, "testdata/exported-container.tar", importOptions)
	require.NoError(t, err)

	image, resolvedName, err := runtime.LookupImage(imported, nil)
	require.NoError(t, err)
	require.Equal(t, imported, resolvedName)
	require.Equal(t, imported, "sha256:"+image.ID())
}
