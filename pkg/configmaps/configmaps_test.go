package configmaps

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var drivertype = "file"

var opts map[string]string

func setup() (*ConfigMapManager, string, error) {
	testpath, err := os.MkdirTemp("", "cmdata")
	if err != nil {
		return nil, "", err
	}
	manager, err := NewManager(testpath)
	opts = map[string]string{"path": testpath}
	return manager, testpath, err
}

func cleanup(testpath string) {
	os.RemoveAll(testpath)
}

func TestAddSecretAndLookupData(t *testing.T) {
	manager, testpath, err := setup()
	require.NoError(t, err)
	defer cleanup(testpath)

	_, err = manager.Store("myconfigmap", []byte("mydata"), drivertype, opts)
	require.NoError(t, err)

	_, err = manager.lookupConfigMap("myconfigmap")
	require.NoError(t, err)

	_, data, err := manager.LookupConfigMapData("myconfigmap")
	require.NoError(t, err)
	if !bytes.Equal(data, []byte("mydata")) {
		t.Errorf("error: configmap data not equal")
	}
}

func TestAddConfigMapName(t *testing.T) {
	manager, testpath, err := setup()
	require.NoError(t, err)
	defer cleanup(testpath)

	// test one char configmap name
	_, err = manager.Store("a", []byte("mydata"), drivertype, opts)
	require.NoError(t, err)

	_, err = manager.lookupConfigMap("a")
	require.NoError(t, err)

	// name too short
	_, err = manager.Store("", []byte("mydata"), drivertype, opts)
	require.Error(t, err)
	// name too long
	_, err = manager.Store("uatqsbssrapurkuqoapubpifvsrissslzjehalxcesbhpxcvhsozlptrmngrivaiz", []byte("mydata"), drivertype, opts)
	require.Error(t, err)
	// invalid chars
	_, err = manager.Store("??", []byte("mydata"), drivertype, opts)
	require.Error(t, err)
	_, err = manager.Store("-a", []byte("mydata"), drivertype, opts)
	require.Error(t, err)
	_, err = manager.Store("a-", []byte("mydata"), drivertype, opts)
	require.Error(t, err)
	_, err = manager.Store(".a", []byte("mydata"), drivertype, opts)
	require.Error(t, err)
	_, err = manager.Store("a.", []byte("mydata"), drivertype, opts)
	require.Error(t, err)
}

func TestAddMultipleConfigMaps(t *testing.T) {
	manager, testpath, err := setup()
	require.NoError(t, err)
	defer cleanup(testpath)

	id, err := manager.Store("myconfigmap", []byte("mydata"), drivertype, opts)
	require.NoError(t, err)

	id2, err := manager.Store("myconfigmap2", []byte("mydata2"), drivertype, opts)
	require.NoError(t, err)

	configmaps, err := manager.List()
	require.NoError(t, err)
	require.Len(t, configmaps, 2)

	_, err = manager.lookupConfigMap("myconfigmap")
	require.NoError(t, err)

	_, err = manager.lookupConfigMap("myconfigmap2")
	require.NoError(t, err)

	_, data, err := manager.LookupConfigMapData(id)
	require.NoError(t, err)
	if !bytes.Equal(data, []byte("mydata")) {
		t.Errorf("error: configmap data not equal")
	}

	_, data2, err := manager.LookupConfigMapData(id2)
	require.NoError(t, err)
	if !bytes.Equal(data2, []byte("mydata2")) {
		t.Errorf("error: configmap data not equal")
	}
}

func TestAddConfigMapDupName(t *testing.T) {
	manager, testpath, err := setup()
	require.NoError(t, err)
	defer cleanup(testpath)

	_, err = manager.Store("myconfigmap", []byte("mydata"), drivertype, opts)
	require.NoError(t, err)

	_, err = manager.Store("myconfigmap", []byte("mydata"), drivertype, opts)
	require.Error(t, err)
}

func TestAddConfigMapPrefix(t *testing.T) {
	manager, testpath, err := setup()
	require.NoError(t, err)
	defer cleanup(testpath)

	// If the randomly generated configmap id is something like "abcdeiuoergnadufigh"
	// we should still allow someone to store a configmap with the name "abcd" or "a"
	configmapID, err := manager.Store("myconfigmap", []byte("mydata"), drivertype, opts)
	require.NoError(t, err)

	_, err = manager.Store(configmapID[0:5], []byte("mydata"), drivertype, opts)
	require.NoError(t, err)
}

func TestRemoveConfigMap(t *testing.T) {
	manager, testpath, err := setup()
	require.NoError(t, err)
	defer cleanup(testpath)

	_, err = manager.Store("myconfigmap", []byte("mydata"), drivertype, opts)
	require.NoError(t, err)

	_, err = manager.lookupConfigMap("myconfigmap")
	require.NoError(t, err)

	_, err = manager.Delete("myconfigmap")
	require.NoError(t, err)

	_, err = manager.lookupConfigMap("myconfigmap")
	require.Error(t, err)

	_, _, err = manager.LookupConfigMapData("myconfigmap")
	require.Error(t, err)
}

func TestRemoveConfigMapNoExist(t *testing.T) {
	manager, testpath, err := setup()
	require.NoError(t, err)
	defer cleanup(testpath)

	_, err = manager.Delete("myconfigmap")
	require.Error(t, err)
}

func TestLookupAllConfigMaps(t *testing.T) {
	manager, testpath, err := setup()
	require.NoError(t, err)
	defer cleanup(testpath)

	id, err := manager.Store("myconfigmap", []byte("mydata"), drivertype, opts)
	require.NoError(t, err)

	// inspect using configmap name
	lookup, err := manager.Lookup("myconfigmap")
	require.NoError(t, err)
	require.Equal(t, lookup.ID, id)
}

func TestInspectConfigMapId(t *testing.T) {
	manager, testpath, err := setup()
	require.NoError(t, err)
	defer cleanup(testpath)

	id, err := manager.Store("myconfigmap", []byte("mydata"), drivertype, opts)
	require.NoError(t, err)

	_, err = manager.lookupConfigMap("myconfigmap")
	require.NoError(t, err)

	// inspect using configmap id
	lookup, err := manager.Lookup(id)
	require.NoError(t, err)
	require.Equal(t, lookup.ID, id)

	// inspect using id prefix
	short := id[0:5]
	lookupshort, err := manager.Lookup(short)
	require.NoError(t, err)
	require.Equal(t, lookupshort.ID, id)
}

func TestInspectConfigMapBogus(t *testing.T) {
	manager, testpath, err := setup()
	require.NoError(t, err)
	defer cleanup(testpath)

	_, err = manager.Lookup("bogus")
	require.Error(t, err)
}

func TestConfigMapList(t *testing.T) {
	manager, testpath, err := setup()
	require.NoError(t, err)
	defer cleanup(testpath)

	_, err = manager.Store("myconfigmap", []byte("mydata"), drivertype, opts)
	require.NoError(t, err)
	_, err = manager.Store("myconfigmap2", []byte("mydata2"), drivertype, opts)
	require.NoError(t, err)

	allConfigmaps, err := manager.List()
	require.NoError(t, err)
	require.Len(t, allConfigmaps, 2)
}
