package types_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/containers/common/libnetwork/types"
)

func TestUnmarshalMacAddress(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    types.HardwareAddr
		wantErr bool
	}{
		{
			name: "mac as string with colon",
			json: `"52:54:00:1c:2e:46"`,
			want: types.HardwareAddr{0x52, 0x54, 0x00, 0x1c, 0x2e, 0x46},
		},
		{
			name: "mac as string with dash",
			json: `"52-54-00-1c-2e-46"`,
			want: types.HardwareAddr{0x52, 0x54, 0x00, 0x1c, 0x2e, 0x46},
		},
		{
			name: "mac as byte array",
			json: `[82, 84, 0, 28, 46, 70]`,
			want: types.HardwareAddr{0x52, 0x54, 0x00, 0x1c, 0x2e, 0x46},
		},
		{
			name: "null value",
			json: `null`,
			want: nil,
		},
		{
			name: "mac as base64",
			json: `"qrvM3e7/"`,
			want: types.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		},
		{
			name:    "invalid string",
			json:    `"52:54:00:1c:2e`,
			wantErr: true,
		},
		{
			name:    "invalid array",
			json:    `[82, 84, 0, 28, 46`,
			wantErr: true,
		},

		{
			name:    "invalid value",
			json:    `ab`,
			wantErr: true,
		},
		{
			name:    "invalid object",
			json:    `{}`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			mac := types.HardwareAddr{}
			err := json.Unmarshal([]byte(test.json), &mac)
			if (err != nil) != test.wantErr {
				t.Errorf("types.HardwareAddress Unmarshal() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if test.wantErr {
				return
			}
			if !reflect.DeepEqual(mac, test.want) {
				t.Errorf("types.HardwareAddress Unmarshal() got = %v, want %v", mac, test.want)
			}
		})
	}
}

func TestMarshalMacAddress(t *testing.T) {
	tests := []struct {
		name    string
		arg     types.HardwareAddr
		want    string
		wantErr bool
	}{
		{
			name: "marshal mac",
			arg:  types.HardwareAddr{0x44, 0x33, 0x22, 0x44, 0x33, 0x22},
			want: `"44:33:22:44:33:22"`,
		},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			g, err := json.Marshal(test.arg)
			got := string(g)
			if (err != nil) != test.wantErr {
				t.Errorf("types.HardwareAddress Marshal() (got = %v) error = %v, wantErr %v", got, err, test.wantErr)
				return
			}
			if test.wantErr {
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("types.HardwareAddress Marshal() got = %v, want %v", got, test.want)
			}
		})
	}
}
