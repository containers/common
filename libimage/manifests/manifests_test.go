package manifests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/containerd/containerd/platforms"
	"github.com/containers/common/pkg/manifests"
	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/directory"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/pkg/compression"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	digest "github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_ List = &list{}

	sys = &types.SystemContext{
		SystemRegistriesConfPath: "../../tests/registries.conf",
		SignaturePolicyPath:      "../../tests/policy.json",
	}
	amd64sys = &types.SystemContext{ArchitectureChoice: "amd64"}
	arm64sys = &types.SystemContext{ArchitectureChoice: "arm64"}
	ppc64sys = &types.SystemContext{ArchitectureChoice: "ppc64le"}
)

type listPtr = *list

const (
	listImageName = "foo"

	otherListImage          = "docker://registry.k8s.io/pause:3.1"
	otherListDigest         = "sha256:f78411e19d84a252e53bff71a4407a5686c46983a2c2eeed83929b888179acea"
	otherListAmd64Digest    = "sha256:59eec8837a4d942cc19a52b8c09ea75121acc38114a2c68b98983ce9356b8610"
	otherListArm64Digest    = "sha256:f365626a556e58189fc21d099fc64603db0f440bff07f77c740989515c544a39"
	otherListPpc64Digest    = "sha256:bcf9771c0b505e68c65440474179592ffdfa98790eb54ffbf129969c5e429990"
	otherListInstanceDigest = "docker://registry.k8s.io/pause@sha256:f365626a556e58189fc21d099fc64603db0f440bff07f77c740989515c544a39"
)

func TestSaveLoad(t *testing.T) {
	if unshare.IsRootless() {
		t.Skip("Test can only run as root")
	}

	dir := t.TempDir()
	storeOptions := storage.StoreOptions{
		GraphRoot:       filepath.Join(dir, "root"),
		RunRoot:         filepath.Join(dir, "runroot"),
		GraphDriverName: "vfs",
	}
	store, err := storage.GetStore(storeOptions)
	assert.NoError(t, err, "error opening store")
	if store == nil {
		return
	}
	defer func() {
		if _, err := store.Shutdown(true); err != nil {
			assert.NoError(t, err, "error closing store")
		}
	}()

	list := Create()
	list.(listPtr).artifacts.Detached[otherListDigest] = "relative-path-names-are-messy" // set to check that this data is recorded
	assert.NotNil(t, list, "Create() returned nil?")

	image, err := list.SaveToImage(store, "", []string{listImageName}, manifest.DockerV2ListMediaType)
	assert.NoError(t, err, "SaveToImage(1)")
	locker, err := LockerForImage(store, image)
	assert.NoError(t, err, "LockerForImage()")
	locker.Lock()
	defer locker.Unlock()
	imageReused, err := list.SaveToImage(store, image, nil, manifest.DockerV2ListMediaType)
	assert.NoError(t, err, "SaveToImage(2)")

	_, list, err = LoadFromImage(store, image)
	assert.NoError(t, err, "LoadFromImage(1)")
	assert.NotNilf(t, list, "LoadFromImage(1)")
	_, list, err = LoadFromImage(store, imageReused)
	assert.NoError(t, err, "LoadFromImage(2)")
	assert.NotNilf(t, list, "LoadFromImage(2)")
	_, list, err = LoadFromImage(store, listImageName)
	assert.NoError(t, err, "LoadFromImage(3)")
	assert.NotNilf(t, list, "LoadFromImage(3)")

	assert.Equal(t, list.(listPtr).artifacts.Detached[otherListDigest], "relative-path-names-are-messy") // check that this data is loaded
}

func TestAddRemove(t *testing.T) {
	if unshare.IsRootless() {
		t.Skip("Test can only run as root")
	}
	ctx := context.Background()

	ref, err := alltransports.ParseImageName(otherListImage)
	assert.NoError(t, err, "ParseImageName(%q)", otherListImage)
	src, err := ref.NewImageSource(ctx, sys)
	assert.NoError(t, err, "NewImageSource(%q)", otherListImage)
	defer assert.NoError(t, src.Close(), "ImageSource.Close()")
	m, _, err := src.GetManifest(ctx, nil)
	assert.NoError(t, err, "ImageSource.GetManifest()")
	assert.NoError(t, src.Close(), "ImageSource.GetManifest()")
	listDigest, err := manifest.Digest(m)
	assert.NoError(t, err, "manifest.Digest()")
	assert.Equalf(t, listDigest.String(), otherListDigest, "digest for image %q changed?", otherListImage)

	l, err := manifests.FromBlob(m)
	assert.NoError(t, err, "manifests.FromBlob()")
	assert.NotNilf(t, l, "manifests.FromBlob()")
	assert.Equalf(t, len(l.Instances()), 5, "image %q had an arch added?", otherListImage)

	list := Create()
	instanceDigest, err := list.Add(ctx, amd64sys, ref, false)
	assert.NoError(t, err, "list.Add(all=false)")
	assert.Equal(t, instanceDigest.String(), otherListAmd64Digest)
	assert.Equalf(t, len(list.Instances()), 1, "too many instances added")

	list = Create()
	instanceDigest, err = list.Add(ctx, arm64sys, ref, false)
	assert.NoError(t, err, "list.Add(all=false)")
	assert.Equal(t, instanceDigest.String(), otherListArm64Digest)
	assert.Equalf(t, len(list.Instances()), 1, "too many instances added")

	list = Create()
	instanceDigest, err = list.Add(ctx, ppc64sys, ref, false)
	assert.NoError(t, err, "list.Add(all=false)")
	assert.Equal(t, instanceDigest.String(), otherListPpc64Digest)
	assert.Equalf(t, len(list.Instances()), 1, "too many instances added")

	_, err = list.Add(ctx, sys, ref, true)
	assert.NoError(t, err, "list.Add(all=true)")
	assert.Equalf(t, len(list.Instances()), 5, "too many instances added")

	list = Create()
	_, err = list.Add(ctx, sys, ref, true)
	assert.NoError(t, err, "list.Add(all=true)")
	assert.Equalf(t, len(list.Instances()), 5, "too many instances added", otherListImage)

	for _, instance := range list.Instances() {
		assert.NoErrorf(t, list.Remove(instance), "error removing instance %q", instance)
	}
	assert.Equalf(t, len(list.Instances()), 0, "should have removed all instances")

	ref, err = alltransports.ParseImageName(otherListInstanceDigest)
	assert.NoError(t, err, "ParseImageName(%q)", otherListInstanceDigest)

	list = Create()
	_, err = list.Add(ctx, sys, ref, false)
	assert.NoError(t, err, "list.Add(all=false)")
	assert.Equalf(t, len(list.Instances()), 1, "too many instances added", otherListInstanceDigest)

	list = Create()
	_, err = list.Add(ctx, sys, ref, true)
	assert.NoError(t, err, "list.Add(all=true)")
	assert.Equalf(t, len(list.Instances()), 1, "too many instances added", otherListInstanceDigest)
}

func TestAddArtifact(t *testing.T) {
	if unshare.IsRootless() {
		t.Skip("Test can only run as root")
	}
	ctx := context.Background()
	dir := t.TempDir()
	storeOptions := storage.StoreOptions{
		GraphRoot:       filepath.Join(dir, "root"),
		RunRoot:         filepath.Join(dir, "runroot"),
		GraphDriverName: "vfs",
	}
	emptyConfigFile := filepath.Join(dir, "empty.json")
	err := os.WriteFile(emptyConfigFile, []byte("{}"), 0o600)
	assert.NoError(t, err, "error creating a mostly-empty file")
	store, err := storage.GetStore(storeOptions)
	assert.NoError(t, err, "error opening store")
	if store == nil {
		return
	}
	defer func() {
		if _, err := store.Shutdown(true); err != nil {
			assert.NoError(t, err, "error closing store")
		}
	}()
	subjectImageName := "oci-archive:" + filepath.Join("..", "testdata", "oci-name-only.tar.gz")
	subjectReference, err := alltransports.ParseImageName(subjectImageName)
	require.NoError(t, err)
	testCombination := func(t *testing.T, file, fileMediaType string, manifestArtifactType *string, platform v1.Platform, configDescriptor *v1.Descriptor, configFile string, layerMediaType *string, annotations map[string]string, subjectReference types.ImageReference, excludeTitles bool) {
		subjectName := "<nil>"
		if subjectReference != nil {
			subjectName = transports.ImageName(subjectReference)
		}
		configMediaType := "<nil>"
		if configDescriptor != nil {
			configMediaType = configDescriptor.MediaType
		}
		var platformStr string
		if platform.OS != "" && platform.Architecture != "" {
			platformStr = platforms.Format(platform)
		}
		manifestArtifactTypeStr := "<nil>"
		if manifestArtifactType != nil {
			manifestArtifactTypeStr = *manifestArtifactType
		}
		layerMediaTypeStr := "<nil>"
		if layerMediaType != nil {
			layerMediaTypeStr = *layerMediaType
		}
		annotationsStr := "<nil>"
		if annotations != nil {
			var annotationsSlice []string
			for k, v := range annotations {
				annotationsSlice = append(annotationsSlice, fmt.Sprintf("%q=%q", k, v))
			}
			sort.Strings(annotationsSlice)
			annotationsStr = "{" + strings.Join(annotationsSlice, ",") + "}"
		}
		fileBasename := ""
		if file != "" {
			fileBasename = filepath.Base(file)
		}
		desc := fmt.Sprint("file=", fileBasename, ",fileMediaType=", fileMediaType, ",manifestArtifactType=", manifestArtifactTypeStr, ",platform=", platformStr, ",configMediaType=", configMediaType, ",configFile=", configFile, ",layerMediaType=", layerMediaTypeStr, ",annotations=", annotationsStr, ",subject=", subjectName, ",excludeTitles=", excludeTitles)
		t.Run(desc, func(t *testing.T) {
			// create the new index and add the file to it
			options := AddArtifactOptions{
				ManifestArtifactType: manifestArtifactType,
				Platform:             platform,
				ConfigDescriptor:     configDescriptor,
				ConfigFile:           configFile,
				LayerMediaType:       layerMediaType,
				Annotations:          annotations,
				SubjectReference:     subjectReference,
				ExcludeTitles:        excludeTitles,
			}
			list := Create()
			var instanceDigest digest.Digest
			if file != "" {
				instanceDigest, err = list.AddArtifact(ctx, sys, options, file)
			} else {
				instanceDigest, err = list.AddArtifact(ctx, sys, options)
			}
			assert.NoErrorf(t, err, "list.AddArtifact(%#v)", options)
			assert.Equal(t, 1, len(list.Instances()), "too many instances added")
			// have to save it before we can create a reference to it
			_, err = list.SaveToImage(store, "", nil, "")
			require.NoError(t, err)
			// get ready to copy it, sort of
			ref, err := list.Reference(store, cp.CopyAllImages, nil)
			require.NoError(t, err)
			// fetch the manifest for the artifact that we just added to the index
			src, err := ref.NewImageSource(ctx, &types.SystemContext{})
			require.NoError(t, err)
			defer src.Close()
			manifestBytes, manifestType, err := src.GetManifest(ctx, &instanceDigest)
			require.NoError(t, err)
			// decode the artifact manifest
			var m v1.Manifest
			if annotations != nil {
				m.Annotations = make(map[string]string)
			}
			err = json.Unmarshal(manifestBytes, &m)
			require.NoError(t, err)
			// check that the artifact manifest looks right
			assert.Equal(t, v1.MediaTypeImageManifest, manifestType)
			assert.Equal(t, v1.MediaTypeImageManifest, m.MediaType)
			expectedManifestArtifactType := "application/vnd.unknown.artifact.v1"
			if manifestArtifactType != nil {
				expectedManifestArtifactType = *manifestArtifactType
			}
			assert.Equal(t, expectedManifestArtifactType, m.ArtifactType)
			// check the config blob info
			configFileSize := v1.DescriptorEmptyJSON.Size
			configFileDigest := v1.DescriptorEmptyJSON.Digest
			if configFile != "" {
				f, err := os.Open(configFile)
				require.NoError(t, err)
				t.Cleanup(func() { assert.NoError(t, f.Close()) })
				st, err := f.Stat()
				require.NoError(t, err)
				configFileSize = st.Size()
				digester := digest.Canonical.Digester()
				_, err = io.Copy(digester.Hash(), f)
				require.NoError(t, err)
				configFileDigest = digester.Digest()
			}
			switch {
			case configDescriptor != nil && configFile != "":
				assert.Equal(t, configDescriptor.MediaType, m.Config.MediaType, "did not record expected media type for config with file")
				assert.Equal(t, configFileDigest, m.Config.Digest, "did not record expected digest for config with file")
				assert.Equal(t, configFileSize, m.Config.Size, "did not record expected size for config with file")
			case configFile != "":
				assert.Equal(t, v1.MediaTypeImageConfig, m.Config.MediaType, "did not record expected media type for config file")
				assert.Equal(t, configFileDigest, m.Config.Digest, "did not record expected digest for config with file")
				assert.Equal(t, configFileSize, m.Config.Size, "did not record expected size for config with file")
			case configDescriptor != nil:
				assert.Equal(t, configDescriptor.MediaType, m.Config.MediaType, "did not record expected mediaType for empty config")
				assert.Equal(t, configDescriptor.Digest, m.Config.Digest, "did not record expected digest for empty config")
				assert.Equal(t, configDescriptor.Size, m.Config.Size, "did not record expected digest for empty config")
			default:
				assert.Equal(t, v1.DescriptorEmptyJSON.MediaType, m.Config.MediaType, "did not record expected mediaType for empty config and no config file")
				assert.Equal(t, v1.DescriptorEmptyJSON.Digest, m.Config.Digest, "did not record expected digest for empty config and no config file")
				assert.Equal(t, v1.DescriptorEmptyJSON.Size, m.Config.Size, "did not record expected digest for empty config and no config file")
			}
			// if we had a file, it should be there as the "layer", otherwise it should be the empty descriptor
			assert.Equal(t, 1, len(m.Layers), "expected only one layer")
			if file == "" {
				assert.Equal(t, v1.DescriptorEmptyJSON.MediaType, m.Layers[0].MediaType, "did not record empty JSON as layer")
				assert.Equal(t, v1.DescriptorEmptyJSON.Digest, m.Layers[0].Digest, "did not record empty JSON as layer")
				assert.Equal(t, v1.DescriptorEmptyJSON.Size, m.Layers[0].Size, "did not record empty JSON as layer")
			} else {
				// we need to have preserved its size
				st, err := os.Stat(file)
				require.NoError(t, err)
				assert.Equal(t, st.Size(), m.Layers[0].Size, "did not record size of file")
				// did we set the type correctly?
				expectedLayerMediaType := fileMediaType
				if layerMediaType != nil {
					expectedLayerMediaType = *layerMediaType
				}
				assert.Equal(t, expectedLayerMediaType, m.Layers[0].MediaType, "recorded MediaType for layer was wrong")
				// did we set the digest correctly?
				f, err := os.Open(file)
				require.NoError(t, err)
				defer f.Close()
				digester := m.Layers[0].Digest.Algorithm().Digester()
				_, err = io.Copy(digester.Hash(), f)
				require.NoError(t, err)
				assert.Equal(t, digester.Digest().String(), m.Layers[0].Digest.String(), "recorded digest was wrong")
				// did we add that annotation?
				if excludeTitles && file != "" {
					assert.Nil(t, m.Layers[0].Annotations, "expected no layer annotations")
				} else {
					assert.Equal(t, 1, len(m.Layers[0].Annotations), "expected a layer annotation")
					assert.Equal(t, fileBasename, m.Layers[0].Annotations[v1.AnnotationTitle], "expected a title annotation")
				}
			}
			// did we set the annotations?
			assert.EqualValues(t, annotations, m.Annotations, "recorded annotations were wrong")
			if subjectReference != nil {
				// did we set the subject right?
				subject, err := subjectReference.NewImageSource(ctx, &types.SystemContext{})
				require.NoError(t, err)
				defer subject.Close()
				subjectManifestBytes, subjectManifestType, err := subject.GetManifest(ctx, nil)
				require.NoError(t, err)
				subjectManifestDigest, err := manifest.Digest(subjectManifestBytes)
				require.NoError(t, err)
				var s v1.Manifest
				err = json.Unmarshal(subjectManifestBytes, &s)
				require.NoError(t, err)
				assert.Equal(t, m.Subject.Digest, subjectManifestDigest)
				assert.Equal(t, m.Subject.MediaType, subjectManifestType)
				assert.Equal(t, int64(len(subjectManifestBytes)), m.Subject.Size)
			}
		})
	}
	for file, fileMediaType := range map[string]string{
		"": v1.DescriptorEmptyJSON.MediaType,
		filepath.Join("..", "testdata", "containers.conf"):      "text/plain",
		filepath.Join("..", "testdata", "oci-name-only.tar.gz"): "application/gzip",
		filepath.Join("..", "..", "logos", "containers.png"):    "image/png",
	} {
		defaultManifestArtifactType := "application/vnd.unknown.artifact.v1"
		manifestArtifactType := &defaultManifestArtifactType
		platform := v1.Platform{OS: runtime.GOOS, Architecture: runtime.GOARCH}
		configDescriptor := &v1.DescriptorEmptyJSON
		configFile := ""
		emptyString := ""
		layerMediaType := &emptyString
		annotations := make(map[string]string)
		excludeTitles := false
		for _, manifestArtifactType := range []string{"(nil)", "", "application/vnd.unknown.artifact.v1"} {
			manifestArtifactType := &manifestArtifactType
			if *manifestArtifactType == "(nil)" {
				manifestArtifactType = nil
			}
			testCombination(t, file, fileMediaType, manifestArtifactType, platform, configDescriptor, configFile, layerMediaType, annotations, subjectReference, excludeTitles)
		}
		for _, platform := range []v1.Platform{
			{},
			{
				OS:           runtime.GOOS,
				Architecture: runtime.GOARCH,
			},
		} {
			testCombination(t, file, fileMediaType, manifestArtifactType, platform, configDescriptor, configFile, layerMediaType, annotations, subjectReference, excludeTitles)
		}
		for _, configDescriptor := range []*v1.Descriptor{
			nil,
			{MediaType: "application/x-config", Size: 0, Digest: digest.Canonical.FromString("")},
			{MediaType: v1.MediaTypeImageConfig, Size: 0, Digest: digest.Canonical.FromString("")},
			&v1.DescriptorEmptyJSON,
		} {
			for _, configFile := range []string{
				"",
				emptyConfigFile,
			} {
				testCombination(t, file, fileMediaType, manifestArtifactType, platform, configDescriptor, configFile, layerMediaType, annotations, subjectReference, excludeTitles)
			}
		}
		for _, layerMediaType := range []string{"(nil)", "", "text/plain", "application/octet-stream"} {
			layerMediaType := &layerMediaType
			if *layerMediaType == "(nil)" {
				layerMediaType = nil
			}
			testCombination(t, file, fileMediaType, manifestArtifactType, platform, configDescriptor, configFile, layerMediaType, annotations, subjectReference, excludeTitles)
		}
		for _, annotations := range []map[string]string{
			nil,
			{},
			{
				"annotationA": "valueA",
			},
			{
				"annotationB": "valueB",
				"annotationC": "valueC",
			},
		} {
			testCombination(t, file, fileMediaType, manifestArtifactType, platform, configDescriptor, configFile, layerMediaType, annotations, subjectReference, excludeTitles)
		}
		for _, subjectName := range []string{"", subjectImageName} {
			var subjectReference types.ImageReference
			if subjectName != "" {
				var err error
				subjectReference, err = alltransports.ParseImageName(subjectName)
				require.NoError(t, err)
			}
			testCombination(t, file, fileMediaType, manifestArtifactType, platform, configDescriptor, configFile, layerMediaType, annotations, subjectReference, excludeTitles)
		}
		for _, excludeTitles := range []bool{false, true} {
			testCombination(t, file, fileMediaType, manifestArtifactType, platform, configDescriptor, configFile, layerMediaType, annotations, subjectReference, excludeTitles)
		}
	}
}

func TestReference(t *testing.T) {
	if unshare.IsRootless() {
		t.Skip("Test can only run as root")
	}
	ctx := context.Background()

	dir := t.TempDir()
	storeOptions := storage.StoreOptions{
		GraphRoot:       filepath.Join(dir, "root"),
		RunRoot:         filepath.Join(dir, "runroot"),
		GraphDriverName: "vfs",
	}
	store, err := storage.GetStore(storeOptions)
	assert.NoError(t, err, "error opening store")
	if store == nil {
		return
	}
	defer func() {
		if _, err := store.Shutdown(true); err != nil {
			assert.NoError(t, err, "error closing store")
		}
	}()

	policy, err := signature.NewPolicyFromFile(sys.SignaturePolicyPath)
	assert.NoErrorf(t, err, "NewPolicyFromFile()")
	policyContext, err := signature.NewPolicyContext(policy)
	assert.NoErrorf(t, err, "NewPolicyContext()")
	destRef, err := directory.NewReference(filepath.Join(dir, "directory"))
	assert.NoErrorf(t, err, "NewReference()")
	checkCopy := func(ref types.ImageReference, selection cp.ImageListSelection, instances []digest.Digest) error {
		if ref == nil {
			return errors.New("no reference")
		}
		copyOptions := &cp.Options{
			ImageListSelection: selection,
			SourceCtx:          sys,
			DestinationCtx:     sys,
			Instances:          instances,
		}
		// t.Helper()
		_, err := cp.Image(ctx, policyContext, destRef, ref, copyOptions)
		return err
	}
	ref, err := alltransports.ParseImageName(otherListImage)
	assert.NoErrorf(t, err, "ParseImageName(%q)", otherListImage)

	list := Create()
	_, err = list.Add(ctx, ppc64sys, ref, false)
	assert.NoError(t, err, "list.Add(all=false)")

	smallJSON := filepath.Join(dir, "small.json")
	err = os.WriteFile(smallJSON, []byte(`{"slice":[1, 2, 3]}`), 0o600)
	assert.NoError(t, err)

	minimumJSON := filepath.Join(dir, "minimum.json")
	err = os.WriteFile(minimumJSON, []byte("{}"), 0o600)
	assert.NoError(t, err)

	emptyJSON := filepath.Join(dir, "empty.json")
	err = os.WriteFile(emptyJSON, []byte(""), 0o600)
	assert.NoError(t, err)

	artifactOptions := AddArtifactOptions{
		ConfigFile: emptyJSON,
	}
	_, err = list.AddArtifact(ctx, &types.SystemContext{}, artifactOptions)
	assert.NoErrorf(t, err, "list.AddArtifact(file=%s)", emptyJSON)

	artifactOptions = AddArtifactOptions{
		ConfigDescriptor: &v1.DescriptorEmptyJSON,
	}
	minimumArtifactDigest, err := list.AddArtifact(ctx, &types.SystemContext{}, artifactOptions, minimumJSON)
	assert.NoError(t, err, "list.AddArtifact(file=%s)", minimumJSON)

	artifactOptions = AddArtifactOptions{}
	smallArtifactDigest, err := list.AddArtifact(ctx, &types.SystemContext{}, artifactOptions, smallJSON)
	assert.NoError(t, err, "list.AddArtifact(file=%s)", smallJSON)

	listRef, err := list.Reference(store, cp.CopyAllImages, nil)
	assert.Error(t, err, "list.Reference(never saved)")
	assert.Nilf(t, listRef, "list.Reference(never saved)")

	listRef, err = list.Reference(store, cp.CopyAllImages, nil)
	assert.Error(t, err, "list.Reference(never saved)")
	assert.Nilf(t, listRef, "list.Reference(never saved)")

	listRef, err = list.Reference(store, cp.CopySystemImage, nil)
	assert.Error(t, err, "list.Reference(never saved)")
	assert.Nilf(t, listRef, "list.Reference(never saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest})
	assert.Error(t, err, "list.Reference(never saved)")
	assert.Nilf(t, listRef, "list.Reference(never saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest})
	assert.Error(t, err, "list.Reference(never saved)")
	assert.Nilf(t, listRef, "list.Reference(never saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest, minimumArtifactDigest})
	assert.Error(t, err, "list.Reference(never saved)")
	assert.Nilf(t, listRef, "list.Reference(never saved)")
	err = checkCopy(listRef, cp.CopySpecificImages, []digest.Digest{minimumArtifactDigest})
	assert.Error(t, err, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest, minimumArtifactDigest, smallArtifactDigest})
	assert.Error(t, err, "list.Reference(never saved)")
	assert.Nilf(t, listRef, "list.Reference(never saved)")
	err = checkCopy(listRef, cp.CopySpecificImages, []digest.Digest{minimumArtifactDigest, smallArtifactDigest})
	assert.Error(t, err, "list.Reference(saved)")

	_, err = list.SaveToImage(store, "", []string{listImageName}, "")
	assert.NoError(t, err, "SaveToImage")

	listRef, err = list.Reference(store, cp.CopyAllImages, nil)
	assert.NoError(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySystemImage, nil)
	assert.NoError(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, nil)
	assert.NoError(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest})
	assert.NoError(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest})
	assert.NoError(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest, minimumArtifactDigest})
	assert.NoError(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")
	err = checkCopy(listRef, cp.CopySpecificImages, []digest.Digest{minimumArtifactDigest})
	assert.NoError(t, err, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest, minimumArtifactDigest, smallArtifactDigest})
	assert.NoError(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")
	err = checkCopy(listRef, cp.CopySpecificImages, []digest.Digest{minimumArtifactDigest, smallArtifactDigest})
	assert.NoError(t, err, "list.Reference(saved)")

	_, err = list.Add(ctx, sys, ref, true)
	assert.NoError(t, err, "list.Add(all=true)")

	listRef, err = list.Reference(store, cp.CopyAllImages, nil)
	assert.NoError(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")
	err = checkCopy(listRef, cp.CopyAllImages, nil)
	assert.NoError(t, err, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySystemImage, nil)
	assert.NoError(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")
	err = checkCopy(listRef, cp.CopySystemImage, nil)
	assert.NoError(t, err, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, nil)
	assert.NoError(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")
	err = checkCopy(listRef, cp.CopySpecificImages, nil)
	assert.NoError(t, err, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest})
	assert.NoError(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")
	err = checkCopy(listRef, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest})
	assert.NoError(t, err, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest})
	assert.NoError(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")
	err = checkCopy(listRef, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest})
	assert.NoError(t, err, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest, minimumArtifactDigest})
	assert.NoError(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")
	err = checkCopy(listRef, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest, minimumArtifactDigest})
	assert.NoError(t, err, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest, minimumArtifactDigest, smallArtifactDigest})
	assert.NoError(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")
	err = checkCopy(listRef, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest, minimumArtifactDigest, smallArtifactDigest})
	assert.NoError(t, err, "list.Reference(saved)")
}

func TestPushManifest(t *testing.T) {
	if unshare.IsRootless() {
		t.Skip("Test can only run as root")
	}
	ctx := context.Background()

	dir := t.TempDir()
	storeOptions := storage.StoreOptions{
		GraphRoot:       filepath.Join(dir, "root"),
		RunRoot:         filepath.Join(dir, "runroot"),
		GraphDriverName: "vfs",
	}
	store, err := storage.GetStore(storeOptions)
	assert.NoError(t, err, "error opening store")
	if store == nil {
		return
	}
	defer func() {
		if _, err := store.Shutdown(true); err != nil {
			assert.NoError(t, err, "error closing store")
		}
	}()

	destRef, err := alltransports.ParseImageName(fmt.Sprintf("dir:%s", t.TempDir()))
	assert.NoError(t, err, "ParseImageName()")

	ref, err := alltransports.ParseImageName(otherListImage)
	assert.NoErrorf(t, err, "ParseImageName(%q)", otherListImage)

	list := Create()
	_, err = list.Add(ctx, sys, ref, true)
	assert.NoError(t, err, "list.Add(all=true)")

	_, err = list.SaveToImage(store, "", []string{listImageName}, "")
	assert.NoError(t, err, "SaveToImage")

	options := PushOptions{
		Store:              store,
		SystemContext:      sys,
		ImageListSelection: cp.CopyAllImages,
		Instances:          nil,
	}
	_, _, err = list.Push(ctx, destRef, options)
	assert.NoError(t, err, "list.Push(all)")

	options.ImageListSelection = cp.CopySystemImage
	_, _, err = list.Push(ctx, destRef, options)
	assert.NoError(t, err, "list.Push(local)")

	options.ImageListSelection = cp.CopySpecificImages
	_, _, err = list.Push(ctx, destRef, options)
	assert.NoError(t, err, "list.Push(none specified)")

	options.Instances = []digest.Digest{otherListAmd64Digest}
	_, _, err = list.Push(ctx, destRef, options)
	assert.NoError(t, err, "list.Push(one specified)")

	options.Instances = append(options.Instances, otherListArm64Digest)
	_, _, err = list.Push(ctx, destRef, options)
	assert.NoError(t, err, "list.Push(two specified)")

	options.Instances = append(options.Instances, otherListPpc64Digest)
	_, _, err = list.Push(ctx, destRef, options)
	assert.NoError(t, err, "list.Push(three specified)")

	options.Instances = append(options.Instances, otherListDigest)
	_, _, err = list.Push(ctx, destRef, options)
	assert.NoError(t, err, "list.Push(four specified)")

	bogusDestRef, err := alltransports.ParseImageName("docker://localhost/bogus/dest:latest")
	assert.NoErrorf(t, err, "ParseImageName()")

	var logBuffer bytes.Buffer
	logBuffer = bytes.Buffer{}
	logrus.SetOutput(&logBuffer)
	maxRetry := uint(5)
	delay := 3 * time.Second
	options.MaxRetries = &maxRetry
	_, _, err = list.Push(ctx, bogusDestRef, options)
	assert.Error(t, err)
	logString := logBuffer.String()
	// Must show warning where libimage is going to retry 5 times with 1s delay
	assert.Contains(t, logString, "Failed, retrying in 1s ... (1/5)", "warning not matched")

	logBuffer = bytes.Buffer{}
	logrus.SetOutput(&logBuffer)
	options.RetryDelay = &delay
	_, _, err = list.Push(ctx, bogusDestRef, options)
	assert.Error(t, err)
	logString = logBuffer.String()
	// Must show warning where libimage is going to retry 5 times with 3s delay
	assert.Contains(t, logString, "Failed, retrying in 3s ... (1/5)", "warning not matched")

	options.AddCompression = []string{"zstd"}
	options.ImageListSelection = cp.CopyAllImages
	_, _, err = list.Push(ctx, destRef, options)
	assert.NoError(t, err, "list.Push(with replication for zstd specified)")

	options.ForceCompressionFormat = true
	options.ImageListSelection = cp.CopyAllImages
	options.SystemContext.CompressionFormat = &compression.Gzip
	_, _, err = list.Push(ctx, destRef, options)
	assert.NoError(t, err, "list.Push(with ForceCompressionFormat: true)")
}

func TestInstanceByImageAndFiles(t *testing.T) {
	if unshare.IsRootless() {
		t.Skip("Test can only run as root")
	}
	ctx := context.Background()

	dir := t.TempDir()
	storeOptions := storage.StoreOptions{
		GraphRoot:       filepath.Join(dir, "root"),
		RunRoot:         filepath.Join(dir, "runroot"),
		GraphDriverName: "vfs",
	}
	store, err := storage.GetStore(storeOptions)
	assert.NoError(t, err, "error opening store")
	if store == nil {
		return
	}
	defer func() {
		if _, err := store.Shutdown(true); err != nil {
			assert.NoError(t, err, "error closing store")
		}
	}()

	cconfig := filepath.Join("..", "testdata", "containers.conf")
	absCconfig, err := filepath.Abs(cconfig)
	assert.NoError(t, err)
	gzipped := filepath.Join("..", "testdata", "oci-name-only.tar.gz")
	absGzipped, err := filepath.Abs(gzipped)
	assert.NoError(t, err)
	pngfile := filepath.Join("..", "..", "logos", "containers.png")
	absPngfile, err := filepath.Abs(pngfile)
	assert.NoError(t, err)

	list := Create()
	options := AddArtifactOptions{}
	firstInstanceDigest, err := list.AddArtifact(ctx, sys, options, cconfig, gzipped)
	assert.NoError(t, err)
	secondInstanceDigest, err := list.AddArtifact(ctx, sys, options, pngfile)
	assert.NoError(t, err)

	candidate, err := list.InstanceByFile(cconfig)
	assert.NoError(t, err)
	assert.Equal(t, firstInstanceDigest, candidate)
	candidate, err = list.InstanceByFile(gzipped)
	assert.NoError(t, err)
	assert.Equal(t, firstInstanceDigest, candidate)

	firstFiles, err := list.Files(firstInstanceDigest)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{absCconfig, absGzipped}, firstFiles)

	candidate, err = list.InstanceByFile(pngfile)
	assert.NoError(t, err)
	assert.Equal(t, secondInstanceDigest, candidate)

	secondFiles, err := list.Files(secondInstanceDigest)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{absPngfile}, secondFiles)

	_, err = list.InstanceByFile("ha ha, fooled you")
	assert.Error(t, err)
	assert.ErrorIs(t, err, os.ErrNotExist)

	otherDigest, err := digest.Parse(otherListDigest)
	assert.NoError(t, err)
	noFiles, err := list.Files(otherDigest)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{}, noFiles)
}

// TestAddIndexOfArtifacts ensures that we don't fail to preserve artifactType
// fields in artifact manifests when added from one list to another.
func TestAddIndexOfArtifacts(t *testing.T) {
	ctx := context.Background()

	absPath, err := filepath.Abs(filepath.Join("..", "..", "pkg", "manifests", "testdata", "artifacts", "index"))
	require.NoError(t, err)
	rawPath := "oci:" + absPath
	ref, err := alltransports.ParseImageName(rawPath)
	require.NoErrorf(t, err, "ParseImageName(%q)", rawPath)

	cookedList := Create()
	_, err = cookedList.Add(ctx, sys, ref, true)
	assert.NoError(t, err, "list.Add()")

	cooked := cookedList.OCIv1()
	for _, instance := range cooked.Manifests {
		assert.NotEmpty(t, instance.ArtifactType, "lost the artifactType field")
	}
}
