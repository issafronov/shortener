package main

import "testing"

func Test_runServer(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runServer(); (err != nil) != tt.wantErr {
				t.Errorf("runServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
