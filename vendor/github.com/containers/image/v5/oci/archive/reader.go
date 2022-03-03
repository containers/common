package archive

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containers/image/v5/internal/tmpdir"
	"github.com/containers/image/v5/oci/internal"
	"github.com/containers/image/v5/transports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage/pkg/archive"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Reader manages the temp directory that the oci archive is untarred to and the
// manifest of the images. It allows listing its contents and accessing
// individual images with less overhead than creating image references individually
// (because the archive is, if necessary, copied or decompressed only once)
type Reader struct {
	manifest      *imgspecv1.Index
	tempDirectory string
	path          string // The original, user-specified path
}

// NewReader returns a Reader for src. The caller should call Close() on the returned object
func NewReader(ctx context.Context, sys *types.SystemContext, ref types.ImageReference) (*Reader, error) {
	standalone, ok := ref.(ociArchiveReference)
	if !ok {
		return nil, fmt.Errorf("Internal error: NewReader called for a non-oci/archive ImageReference %s", transports.ImageName(ref))
	}
	if standalone.archiveReader != nil {
		return nil, fmt.Errorf("Internal error: NewReader called for a reader-bound reference %s", standalone.StringWithinTransport())
	}

	src := standalone.resolvedFile
	arch, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer arch.Close()

	dst, err := ioutil.TempDir(tmpdir.TemporaryDirectoryForBigFiles(sys), "oci")
	if err != nil {
		return nil, fmt.Errorf("error creating temp directory: %w", err)
	}

	reader := Reader{
		tempDirectory: dst,
		path:          src,
	}

	succeeded := false
	defer func() {
		if !succeeded {
			reader.Close()
		}
	}()
	if err := archive.NewDefaultArchiver().Untar(arch, dst, &archive.TarOptions{NoLchown: true}); err != nil {
		return nil, fmt.Errorf("error untarring file %q: %w", dst, err)
	}

	indexJSON, err := os.Open(filepath.Join(dst, "index.json"))
	if err != nil {
		return nil, err
	}
	defer indexJSON.Close()
	reader.manifest = &imgspecv1.Index{}
	if err := json.NewDecoder(indexJSON).Decode(reader.manifest); err != nil {
		return nil, err
	}
	succeeded = true
	return &reader, nil
}

// ListResult wraps the image reference and the manifest for loading
type ListResult struct {
	ImageRef           types.ImageReference
	ManifestDescriptor imgspecv1.Descriptor
}

// List returns a slice of manifests included in the archive
func (r *Reader) List() ([]ListResult, error) {
	var res []ListResult

	for _, md := range r.manifest.Manifests {
		refName := internal.NameFromAnnotations(md.Annotations)
		ref, err := newReference(r.path, refName, -1, r, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating image reference: %w", err)
		}
		reference := ListResult{
			ImageRef:           ref,
			ManifestDescriptor: md,
		}
		res = append(res, reference)
	}
	return res, nil
}

// Close deletes temporary files associated with the Reader, if any.
func (r *Reader) Close() error {
	return os.RemoveAll(r.tempDirectory)
}
