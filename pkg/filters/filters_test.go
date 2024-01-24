package filters

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchLabelFilters(t *testing.T) {
	testLabels := map[string]string{
		"label1": "",
		"label2": "test",
		"label3": "",
	}
	type args struct {
		filterValues []string
		labels       map[string]string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Match when all filters the same as labels",
			args: args{
				filterValues: []string{"label1", "label3", "label2=test"},
				labels:       testLabels,
			},
			want: true,
		},
		{
			name: "Match when filter value not provided in args",
			args: args{
				filterValues: []string{"label2"},
				labels:       testLabels,
			},
			want: true,
		},
		{
			name: "Match when no filter value is given",
			args: args{
				filterValues: []string{"label2="},
				labels:       testLabels,
			},
			want: true,
		},
		{
			name: "Do not match when filter value differs",
			args: args{
				filterValues: []string{"label2=differs"},
				labels:       testLabels,
			},
			want: false,
		},
		{
			name: "Do not match when filter value not listed in labels",
			args: args{
				filterValues: []string{"label1=xyz"},
				labels:       testLabels,
			},
			want: false,
		},
		{
			name: "Do not match when one from many not ok",
			args: args{
				filterValues: []string{"label1=xyz", "invalid=valid"},
				labels:       testLabels,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchLabelFilters(tt.args.filterValues, tt.args.labels); got != tt.want {
				t.Errorf("MatchLabelFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchNegatedLabelFilters(t *testing.T) {
	testLabels := map[string]string{
		"label1": "",
		"label2": "test",
		"label3": "",
	}
	type args struct {
		filterValues []string
		labels       map[string]string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Do not match when all filters the same as labels",
			args: args{
				filterValues: []string{"label1", "label3", "label2=test"},
				labels:       testLabels,
			},
			want: false,
		},
		{
			name: "Do not match when filter value not provided in args",
			args: args{
				filterValues: []string{"label2"},
				labels:       testLabels,
			},
			want: false,
		},
		{
			name: "Do not match when no filter value is given",
			args: args{
				filterValues: []string{"label2="},
				labels:       testLabels,
			},
			want: false,
		},
		{
			name: "Match when filter value differs",
			args: args{
				filterValues: []string{"label2=differs"},
				labels:       testLabels,
			},
			want: true,
		},
		{
			name: "Match when filter value not listed in labels",
			args: args{
				filterValues: []string{"label1=xyz"},
				labels:       testLabels,
			},
			want: true,
		},
		{
			name: "Match when one from many not ok",
			args: args{
				filterValues: []string{"label1=xyz", "invalid=valid"},
				labels:       testLabels,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchNegatedLabelFilters(tt.args.filterValues, tt.args.labels); got != tt.want {
				t.Errorf("MatchNegatedLabelFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeUntilTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "Return error when more values in list",
			args:    []string{"5h", "6s"},
			wantErr: true,
		},
		{
			name:    "Return error when invalid time",
			args:    []string{"invalidTime"},
			wantErr: true,
		},
		{
			name:    "Do not return error when correct time format supplied",
			args:    []string{"44m"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := ComputeUntilTimestamp(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ComputeUntilTimestamp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestFiltersFromRequest(t *testing.T) {
	// simulate https request
	req := http.Request{
		URL: &url.URL{
			RawQuery: "all=false&filters=%7B%22label%22%3A%5B%22xyz%3Dbar%22%2C%22abc%22%5D%2C%22reference%22%3A%5B%22test%22%5D%7D",
		},
	}
	// call req.ParseForm so it can parse the RawQuery data
	err := req.ParseForm()
	require.NoError(t, err)

	expectedLibpodFilters := []string{"label=xyz=bar", "label=abc", "reference=test"}
	got, err := FiltersFromRequest(&req)
	require.NoError(t, err)
	assert.ElementsMatch(t, expectedLibpodFilters, got)
}
