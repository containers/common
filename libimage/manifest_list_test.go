//go:build !remote

package libimage

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	mathrand "math/rand"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/containers/common/pkg/config"
	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/pkg/compression"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateManifestList(t *testing.T) {
	runtime := testNewRuntime(t)
	ctx := context.Background()

	list, err := runtime.CreateManifestList("mylist")
	require.NoError(t, err)
	require.NotNil(t, list)
	initialID := list.ID()

	list, err = runtime.LookupManifestList("mylist")
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Equal(t, initialID, list.ID())

	_, rmErrors := runtime.RemoveImages(ctx, []string{"mylist"}, nil)
	require.Nil(t, rmErrors)

	_, err = runtime.LookupManifestList("nosuchthing")
	require.Error(t, err)
	require.True(t, errors.Is(err, storage.ErrImageUnknown))

	_, err = runtime.Pull(ctx, "busybox", config.PullPolicyMissing, nil)
	require.NoError(t, err)
	_, err = runtime.LookupManifestList("busybox")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotAManifestList))
}

// Inspect must contain both formats i.e OCIv1 and docker
func TestInspectManifestListWithAnnotations(t *testing.T) {
	listName := "testinspect"
	runtime := testNewRuntime(t)
	ctx := context.Background()

	list, err := runtime.CreateManifestList(listName)
	require.NoError(t, err)
	require.NotNil(t, list)

	manifestListOpts := &ManifestListAddOptions{All: true}
	_, err = list.Add(ctx, "docker://busybox", manifestListOpts)
	require.NoError(t, err)

	list, err = runtime.LookupManifestList(listName)
	require.NoError(t, err)
	require.NotNil(t, list)

	inspectReport, err := list.Inspect()
	// get digest of the first instance
	digest := inspectReport.Manifests[0].Digest
	require.NoError(t, err)

	annotateOptions := ManifestListAnnotateOptions{}
	annotations := map[string]string{"hello": "world"}
	annotateOptions.Annotations = annotations
	indexAnnotations := map[string]string{"goodbye": "globe"}
	annotateOptions.IndexAnnotations = indexAnnotations

	subjectPath, err := filepath.Abs(filepath.Join("..", "pkg", "manifests", "testdata", "artifacts", "blobs-only"))
	require.NoError(t, err)
	annotateOptions.Subject = "oci:" + subjectPath

	err = list.AnnotateInstance(digest, &annotateOptions)
	require.NoError(t, err)
	// Inspect list again
	inspectReport, err = list.Inspect()
	require.NoError(t, err)
	// verify annotation
	require.Contains(t, inspectReport.Manifests[0].Annotations, "hello")
	require.Equal(t, inspectReport.Manifests[0].Annotations["hello"], annotations["hello"])
	require.Equal(t, inspectReport.Annotations, indexAnnotations)
	require.Equal(t, inspectReport.Subject.MediaType, imgspecv1.MediaTypeImageManifest)

	// verify that we can clear the variant field by not setting it when we set the arch
	annotateOptions = ManifestListAnnotateOptions{
		Architecture: "arm64",
		Variant:      "v8",
	}
	err = list.AnnotateInstance(digest, &annotateOptions)
	require.NoError(t, err)
	inspectReport, err = list.Inspect()
	require.NoError(t, err)
	require.Equal(t, "arm64", inspectReport.Manifests[0].Platform.Architecture)
	require.Equal(t, "v8", inspectReport.Manifests[0].Platform.Variant)

	annotateOptions = ManifestListAnnotateOptions{
		Architecture: "arm64",
	}
	err = list.AnnotateInstance(digest, &annotateOptions)
	require.NoError(t, err)
	inspectReport, err = list.Inspect()
	require.NoError(t, err)
	require.Equal(t, "arm64", inspectReport.Manifests[0].Platform.Architecture)
	require.Equal(t, "", inspectReport.Manifests[0].Platform.Variant)
}

// Following test ensure that `Tag` tags the manifest list instead of resolved image.
// Both the tags should point to same image id
func TestCreateAndTagManifestList(t *testing.T) {
	tagName := "testlisttagged"
	listName := "testlist"
	runtime := testNewRuntime(t)
	ctx := context.Background()

	list, err := runtime.CreateManifestList(listName)
	require.NoError(t, err)
	require.NotNil(t, list)

	_, err = runtime.Load(ctx, "testdata/oci-unnamed.tar.gz", nil)
	require.NoError(t, err)

	// add a remote reference
	manifestListOpts := &ManifestListAddOptions{All: true}
	_, err = list.Add(ctx, "docker://busybox", manifestListOpts)
	require.NoError(t, err)

	// add a remote reference where we have to figure out that it's remote
	_, err = list.Add(ctx, "busybox", manifestListOpts)
	require.NoError(t, err)

	// add using a local image's ID
	_, err = list.Add(ctx, "5c8aca8137ac47e84c69ae93ce650ce967917cc001ba7aad5494073fac75b8b6", manifestListOpts)
	require.NoError(t, err)

	list, err = runtime.LookupManifestList(listName)
	require.NoError(t, err)
	require.NotNil(t, list)

	lookupOptions := &LookupImageOptions{ManifestList: true}
	image, _, err := runtime.LookupImage(listName, lookupOptions)
	require.NoError(t, err)
	require.NotNil(t, image)
	err = image.Tag(tagName)
	require.NoError(t, err, "tag should have succeeded: %s", tagName)

	taggedImage, _, err := runtime.LookupImage(tagName, lookupOptions)
	require.NoError(t, err)
	require.NotNil(t, taggedImage)

	// Both origin list and newly tagged list should point to same image id
	require.Equal(t, image.ID(), taggedImage.ID())
}

// Following test ensure that we test  Removing a manifestList
// Test tags two manifestlist and deletes one of them and
// confirms if other one is not deleted.
func TestCreateAndRemoveManifestList(t *testing.T) {
	tagName := "manifestlisttagged"
	listName := "manifestlist"
	runtime := testNewRuntime(t)
	ctx := context.Background()

	list, err := runtime.CreateManifestList(listName)
	require.NoError(t, err)
	require.NotNil(t, list)

	manifestListOpts := &ManifestListAddOptions{All: true}
	_, err = list.Add(ctx, "docker://busybox", manifestListOpts)
	require.NoError(t, err)

	lookupOptions := &LookupImageOptions{ManifestList: true}
	image, _, err := runtime.LookupImage(listName, lookupOptions)
	require.NoError(t, err)
	require.NotNil(t, image)
	err = image.Tag(tagName)
	require.NoError(t, err, "tag should have succeeded: %s", tagName)

	// Try deleting the manifestList with tag
	rmReports, rmErrors := runtime.RemoveImages(ctx, []string{tagName}, &RemoveImagesOptions{Force: true, LookupManifest: true})
	require.Nil(t, rmErrors)
	require.Equal(t, []string{"localhost/manifestlisttagged:latest"}, rmReports[0].Untagged)

	// Remove original listname as well
	rmReports, rmErrors = runtime.RemoveImages(ctx, []string{listName}, &RemoveImagesOptions{Force: true, LookupManifest: true})
	require.Nil(t, rmErrors)
	// output should contain log of untagging the original manifestlist
	require.True(t, rmReports[0].Removed)
	require.Equal(t, []string{"localhost/manifestlist:latest"}, rmReports[0].Untagged)
}

// TestAddSomeArtifacts ensures that we don't fail to add artifact manifests to
// a manifest list, even (or especially) when their config blobs aren't valid
// OCI or Docker config blobs.
func TestAddSomeArtifacts(t *testing.T) {
	listName := "manifestlist"
	runtime := testNewRuntime(t)
	ctx := context.Background()

	list, err := runtime.CreateManifestList(listName)
	require.NoError(t, err)
	require.NotNil(t, list)

	manifestListOpts := &ManifestListAddOptions{All: true}
	absPath, err := filepath.Abs(filepath.Join("..", "pkg", "manifests", "testdata", "artifacts", "blobs-only"))
	require.NoError(t, err)
	_, err = list.Add(ctx, "oci:"+absPath, manifestListOpts)
	require.NoError(t, err)

	absPath, err = filepath.Abs(filepath.Join("..", "pkg", "manifests", "testdata", "artifacts", "config-only"))
	require.NoError(t, err)
	_, err = list.Add(ctx, "oci:"+absPath, manifestListOpts)
	require.NoError(t, err)

	absPath, err = filepath.Abs(filepath.Join("..", "pkg", "manifests", "testdata", "artifacts", "no-blobs"))
	require.NoError(t, err)
	_, err = list.Add(ctx, "oci:"+absPath, manifestListOpts)
	require.NoError(t, err)
}

// TestAddArtifacts ensures that we don't fail to add artifact manifests to
// a manifest list, even (or especially) when their config blobs aren't valid
// OCI or Docker config blobs.
func TestAddArtifacts(t *testing.T) {
	listName := "manifestlist"
	ctx := context.Background()
	dir := t.TempDir()
	annotations := map[string]string{
		"a": "b",
	}
	indexAnnotations := map[string]string{
		"c": "d",
	}
	files := []struct {
		path             string
		size             int
		data             []byte
		noCompress       bool
		guessedMediaType string // what we expect, might be wrong
	}{
		{path: "first.txt", size: mathrand.Intn(256), guessedMediaType: "text/plain"},
		{path: "second.qcow2", size: 512 + mathrand.Intn(256), guessedMediaType: "application/x-qemu-disk"},
		{path: "third", size: 1024 + mathrand.Intn(256), guessedMediaType: "application/x-gzip"},
		{path: "fourth", size: 2048 + mathrand.Intn(256), noCompress: true, guessedMediaType: "application/octet-stream"},
	}
	artifacts := make([]string, 0, len(files))
	for n := range files {
		file := filepath.Join(dir, files[n].path)
		abs, err := filepath.Abs(file)
		require.NoError(t, err)
		files[n].path = abs
		if files[n].data == nil {
			buf := bytes.Buffer{}
			wc := ioutils.NopWriteCloser(&buf)
			if !files[n].noCompress {
				wc, err = compression.CompressStream(&buf, compression.Gzip, nil)
				require.NoError(t, err)
			}
			_, err = io.CopyN(wc, rand.Reader, int64(files[n].size))
			require.NoError(t, err)
			wc.Close()
			files[n].size = buf.Len()
			files[n].data = buf.Bytes()
		}
		err = os.WriteFile(abs, files[n].data, 0o600)
		require.NoError(t, err)
		artifacts = append(artifacts, abs)
	}
	artifactSubjectPath, err := filepath.Abs(filepath.Join("..", "pkg", "manifests", "testdata", "artifacts", "blobs-only"))
	require.NoError(t, err)
	artifactSubject := "oci:" + artifactSubjectPath
	indexSubjectPath, err := filepath.Abs(filepath.Join("..", "pkg", "manifests", "testdata", "artifacts", "config-only"))
	require.NoError(t, err)
	indexSubject := "oci:" + indexSubjectPath
	runtime := testNewRuntime(t)
	descriptorForSubject := func(t *testing.T, refName string) imgspecv1.Descriptor {
		if refName == "" {
			return imgspecv1.Descriptor{}
		}
		ref, err := alltransports.ParseImageName(refName)
		require.NoError(t, err)
		src, err := ref.NewImageSource(ctx, nil)
		require.NoError(t, err)
		defer src.Close()
		manifestBytes, manifestType, err := src.GetManifest(ctx, nil)
		require.NoError(t, err)
		manifestDigest, err := manifest.Digest(manifestBytes)
		require.NoError(t, err)
		artifactType := ""
		if !manifest.MIMETypeIsMultiImage(manifestType) {
			var manifestContents imgspecv1.Manifest
			require.NoError(t, json.Unmarshal(manifestBytes, &manifestContents))
			artifactType = manifestContents.ArtifactType
		}
		return imgspecv1.Descriptor{
			MediaType:    manifestType,
			ArtifactType: artifactType,
			Digest:       manifestDigest,
			Size:         int64(len(manifestBytes)),
		}
	}
	listIndex := 0
	testWith := func(t *testing.T, testName string, artifactTypeSpec string, configType string, configData string, layerType string, excludeTitles bool, artifactSubject string, artifactSubjectDescriptor imgspecv1.Descriptor, indexSubject string, indexSubjectDescriptor imgspecv1.Descriptor) {
		listIndex++
		listName := listName + strconv.Itoa(listIndex)
		t.Run(testName, func(t *testing.T) {
			var artifactType *string
			if artifactTypeSpec != "<nil>" {
				artifactType = &artifactTypeSpec
			}
			options := ManifestListAddArtifactOptions{
				Type:          artifactType,
				ConfigType:    configType,
				Config:        configData,
				LayerType:     layerType,
				ExcludeTitles: excludeTitles,
				Annotations:   annotations,
				Subject:       artifactSubject,
			}
			list, err := runtime.CreateManifestList(listName)
			require.NoError(t, err)
			require.NotNil(t, list)

			d, err := list.AddArtifact(ctx, &options, artifacts...)
			require.NoError(t, err)

			aoptions := ManifestListAnnotateOptions{
				IndexAnnotations: indexAnnotations,
				Subject:          indexSubject,
			}
			err = list.AnnotateInstance(d, &aoptions)
			require.NoError(t, err)

			destination, err := os.MkdirTemp(dir, "pushed")
			require.NoError(t, err)

			_, err = list.Push(ctx, "oci:"+destination+":tag", &ManifestListPushOptions{ImageListSelection: cp.CopyAllImages})
			require.NoError(t, err)

			ref, err := alltransports.ParseImageName("oci:" + destination + ":tag")
			require.NoError(t, err)

			src, err := ref.NewImageSource(ctx, list.image.runtime.systemContextCopy())
			require.NoError(t, err)
			indexManifest, indexType, err := src.GetManifest(ctx, nil)
			require.NoError(t, err)
			require.True(t, manifest.MIMETypeIsMultiImage(indexType))
			var index imgspecv1.Index
			require.NoError(t, json.Unmarshal(indexManifest, &index))
			// check some things in the image index
			assert.Equal(t, index.Annotations, indexAnnotations)
			if index.Subject != nil {
				assert.Equal(t, indexSubjectDescriptor, *index.Subject, "subject in index was not preserved")
			}
			for _, descriptor := range index.Manifests {
				artifactManifest, artifactManifestType, err := src.GetManifest(ctx, &descriptor.Digest)
				require.NoError(t, err)
				require.False(t, manifest.MIMETypeIsMultiImage(artifactManifestType))
				var artifact imgspecv1.Manifest
				require.NoError(t, json.Unmarshal(artifactManifest, &artifact))
				// check some things in the artifact manifest
				switch artifactTypeSpec {
				case "<nil>":
					assert.Equal(t, "application/vnd.unknown.artifact.v1", artifact.ArtifactType)
				default:
					assert.Equal(t, *artifactType, artifact.ArtifactType)
				}
				// FIXME: require.Equal(t, artifact.ArtifactType, descriptor.ArtifactType, "artifact type in index descriptor not preserved during push")
				switch configType {
				case "":
					if len(configData) > 0 {
						assert.Equal(t, imgspecv1.MediaTypeImageConfig, artifact.Config.MediaType)
					} else {
						assert.Equal(t, imgspecv1.DescriptorEmptyJSON.MediaType, artifact.Config.MediaType)
					}
				default:
					assert.Equal(t, configType, artifact.Config.MediaType)
				}
				for i, layer := range artifact.Layers {
					switch layerType {
					case "":
						var rawMediaType string
						baseName := filepath.Base(files[i].path)
						if dotIndex := strings.LastIndex(filepath.Base(files[i].path), "."); dotIndex != -1 {
							rawMediaType = mime.TypeByExtension(baseName[dotIndex:])
						} else {
							rawMediaType = http.DetectContentType(files[i].data)
						}
						parsedMediaType, _, err := mime.ParseMediaType(rawMediaType)
						require.NoError(t, err)
						assert.Equal(t, files[i].guessedMediaType, parsedMediaType)
					default:
						assert.Equal(t, layerType, layer.MediaType)
					}
					if excludeTitles {
						assert.NotContains(t, layer.Annotations, imgspecv1.AnnotationTitle)
						// FIXME: } else {
						// FIXME: require.Contains(t, layer.Annotations, imgspecv1.AnnotationTitle, "layer annotations lost during push")
						// FIXME: assert.Equal(t, filepath.Base(files[i].path), layer.Annotations[imgspecv1.AnnotationTitle], "layer annotations lost during push")
					}
					if layer.MediaType != imgspecv1.MediaTypeImageLayerGzip { // might have been (re)compressed
						assert.Equal(t, digest.FromBytes(files[i].data), layer.Digest, "layer content digest changed during push")
						assert.Equal(t, int64(len(files[i].data)), layer.Size, "layer content size changed during push")
					}
					if artifact.Subject != nil {
						assert.Equal(t, artifactSubjectDescriptor, *artifact.Subject)
					}
				}
			}
		})
	}
	for _, artifactTypeSpec := range []string{
		"<nil>",
		"",
		"application/vnd.unknown.artifact.v1",
		"application/x-something-else",
	} {
		testName := "artifactType=" + artifactTypeSpec
		for _, configType := range []string{
			"",
			imgspecv1.MediaTypeImageConfig,
			imgspecv1.DescriptorEmptyJSON.MediaType,
		} {
			testName := testName + ",configType=" + configType
			for _, configData := range []string{
				"",
				`{"a":"b"}`,
			} {
				testName := testName + ",configLength=" + strconv.Itoa(len(configData))
				for _, layerType := range []string{
					"",
					"application/octet-stream",
					imgspecv1.MediaTypeImageLayerGzip,
				} {
					testName := testName + ",layerType=" + layerType
					for _, excludeTitles := range []bool{false, true} {
						testName := testName + ",excludeTitles=" + fmt.Sprintf("%v", excludeTitles)
						for _, artifactSubject := range []string{"", artifactSubject} {
							testName := testName + ",artifactSubject="
							if artifactSubject != "" {
								testName += filepath.Base(artifactSubjectPath)
							}
							artifactSubjectDescriptor := descriptorForSubject(t, artifactSubject)
							for _, indexSubject := range []string{"", indexSubject} {
								testName := testName + ",indexSubject="
								if indexSubject != "" {
									testName += filepath.Base(indexSubjectPath)
								}
								indexSubjectDescriptor := descriptorForSubject(t, indexSubject)
								testWith(t, testName, artifactTypeSpec, configType, configData, layerType, excludeTitles, artifactSubject, artifactSubjectDescriptor, indexSubject, indexSubjectDescriptor)
							}
						}
					}
				}
			}
		}
	}
}
