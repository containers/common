package manifests

import (
	"os"
	"reflect"
	"testing"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/storage/pkg/reexec"
	digest "github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
)

const (
	expectedInstance = digest.Digest("sha256:c829b1810d2dbb456e74a695fd3847530c8319e5a95dca623e9f1b1b89020d8b")
	ociFixture       = "testdata/fedora.index.json"
	dockerFixture    = "testdata/fedora.list.json"
)

var _ List = &list{}

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

func TestCreate(t *testing.T) {
	list := Create()
	if list == nil {
		t.Fatalf("error creating an empty list")
	}
}

func TestFromBlob(t *testing.T) {
	for _, version := range []string{
		ociFixture,
		dockerFixture,
	} {
		bytes, err := os.ReadFile(version)
		if err != nil {
			t.Fatalf("error loading %s: %v", version, err)
		}
		list, err := FromBlob(bytes)
		if err != nil {
			t.Fatalf("error parsing %s: %v", version, err)
		}
		if len(list.Docker().Manifests) != len(list.OCIv1().Manifests) {
			t.Fatalf("%s: expected the same number of manifests, but %d != %d", version, len(list.Docker().Manifests), len(list.OCIv1().Manifests))
		}
		for i := range list.Docker().Manifests {
			d := list.Docker().Manifests[i]
			o := list.OCIv1().Manifests[i]
			if d.Platform.OS != o.Platform.OS {
				t.Fatalf("%s: expected the same OS", version)
			}
			if d.Platform.Architecture != o.Platform.Architecture {
				t.Fatalf("%s: expected the same Architecture", version)
			}
		}
	}
}

func TestAddInstance(t *testing.T) {
	manifestBytes, err := os.ReadFile("testdata/fedora-minimal.schema2.json")
	if err != nil {
		t.Fatalf("error loading testdata/fedora-minimal.schema2.json: %v", err)
	}
	manifestType := manifest.GuessMIMEType(manifestBytes)
	manifestDigest, err := manifest.Digest(manifestBytes)
	if err != nil {
		t.Fatalf("error digesting testdata/fedora-minimal.schema2.json: %v", err)
	}
	for _, version := range []string{
		ociFixture,
		dockerFixture,
	} {
		bytes, err := os.ReadFile(version)
		if err != nil {
			t.Fatalf("error loading %s: %v", version, err)
		}
		list, err := FromBlob(bytes)
		if err != nil {
			t.Fatalf("error parsing %s: %v", version, err)
		}
		annotations := []string{"A=B", "C=D"}
		expectedAnnotations := map[string]string{"A": "B", "C": "D"}
		if err = list.AddInstance(manifestDigest, int64(len(manifestBytes)), manifestType, "linux", "amd64", "", nil, "", nil, annotations); err != nil {
			t.Fatalf("adding an instance failed in %s: %v", version, err)
		}
		if d, err := list.findDocker(manifestDigest); d == nil || err != nil {
			t.Fatalf("adding an instance failed in %s: %v", version, err)
		}
		if o, err := list.findOCIv1(manifestDigest); o == nil || err != nil || !maps.Equal(o.Annotations, expectedAnnotations) {
			t.Fatalf("adding an instance failed in %s (annotations=%#v): %v", version, o.Annotations, err)
		}

		if list, err = FromBlob(bytes); err != nil {
			t.Fatalf("error parsing %s: %v", version, err)
		}
		if err = list.AddInstance(manifestDigest, int64(len(manifestBytes)), manifestType, "", "", "", nil, "", nil, nil); err != nil {
			t.Fatalf("adding an instance without platform info failed in %s: %v", version, err)
		}
		o, err := list.findOCIv1(manifestDigest)
		if o == nil || err != nil {
			t.Fatalf("adding an instance failed in %s: %v", version, err)
		}
		if o.Platform != nil {
			t.Fatalf("added a Platform field (%+v) where none was expected: %v", o.Platform, err)
		}
	}
}

func TestRemove(t *testing.T) {
	bytes, err := os.ReadFile(ociFixture)
	if err != nil {
		t.Fatalf("error loading blob: %v", err)
	}
	list, err := FromBlob(bytes)
	if err != nil {
		t.Fatalf("error parsing blob: %v", err)
	}
	before := len(list.OCIv1().Manifests)
	instanceDigest := expectedInstance
	if d, err := list.findDocker(instanceDigest); d == nil || err != nil {
		t.Fatalf("finding expected instance failed: %v", err)
	}
	if o, err := list.findOCIv1(instanceDigest); o == nil || err != nil {
		t.Fatalf("finding expected instance failed: %v", err)
	}
	err = list.Remove(instanceDigest)
	if err != nil {
		t.Fatalf("error parsing blob: %v", err)
	}
	after := len(list.Docker().Manifests)
	if after != before-1 {
		t.Fatalf("removing instance should have succeeded")
	}
	if d, err := list.findDocker(instanceDigest); d != nil || err == nil {
		t.Fatalf("finding instance should have failed")
	}
	if o, err := list.findOCIv1(instanceDigest); o != nil || err == nil {
		t.Fatalf("finding instance should have failed")
	}
}

func testString(t *testing.T, values []string, set func(List, digest.Digest, string) error, get func(List, digest.Digest) (string, error)) {
	bytes, err := os.ReadFile(ociFixture)
	if err != nil {
		t.Fatalf("error loading blob: %v", err)
	}
	list, err := FromBlob(bytes)
	if err != nil {
		t.Fatalf("error parsing blob: %v", err)
	}
	for _, testString := range values {
		if err = set(list, expectedInstance, testString); err != nil {
			t.Fatalf("error setting %q: %v", testString, err)
		}
		b, err := list.Serialize("")
		if err != nil {
			t.Fatalf("error serializing list: %v", err)
		}
		list, err := FromBlob(b)
		if err != nil {
			t.Fatalf("error parsing list: %v", err)
		}
		value, err := get(list, expectedInstance)
		if err != nil {
			t.Fatalf("error retrieving value %q: %v", testString, err)
		}
		if value != testString {
			t.Fatalf("expected value %q, got %q: %v", value, testString, err)
		}
	}
}

func testStringSlice(t *testing.T, values [][]string, set func(List, digest.Digest, []string) error, get func(List, digest.Digest) ([]string, error)) {
	bytes, err := os.ReadFile(ociFixture)
	if err != nil {
		t.Fatalf("error loading blob: %v", err)
	}
	list, err := FromBlob(bytes)
	if err != nil {
		t.Fatalf("error parsing blob: %v", err)
	}
	for _, testSlice := range values {
		if err = set(list, expectedInstance, testSlice); err != nil {
			t.Fatalf("error setting %v: %v", testSlice, err)
		}
		b, err := list.Serialize("")
		if err != nil {
			t.Fatalf("error serializing list: %v", err)
		}
		list, err := FromBlob(b)
		if err != nil {
			t.Fatalf("error parsing list: %v", err)
		}
		values, err := get(list, expectedInstance)
		if err != nil {
			t.Fatalf("error retrieving value %v: %v", testSlice, err)
		}
		if !reflect.DeepEqual(values, testSlice) && (len(values) != len(testSlice) || len(testSlice) != 0) {
			t.Fatalf("expected values %v, got %v: %v", testSlice, values, err)
		}
	}
}

func testMap(t *testing.T, values []map[string]string, set func(List, *digest.Digest, map[string]string) error, clear func(List, *digest.Digest) error, get func(List, *digest.Digest) (map[string]string, error)) {
	bytes, err := os.ReadFile(ociFixture)
	if err != nil {
		t.Fatalf("error loading blob: %v", err)
	}
	list, err := FromBlob(bytes)
	if err != nil {
		t.Fatalf("error parsing blob: %v", err)
	}
	instance := expectedInstance
	for _, instanceDigest := range []*digest.Digest{nil, &instance} {
		for _, testMap := range values {
			if err = clear(list, instanceDigest); err != nil {
				t.Fatalf("error clearing %v: %v", testMap, err)
			}
			if err = set(list, instanceDigest, testMap); err != nil {
				t.Fatalf("error setting %v: %v", testMap, err)
			}
			b, err := list.Serialize("")
			if err != nil {
				t.Fatalf("error serializing list: %v", err)
			}
			list, err := FromBlob(b)
			if err != nil {
				t.Fatalf("error parsing list: %v", err)
			}
			values, err := get(list, instanceDigest)
			if err != nil {
				t.Fatalf("error retrieving value %v: %v", testMap, err)
			}
			if len(values) != len(testMap) {
				t.Fatalf("expected %d map entries, got %d", len(testMap), len(values))
			}
			for k, v := range testMap {
				if values[k] != v {
					t.Fatalf("expected map value %q=%q, got %q", k, v, values[k])
				}
			}
			if err = clear(list, instanceDigest); err != nil {
				t.Fatalf("error clearing %v: %v", testMap, err)
			}
			values, err = get(list, instanceDigest)
			if err != nil {
				t.Fatalf("error retrieving value %v: %v", testMap, err)
			}
			if len(values) != 0 {
				t.Fatalf("expected %d map entries, got %d", 0, len(values))
			}
		}
	}
}

func TestAnnotations(t *testing.T) {
	testMap(t,
		[]map[string]string{{"A": "B", "C": "D"}, {"E": "F", "G": "H"}},
		func(l List, i *digest.Digest, m map[string]string) error {
			return l.SetAnnotations(i, m)
		},
		func(l List, i *digest.Digest) error {
			return l.ClearAnnotations(i)
		},
		func(l List, i *digest.Digest) (map[string]string, error) {
			return l.Annotations(i)
		},
	)
}

func TestArchitecture(t *testing.T) {
	testString(t,
		[]string{"", "abacus", "sliderule"},
		func(l List, i digest.Digest, s string) error {
			return l.SetArchitecture(i, s)
		},
		func(l List, i digest.Digest) (string, error) {
			return l.Architecture(i)
		},
	)
}

func TestFeatures(t *testing.T) {
	testStringSlice(t,
		[][]string{nil, {"chrome", "hubcaps"}, {"climate", "control"}},
		func(l List, i digest.Digest, s []string) error {
			return l.SetFeatures(i, s)
		},
		func(l List, i digest.Digest) ([]string, error) {
			return l.Features(i)
		},
	)
}

func TestOS(t *testing.T) {
	testString(t,
		[]string{"", "linux", "darwin"},
		func(l List, i digest.Digest, s string) error {
			return l.SetOS(i, s)
		},
		func(l List, i digest.Digest) (string, error) {
			return l.OS(i)
		},
	)
}

func TestOSFeatures(t *testing.T) {
	testStringSlice(t,
		[][]string{nil, {"ipv6", "containers"}, {"nested", "virtualization"}},
		func(l List, i digest.Digest, s []string) error {
			return l.SetOSFeatures(i, s)
		},
		func(l List, i digest.Digest) ([]string, error) {
			return l.OSFeatures(i)
		},
	)
}

func TestOSVersion(t *testing.T) {
	testString(t,
		[]string{"el7", "el8"},
		func(l List, i digest.Digest, s string) error {
			return l.SetOSVersion(i, s)
		},
		func(l List, i digest.Digest) (string, error) {
			return l.OSVersion(i)
		},
	)
}

func TestURLs(t *testing.T) {
	testStringSlice(t,
		[][]string{{"https://example.com", "https://example.net"}, {"http://example.com", "http://example.net"}},
		func(l List, i digest.Digest, s []string) error {
			return l.SetURLs(i, s)
		},
		func(l List, i digest.Digest) ([]string, error) {
			return l.URLs(i)
		},
	)
}

func TestVariant(t *testing.T) {
	testString(t,
		[]string{"", "workstation", "cloud", "server"},
		func(l List, i digest.Digest, s string) error {
			return l.SetVariant(i, s)
		},
		func(l List, i digest.Digest) (string, error) {
			return l.Variant(i)
		},
	)
}

func TestSerialize(t *testing.T) {
	for _, version := range []string{
		ociFixture,
		dockerFixture,
	} {
		bytes, err := os.ReadFile(version)
		if err != nil {
			t.Fatalf("error loading %s: %v", version, err)
		}
		list, err := FromBlob(bytes)
		if err != nil {
			t.Fatalf("error parsing %s: %v", version, err)
		}
		for _, mimeType := range []string{"", v1.MediaTypeImageIndex, manifest.DockerV2ListMediaType} {
			b, err := list.Serialize(mimeType)
			if err != nil {
				t.Fatalf("error serializing %s with type %q: %v", version, mimeType, err)
			}
			l, err := FromBlob(b)
			if err != nil {
				t.Fatalf("error parsing %s re-encoded as %q: %v\n%s", version, mimeType, err, string(b))
			}
			if !reflect.DeepEqual(list.Docker().Manifests, l.Docker().Manifests) {
				t.Fatalf("re-encoded %s as %q was different\n%#v\n%#v", version, mimeType, list, l)
			}
			for i := range list.OCIv1().Manifests {
				manifest := list.OCIv1().Manifests[i]
				m := l.OCIv1().Manifests[i]
				if manifest.Digest != m.Digest ||
					manifest.MediaType != m.MediaType ||
					manifest.Size != m.Size ||
					!reflect.DeepEqual(list.OCIv1().Manifests[i].Platform, l.OCIv1().Manifests[i].Platform) {
					t.Fatalf("re-encoded %s OCI %d as %q was different\n%#v\n%#v", version, i, mimeType, list, l)
				}
			}
		}
	}
}

func TestMediaType(t *testing.T) {
	testString(t,
		[]string{v1.MediaTypeImageManifest, manifest.DockerV2Schema2MediaType},
		func(l List, i digest.Digest, s string) error {
			return l.SetMediaType(i, s)
		},
		func(l List, i digest.Digest) (string, error) {
			return l.MediaType(i)
		},
	)
}

func TestArtifactType(t *testing.T) {
	testString(t,
		[]string{"text/plain", "application/octet-stream"},
		func(l List, i digest.Digest, s string) error {
			return l.SetArtifactType(&i, s)
		},
		func(l List, i digest.Digest) (string, error) {
			return l.ArtifactType(&i)
		},
	)
	testString(t,
		[]string{"text/plain", "application/octet-stream"},
		func(l List, i digest.Digest, s string) error {
			return l.SetArtifactType(nil, s)
		},
		func(l List, i digest.Digest) (string, error) {
			return l.ArtifactType(nil)
		},
	)
}

func TestPlatform(t *testing.T) {
	bytes, err := os.ReadFile(ociFixture)
	if err != nil {
		t.Fatalf("error loading blob: %v", err)
	}
	list, err := FromBlob(bytes)
	if err != nil {
		t.Fatalf("error parsing blob: %v", err)
	}
	instanceDigest, err := digest.Parse("sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a")
	if err != nil {
		t.Fatalf("error parsing digest: %v", err)
	}
	// we only want to be dealing with one manifest, so ensure that there
	// aren't going to be any yet
	for len(list.OCIv1().Manifests) > 0 {
		assert.NoError(t, list.Remove(list.OCIv1().Manifests[0].Digest))
	}
	// add an instance without platform info
	assert.NoError(t, list.AddInstance(instanceDigest, 2, v1.MediaTypeImageManifest, "", "", "", nil, "", nil, nil))
	assert.Nil(t, list.OCIv1().Manifests[0].Platform, "expected platform to be nil")

	assert.NoError(t, list.SetOS(instanceDigest, "os"))
	assert.NotNil(t, list.OCIv1().Manifests[0].Platform, "expected platform to be set")
	assert.NoError(t, list.SetOS(instanceDigest, ""))
	assert.Nil(t, list.OCIv1().Manifests[0].Platform, "expected platform to be nil")

	assert.NoError(t, list.SetArchitecture(instanceDigest, "archy"))
	assert.NotNil(t, list.OCIv1().Manifests[0].Platform, "expected platform to be set")
	assert.NoError(t, list.SetArchitecture(instanceDigest, ""))
	assert.Nil(t, list.OCIv1().Manifests[0].Platform, "expected platform to be nil")

	assert.NoError(t, list.SetVariant(instanceDigest, "very"))
	assert.NotNil(t, list.OCIv1().Manifests[0].Platform, "expected platform to be set")
	assert.NoError(t, list.SetVariant(instanceDigest, ""))
	assert.Nil(t, list.OCIv1().Manifests[0].Platform, "expected platform to be nil")

	assert.NoError(t, list.SetOSFeatures(instanceDigest, []string{"so", "featureful"}))
	assert.NotNil(t, list.OCIv1().Manifests[0].Platform, "expected platform to be set")
	assert.NoError(t, list.SetOSFeatures(instanceDigest, []string{}))
	assert.Nil(t, list.OCIv1().Manifests[0].Platform, "expected platform to be nil")
}

func TestSubject(t *testing.T) {
	bytes, err := os.ReadFile(ociFixture)
	if err != nil {
		t.Fatalf("error loading blob: %v", err)
	}
	list, err := FromBlob(bytes)
	if err != nil {
		t.Fatalf("error parsing blob: %v", err)
	}
	for _, wrote := range []*v1.Descriptor{
		nil,
		{},
		{
			MediaType: v1.MediaTypeImageManifest,
			Digest:    expectedInstance,
			Size:      1234,
		},
	} {
		err := list.SetSubject(wrote)
		if err != nil {
			t.Fatalf("error setting subject: %v", err)
		}
		b, err := list.Serialize("")
		if err != nil {
			t.Fatalf("error serializing list: %v", err)
		}
		list, err = FromBlob(b)
		if err != nil {
			t.Fatalf("error parsing list: %v", err)
		}
		read, err := list.Subject()
		if err != nil {
			t.Fatalf("error retrieving subject: %v", err)
		}
		if !reflect.DeepEqual(read, wrote) {
			t.Fatalf("expected subject %v, got %v", wrote, read)
		}
	}
}
