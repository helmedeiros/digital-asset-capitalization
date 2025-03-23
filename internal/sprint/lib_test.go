package assetcap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructArrayToCSV(t *testing.T) {
	tests := []struct {
		name    string
		data    []map[string]interface{}
		headers []string
		want    string
		wantErr bool
	}{
		{
			name:    "empty data",
			data:    []map[string]interface{}{},
			headers: []string{},
			want:    "",
		},
		{
			name: "header not found",
			data: []map[string]interface{}{
				{
					"Name": "John",
					"Age":  30,
				},
			},
			headers: []string{"hello"},
			wantErr: true,
		},
		{
			name: "single row",
			data: []map[string]interface{}{
				{
					"Name": "John",
					"Age":  30,
				},
			},
			headers: []string{"Name", "Age"},
			want:    "Name,Age\nJohn,30\n",
		},
		{
			name: "multiple rows",
			data: []map[string]interface{}{
				{
					"Name": "John",
					"Age":  30,
				},
				{
					"Name": "Jane",
					"Age":  25,
				},
			},
			headers: []string{"Name", "Age"},
			want:    "Name,Age\nJohn,30\nJane,25\n",
		},
		{
			name: "multiple rows swaped headers",
			data: []map[string]interface{}{
				{
					"Name": "John",
					"Age":  30,
				},
				{
					"Name": "Jane",
					"Age":  25,
				},
			},
			headers: []string{"Age", "Name"},
			want:    "Age,Name\n30,John\n25,Jane\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StructArrayToCSVOrdered(tt.data, tt.headers)
			if tt.wantErr {
				assert.Error(t, err, "Expected error but got none")
				return
			}
			require.NoError(t, err, "Unexpected error")
			assert.Equal(t, tt.want, got, "CSV output mismatch")
		})
	}
}
