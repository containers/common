package util

import (
	"reflect"
	"testing"
)

func TestParseMTU(t *testing.T) {
	type args struct {
		mtuOption string
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "mtu default",
			args: args{
				mtuOption: "",
			},
			want: 0,
		},
		{
			name: "mtu 1500",
			args: args{
				mtuOption: "1500",
			},
			want: 1500,
		},
		{
			name: "mtu string",
			args: args{
				mtuOption: "thousand-fifty-hundred",
			},
			wantErr: true,
		},
		{
			name: "mtu less than 0",
			args: args{
				mtuOption: "-1",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMTU(tt.args.mtuOption)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMTU() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseMTU() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseVlan(t *testing.T) {
	type args struct {
		vlanOption string
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "vlan default",
			args: args{
				vlanOption: "",
			},
			want: 0,
		},
		{
			name: "vlan 0",
			args: args{
				vlanOption: "0",
			},
			want: 0,
		},
		{
			name: "vlan less than 0",
			args: args{
				vlanOption: "-1",
			},
			wantErr: true,
		},
		{
			name: "vlan 4094",
			args: args{
				vlanOption: "4094",
			},
			want: 4094,
		},
		{
			name: "vlan greater than 4094",
			args: args{
				vlanOption: "4095",
			},
			wantErr: true,
		},
		{
			name: "vlan string",
			args: args{
				vlanOption: "thousand-fifty-hundred",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVlan(tt.args.vlanOption)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVlan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseVlan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseIsolate(t *testing.T) {
	type args struct {
		isolateOption string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "isolate default",
			args: args{
				isolateOption: "",
			},
			want: "false",
		},
		{
			name: "isolate true",
			args: args{
				isolateOption: "true",
			},
			want: "true",
		},
		{
			name: "isolate 1",
			args: args{
				isolateOption: "1",
			},
			want: "true",
		},
		{
			name: "isolate greater than 1",
			args: args{
				isolateOption: "2",
			},
			wantErr: true,
		},
		{
			name: "isolate false",
			args: args{
				isolateOption: "false",
			},
			want: "false",
		},
		{
			name: "isolate 0",
			args: args{
				isolateOption: "0",
			},
			want: "false",
		},
		{
			name: "isolate less than 0",
			args: args{
				isolateOption: "-1",
			},
			wantErr: true,
		},
		{
			name: "isolate strict",
			args: args{
				isolateOption: "strict",
			},
			want: "strict",
		},
		{
			name: "isolate unknown value",
			args: args{
				isolateOption: "foobar",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIsolate(tt.args.isolateOption)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIsolate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseIsolate() = %v, want %v", got, tt.want)
			}
		})
	}
}
