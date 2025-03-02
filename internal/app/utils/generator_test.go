package utils

import "testing"

func TestCreateShortKey(t *testing.T) {
	tests := []struct {
		name       string
		wantLength int
	}{
		{
			name:       "TestCreateShortKeyBigValue",
			wantLength: 17,
		},
		{
			name:       "TestCreateShortKeySmallValue",
			wantLength: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateShortKey(tt.wantLength); len(got) != tt.wantLength {
				t.Errorf("createShortKey() = %v, want lenght %v", got, tt.wantLength)
			}
		})
	}
}
