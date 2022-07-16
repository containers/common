package util

import "testing"

func TestGenerateFilterFunc(t *testing.T) {
	testValues := []string{
		"",
		"test",
		"",
	}
	type args struct {
		keys   []string
		labels []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Match when all filters the same as labels",
			args: args{
				keys:   []string{"label", "label", "label"},
				labels: testValues,
			},
			want: true,
		},
		{
			name: "Match with inverse",
			args: args{
				keys:   []string{"label", "label", "label!"},
				labels: testValues,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			for _, entry := range tt.args.keys {
				if _, err := createFilterFuncs(entry, tt.args.labels); err != nil {
					t.Errorf("createPruneFilterFuncs() failed on %s with entry %s: %s", tt.name, entry, err.Error())
				}
			}
		})
	}
}
