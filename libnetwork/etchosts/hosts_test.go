package etchosts

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

const baseFileContent1Spaces = `127.0.0.1 localhost localhost.localdomain localhost4 localhost4.localdomain4
::1 localhost localhost.localdomain localhost6 localhost6.localdomain6
`

const baseFileContent1Tabs = `127.0.0.1	localhost	localhost.localdomain	localhost4	localhost4.localdomain4
::1	localhost	localhost.localdomain	localhost6	localhost6.localdomain6
`

const baseFileContent1Mixed = `127.0.0.1	localhost localhost.localdomain localhost4 localhost4.localdomain4
::1	localhost localhost.localdomain localhost6 localhost6.localdomain6
`

const targetFileContent1 = `127.0.0.1	localhost localhost.localdomain localhost4 localhost4.localdomain4
::1	localhost localhost.localdomain localhost6 localhost6.localdomain6
`

const baseFileContent2 = `127.0.0.1	localhost
::1	localhost
1.1.1.1	name1
2.2.2.2	name2
`

const targetFileContent2 = `127.0.0.1	localhost
::1	localhost
1.1.1.1	name1
2.2.2.2	name2
`

const baseFileContent3Comments1 = `127.0.0.1	localhost #localhost
::1	localhost
# with comments
`

const baseFileContent3Comments2 = `#localhost
`

const targetFileContent3 = `127.0.0.1	localhost
::1	localhost
`

const baseFileContent4 = `127.0.0.1	localhost
`

const targetFileContent4 = `1.1.1.1	name1
2.2.2.2	name2
127.0.0.1	localhost
`

const targetFileContent5 = `1.1.1.1	name1
2.2.2.2	name2
127.0.1.1	localhost
`

const baseFileContent6 = `127.0.0.1	localhost
::1	localhost
1.1.1.1 host.containers.internal
`

const targetFileContent6 = `127.0.0.1	localhost
::1	localhost
1.1.1.1	host.containers.internal
`

const baseFileContent7 = `
1.1.1.1
`

const targetFileContent7 = `127.0.0.1	localhost
::1	localhost
`

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		// only used to trigger fails for not existing files
		baseFileName              string
		baseFileContent           string
		noWriteBaseFile           bool
		extraHosts                []string
		containerIPs              HostEntries
		hostContainersInternal    string
		expectedTargetFileContent string
		wantErrString             string
	}{
		{
			name:                      "with spaces",
			baseFileContent:           baseFileContent1Spaces,
			expectedTargetFileContent: targetFileContent1,
		},
		{
			name:                      "with tabs",
			baseFileContent:           baseFileContent1Tabs,
			expectedTargetFileContent: targetFileContent1,
		},
		{
			name:                      "with spaces and tabs",
			baseFileContent:           baseFileContent1Mixed,
			expectedTargetFileContent: targetFileContent1,
		},
		{
			name:                      "with more entries",
			baseFileContent:           baseFileContent2,
			expectedTargetFileContent: targetFileContent2,
		},
		{
			name:                      "with no entries",
			baseFileContent:           "",
			expectedTargetFileContent: targetFileContent3,
		},
		{
			name:                      "base file is empty",
			baseFileContent:           "",
			noWriteBaseFile:           true,
			expectedTargetFileContent: targetFileContent3,
		},
		{
			name:                      "with comments 1",
			baseFileContent:           baseFileContent3Comments1,
			expectedTargetFileContent: targetFileContent3,
		},
		{
			name:                      "with comments 2",
			baseFileContent:           baseFileContent3Comments2,
			expectedTargetFileContent: targetFileContent3,
		},
		{
			name:                      "extra hosts",
			baseFileContent:           baseFileContent4,
			extraHosts:                []string{"name1:1.1.1.1", "name2:2.2.2.2"},
			expectedTargetFileContent: targetFileContent4,
		},
		{
			name:                      "extra hosts with localhost",
			baseFileContent:           "",
			extraHosts:                []string{"name1:1.1.1.1", "name2:2.2.2.2", "localhost:127.0.1.1"},
			expectedTargetFileContent: targetFileContent5,
		},
		{
			name:                      "with more entries and extra host",
			baseFileContent:           baseFileContent2,
			extraHosts:                []string{"name1:1.1.1.1"},
			expectedTargetFileContent: "1.1.1.1\tname1\n" + targetFileContent2,
		},
		{
			name:                      "with more entries and extra host",
			baseFileContent:           baseFileContent2,
			extraHosts:                []string{"name1:1.1.1.1"},
			expectedTargetFileContent: "1.1.1.1\tname1\n" + targetFileContent2,
		},
		{
			name:                      "with more entries and extra host",
			baseFileContent:           baseFileContent2,
			extraHosts:                []string{"name1:1.1.1.1"},
			expectedTargetFileContent: "1.1.1.1\tname1\n" + targetFileContent2,
		},
		{
			name:                      "container ips",
			baseFileContent:           baseFileContent1Spaces,
			containerIPs:              []HostEntry{{IP: "1.2.3.4", Names: []string{"conname", "hostname"}}},
			expectedTargetFileContent: targetFileContent1 + "1.2.3.4\tconname hostname\n",
		},
		{
			name:            "container ips 2",
			baseFileContent: baseFileContent1Spaces,
			containerIPs: []HostEntry{
				{IP: "1.2.3.4", Names: []string{"conname", "hostname"}},
				{IP: "fd::1", Names: []string{"conname", "hostname"}},
			},
			expectedTargetFileContent: targetFileContent1 + "1.2.3.4\tconname hostname\nfd::1\tconname hostname\n",
		},
		{
			name:                      "container ips and extra hosts",
			baseFileContent:           baseFileContent1Spaces,
			extraHosts:                []string{"name1:1.1.1.1"},
			containerIPs:              []HostEntry{{IP: "1.2.3.4", Names: []string{"conname", "hostname"}}},
			expectedTargetFileContent: "1.1.1.1\tname1\n" + targetFileContent1 + "1.2.3.4\tconname hostname\n",
		},
		{
			name:                      "container ips and extra hosts 2",
			baseFileContent:           baseFileContent2,
			extraHosts:                []string{"name1:1.1.1.1"},
			containerIPs:              []HostEntry{{IP: "1.2.3.4", Names: []string{"conname", "hostname"}}},
			expectedTargetFileContent: "1.1.1.1\tname1\n" + targetFileContent2 + "1.2.3.4\tconname hostname\n",
		},
		{
			name:                      "container ip name is not added when name is already present",
			baseFileContent:           baseFileContent2,
			containerIPs:              []HostEntry{{IP: "1.2.3.4", Names: []string{"name1", "hostname"}}},
			expectedTargetFileContent: targetFileContent2 + "1.2.3.4\thostname\n",
		},
		{
			name:                      "container ip name is not added when name is already present 2",
			baseFileContent:           baseFileContent2,
			containerIPs:              []HostEntry{{IP: "1.2.3.4", Names: []string{"name1"}}},
			expectedTargetFileContent: targetFileContent2,
		},
		{
			name:                      "container ip name is not added when name is already present in extra hosts",
			baseFileContent:           baseFileContent1Spaces,
			extraHosts:                []string{"somename:1.1.1.1"},
			containerIPs:              []HostEntry{{IP: "1.2.3.4", Names: []string{"somename", "hostname"}}},
			expectedTargetFileContent: "1.1.1.1\tsomename\n" + targetFileContent1 + "1.2.3.4\thostname\n",
		},
		{
			name:                      "with host.containers.internal ip",
			baseFileContent:           baseFileContent1Spaces,
			hostContainersInternal:    "10.0.0.1",
			expectedTargetFileContent: targetFileContent1 + "10.0.0.1\thost.containers.internal\n",
		},
		{
			name:                      "host.containers.internal not added when already present in extra hosts",
			baseFileContent:           baseFileContent1Spaces,
			extraHosts:                []string{"host.containers.internal:1.1.1.1"},
			hostContainersInternal:    "10.0.0.1",
			expectedTargetFileContent: "1.1.1.1\thost.containers.internal\n" + targetFileContent1,
		},
		{
			name:                      "host.containers.internal not added when already present in base hosts",
			baseFileContent:           baseFileContent6,
			hostContainersInternal:    "10.0.0.1",
			expectedTargetFileContent: targetFileContent6,
		},
		{
			name:                      "invalid hosts content",
			baseFileContent:           baseFileContent7,
			expectedTargetFileContent: targetFileContent7,
		},
		// errors
		{
			name:            "base file does not exists",
			baseFileName:    "does/not/exists123456789",
			noWriteBaseFile: true,
			wantErrString:   "no such file or directory",
		},
		{
			name:            "invalid extra hosts hostname empty",
			baseFileContent: baseFileContent1Spaces,
			extraHosts:      []string{":1.1.1.1"},
			wantErrString:   "hostname in host entry \":1.1.1.1\" is empty",
		},
		{
			name:            "invalid extra hosts empty ip",
			baseFileContent: baseFileContent1Spaces,
			extraHosts:      []string{"name:"},
			wantErrString:   "IP address in host entry \"name:\" is empty",
		},
		{
			name:            "invalid extra hosts empty ip",
			baseFileContent: baseFileContent1Spaces,
			extraHosts:      []string{"name:"},
			wantErrString:   "IP address in host entry \"name:\" is empty",
		},
		{
			name:            "invalid extra hosts format",
			baseFileContent: baseFileContent1Spaces,
			extraHosts:      []string{"name"},
			wantErrString:   "unable to parse host entry \"name\": incorrect format",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			baseHostFile := tt.baseFileName
			if !tt.noWriteBaseFile {
				f, err := ioutil.TempFile(t.TempDir(), "basehosts")
				assert.NoErrorf(t, err, "failed to create base host file: %v", err)
				defer f.Close()
				baseHostFile = f.Name()
				_, err = f.WriteString(tt.baseFileContent)
				assert.NoError(t, err, "failed to write base host file: %v", err)
			}

			targetFile := filepath.Join(t.TempDir(), "target")

			params := &Params{
				BaseFile:                 baseHostFile,
				ExtraHosts:               tt.extraHosts,
				ContainerIPs:             tt.containerIPs,
				HostContainersInternalIP: tt.hostContainersInternal,
				TargetFile:               targetFile,
			}

			err := New(params)
			if tt.wantErrString != "" {
				assert.ErrorContains(t, err, tt.wantErrString)
				return
			}
			assert.NoError(t, err, "New() failed")

			content, err := ioutil.ReadFile(targetFile)
			assert.NoErrorf(t, err, "failed to read target host file: %v", err)
			assert.Equal(t, tt.expectedTargetFileContent, string(content), "check hosts content")
		})
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name                      string
		baseFileContent           string
		entries                   HostEntries
		expectedTargetFileContent string
		wantErrString             string
	}{
		{
			name:                      "add entry",
			baseFileContent:           baseFileContent1Mixed,
			entries:                   HostEntries{{IP: "1.1.1.1", Names: []string{"name1", "name2"}}},
			expectedTargetFileContent: targetFileContent1 + "1.1.1.1\tname1 name2\n",
		},
		{
			name:            "add two entries",
			baseFileContent: baseFileContent1Mixed,
			entries: HostEntries{
				{IP: "1.1.1.1", Names: []string{"name1", "name2"}},
				{IP: "1.1.1.2", Names: []string{"name3", "name4"}},
			},
			expectedTargetFileContent: targetFileContent1 + "1.1.1.1\tname1 name2\n1.1.1.2\tname3 name4\n",
		},
		{
			name:                      "add entry to empty file",
			baseFileContent:           "",
			entries:                   HostEntries{{IP: "1.1.1.1", Names: []string{"name1", "name2"}}},
			expectedTargetFileContent: "1.1.1.1\tname1 name2\n",
		},
		{
			name:                      "add entry which already exists",
			baseFileContent:           baseFileContent2,
			entries:                   HostEntries{{IP: "1.1.1.1", Names: []string{"name1", "name2"}}},
			expectedTargetFileContent: targetFileContent2,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f, err := ioutil.TempFile(t.TempDir(), "hosts")
			assert.NoErrorf(t, err, "failed to create base host file: %v", err)
			defer f.Close()
			hostFile := f.Name()
			_, err = f.WriteString(tt.baseFileContent)
			assert.NoError(t, err, "failed to write base host file: %v", err)

			var st unix.Stat_t
			err = unix.Stat(hostFile, &st)
			assert.NoError(t, err, "stat host file: %v", err)

			err = Add(hostFile, tt.entries)
			if tt.wantErrString != "" {
				assert.ErrorContains(t, err, tt.wantErrString)
				return
			}
			assert.NoError(t, err, "Add() failed")

			content, err := ioutil.ReadFile(hostFile)
			assert.NoErrorf(t, err, "failed to read host file: %v", err)
			assert.Equal(t, tt.expectedTargetFileContent, string(content), "check hosts content")

			var st2 unix.Stat_t
			err = unix.Stat(hostFile, &st2)
			assert.NoError(t, err, "stat host file: %v", err)
			assert.Equal(t, st.Ino, st2.Ino, "inode before and after Add() must match")
		})
	}
}

func TestAddIfExists(t *testing.T) {
	tests := []struct {
		name                      string
		baseFileContent           string
		existsEntries             HostEntries
		newEntries                HostEntries
		expectedTargetFileContent string
		wantErrString             string
	}{
		{
			name:                      "add entry",
			baseFileContent:           baseFileContent1Mixed,
			newEntries:                HostEntries{{IP: "1.1.1.1", Names: []string{"name1", "name2"}}},
			expectedTargetFileContent: targetFileContent1 + "1.1.1.1\tname1 name2\n",
		},
		{
			name:                      "add entry with existing entries match",
			baseFileContent:           baseFileContent1Mixed,
			existsEntries:             HostEntries{{IP: "::1", Names: []string{"localhost"}}},
			newEntries:                HostEntries{{IP: "1.1.1.1", Names: []string{"name1", "name2"}}},
			expectedTargetFileContent: targetFileContent1 + "1.1.1.1\tname1 name2\n",
		},
		{
			name:                      "existing entries with no match should not add",
			baseFileContent:           baseFileContent1Mixed,
			existsEntries:             HostEntries{{IP: "::1", Names: []string{"name"}}},
			newEntries:                HostEntries{{IP: "1.1.1.1", Names: []string{"name1", "name2"}}},
			expectedTargetFileContent: targetFileContent1,
		},
		{
			name:            "add two entries",
			baseFileContent: baseFileContent1Mixed,
			newEntries: HostEntries{
				{IP: "1.1.1.1", Names: []string{"name1", "name2"}},
				{IP: "1.1.1.2", Names: []string{"name3", "name4"}},
			},
			expectedTargetFileContent: targetFileContent1 + "1.1.1.1\tname1 name2\n1.1.1.2\tname3 name4\n",
		},
		{
			name:            "add two entries with existing entries match",
			baseFileContent: baseFileContent1Mixed,
			existsEntries:   HostEntries{{IP: "127.0.0.1", Names: []string{"localhost"}}},
			newEntries: HostEntries{
				{IP: "1.1.1.1", Names: []string{"name1", "name2"}},
				{IP: "1.1.1.2", Names: []string{"name3", "name4"}},
			},
			expectedTargetFileContent: targetFileContent1 + "1.1.1.1\tname1 name2\n1.1.1.2\tname3 name4\n",
		},
		{
			name:                      "add entry to empty file",
			baseFileContent:           "",
			newEntries:                HostEntries{{IP: "1.1.1.1", Names: []string{"name1", "name2"}}},
			expectedTargetFileContent: "1.1.1.1\tname1 name2\n",
		},
		{
			name:                      "add entry to empty file with no existing match",
			baseFileContent:           "",
			existsEntries:             HostEntries{{IP: "127.0.0.1", Names: []string{"localhost"}}},
			newEntries:                HostEntries{{IP: "1.1.1.1", Names: []string{"name1", "name2"}}},
			expectedTargetFileContent: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f, err := ioutil.TempFile(t.TempDir(), "hosts")
			assert.NoErrorf(t, err, "failed to create base host file: %v", err)
			defer f.Close()
			hostFile := f.Name()
			_, err = f.WriteString(tt.baseFileContent)
			assert.NoError(t, err, "failed to write base host file: %v", err)

			var st unix.Stat_t
			err = unix.Stat(hostFile, &st)
			assert.NoError(t, err, "stat host file: %v", err)

			err = AddIfExists(hostFile, tt.existsEntries, tt.newEntries)
			if tt.wantErrString != "" {
				assert.ErrorContains(t, err, tt.wantErrString)
				return
			}
			assert.NoError(t, err, "AddIfExists() failed")

			content, err := ioutil.ReadFile(hostFile)
			assert.NoErrorf(t, err, "failed to read host file: %v", err)
			assert.Equal(t, tt.expectedTargetFileContent, string(content), "check hosts content")

			var st2 unix.Stat_t
			err = unix.Stat(hostFile, &st2)
			assert.NoError(t, err, "stat host file: %v", err)
			assert.Equal(t, st.Ino, st2.Ino, "inode before and after AddIfExists() must match")
		})
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name                      string
		baseFileContent           string
		entries                   HostEntries
		expectedTargetFileContent string
	}{
		{
			name:                      "remove entry which does not exists",
			baseFileContent:           baseFileContent1Spaces,
			entries:                   HostEntries{{IP: "1.1.1.1", Names: []string{"name1", "name2"}}},
			expectedTargetFileContent: targetFileContent1,
		},
		{
			name:                      "do not remove entry when only ip matches",
			baseFileContent:           baseFileContent2,
			entries:                   HostEntries{{IP: "1.1.1.1", Names: []string{"new1", "new2"}}},
			expectedTargetFileContent: targetFileContent2,
		},
		{
			name:            "remove two entries",
			baseFileContent: baseFileContent2,
			entries: HostEntries{
				{IP: "1.1.1.1", Names: []string{"name1"}},
				{IP: "2.2.2.2", Names: []string{"name2", "name4"}},
			},
			expectedTargetFileContent: targetFileContent3,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f, err := ioutil.TempFile(t.TempDir(), "hosts")
			assert.NoErrorf(t, err, "failed to create base host file: %v", err)
			defer f.Close()
			hostFile := f.Name()
			_, err = f.WriteString(tt.baseFileContent)
			assert.NoError(t, err, "failed to write base host file: %v", err)

			var st unix.Stat_t
			err = unix.Stat(hostFile, &st)
			assert.NoError(t, err, "stat host file: %v", err)

			err = Remove(hostFile, tt.entries)
			assert.NoError(t, err, "Remove() failed")

			content, err := ioutil.ReadFile(hostFile)
			assert.NoErrorf(t, err, "failed to read host file: %v", err)
			assert.Equal(t, tt.expectedTargetFileContent, string(content), "check hosts content")

			var st2 unix.Stat_t
			err = unix.Stat(hostFile, &st2)
			assert.NoError(t, err, "stat host file: %v", err)
			assert.Equal(t, st.Ino, st2.Ino, "inode before and after Remove() must match")
		})
	}
}
