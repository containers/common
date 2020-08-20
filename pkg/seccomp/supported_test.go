package seccomp

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var statusFile = `
Name:   bash
Umask:  0022
State:  S (sleeping)
Tgid:   17248
Ngid:   0
Pid:    17248
PPid:   17200
TracerPid:      0
Uid:    1000    1000    1000    1000
Gid:    100     100     100     100
FDSize: 256
Groups: 16 33 100
NStgid: 17248
NSpid:  17248
NSpgid: 17248
NSsid:  17200
VmPeak:     131168 kB
VmSize:     131168 kB
VmLck:           0 kB
VmPin:           0 kB
VmHWM:       13484 kB
VmRSS:       13484 kB
RssAnon:     10264 kB
RssFile:      3220 kB
RssShmem:        0 kB
VmData:      10332 kB
VmStk:         136 kB
VmExe:         992 kB
VmLib:        2104 kB
VmPTE:          76 kB
VmPMD:          12 kB
VmSwap:          0 kB
HugetlbPages:          0 kB        # 4.4
Threads:        1
SigQ:   0/3067
SigPnd: 0000000000000000
ShdPnd: 0000000000000000
SigBlk: 0000000000010000
SigIgn: 0000000000384004
SigCgt: 000000004b813efb
CapInh: 0000000000000000
CapPrm: 0000000000000000
CapEff: 0000000000000000
CapBnd: ffffffffffffffff
CapAmb:   0000000000000000
NoNewPrivs:     0
Seccomp:        0
Cpus_allowed:   00000001
Cpus_allowed_list:      0
Mems_allowed:   1
Mems_allowed_list:      0
voluntary_ctxt_switches:        150
nonvoluntary_ctxt_switches:     545
`

func TestParseStatusFile(t *testing.T) {
	for _, tc := range []struct {
		getFilePath func() (string, func())
		shouldErr   bool
		expected    map[string]string
	}{
		{ // success
			getFilePath: func() (string, func()) {
				tempFile, err := ioutil.TempFile("", "parse-status-file-")
				require.Nil(t, err)

				// Valid entry
				_, err = tempFile.WriteString("Seccomp:   0\n")
				require.Nil(t, err)

				// Unparsable entry
				_, err = tempFile.WriteString("wrong")
				require.Nil(t, err)

				return tempFile.Name(), func() {
					require.Nil(t, os.RemoveAll(tempFile.Name()))
				}
			},
			shouldErr: false,
			expected:  map[string]string{"Seccomp": "0"},
		},
		{ // success whole file
			getFilePath: func() (string, func()) {
				tempFile, err := ioutil.TempFile("", "parse-status-file-")
				require.Nil(t, err)

				_, err = tempFile.WriteString(statusFile)
				require.Nil(t, err)

				return tempFile.Name(), func() {
					require.Nil(t, os.RemoveAll(tempFile.Name()))
				}
			},
			shouldErr: false,
			expected: map[string]string{
				"CapAmb":                     "0000000000000000",
				"CapBnd":                     "ffffffffffffffff",
				"CapEff":                     "0000000000000000",
				"CapInh":                     "0000000000000000",
				"CapPrm":                     "0000000000000000",
				"Cpus_allowed":               "00000001",
				"Cpus_allowed_list":          "0",
				"FDSize":                     "256",
				"Gid":                        "100     100     100     100",
				"Groups":                     "16 33 100",
				"HugetlbPages":               "0 kB        # 4.4",
				"Mems_allowed":               "1",
				"Mems_allowed_list":          "0",
				"NSpgid":                     "17248",
				"NSpid":                      "17248",
				"NSsid":                      "17200",
				"NStgid":                     "17248",
				"Name":                       "bash",
				"Ngid":                       "0",
				"NoNewPrivs":                 "0",
				"PPid":                       "17200",
				"Pid":                        "17248",
				"RssAnon":                    "10264 kB",
				"RssFile":                    "3220 kB",
				"RssShmem":                   "0 kB",
				"Seccomp":                    "0",
				"ShdPnd":                     "0000000000000000",
				"SigBlk":                     "0000000000010000",
				"SigCgt":                     "000000004b813efb",
				"SigIgn":                     "0000000000384004",
				"SigPnd":                     "0000000000000000",
				"SigQ":                       "0/3067",
				"State":                      "S (sleeping)",
				"Tgid":                       "17248",
				"Threads":                    "1",
				"TracerPid":                  "0",
				"Uid":                        "1000    1000    1000    1000",
				"Umask":                      "0022",
				"VmData":                     "10332 kB",
				"VmExe":                      "992 kB",
				"VmHWM":                      "13484 kB",
				"VmLck":                      "0 kB",
				"VmLib":                      "2104 kB",
				"VmPMD":                      "12 kB",
				"VmPTE":                      "76 kB",
				"VmPeak":                     "131168 kB",
				"VmPin":                      "0 kB",
				"VmRSS":                      "13484 kB",
				"VmSize":                     "131168 kB",
				"VmStk":                      "136 kB",
				"VmSwap":                     "0 kB",
				"nonvoluntary_ctxt_switches": "545",
				"voluntary_ctxt_switches":    "150",
			},
		},
		{ // error opening file
			getFilePath: func() (string, func()) {
				tempFile, err := ioutil.TempFile("", "parse-status-file-")
				require.Nil(t, err)

				require.Nil(t, os.RemoveAll(tempFile.Name()))

				return tempFile.Name(), func() {}
			},
			shouldErr: true,
		},
	} {
		filePath, cleanup := tc.getFilePath()
		defer cleanup()
		res, err := parseStatusFile(filePath)
		if tc.shouldErr {
			require.NotNil(t, err)
		} else {
			require.Equal(t, tc.expected, res)
		}
	}
}
