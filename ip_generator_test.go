package main

import "testing"

func TestIPGenerator_Next(t *testing.T) {
	ig := NewIPGenerator()
	tests := []struct {
		name  string
		ig    *IPGenerator
		want  int
		want1 string
	}{
		{"first", ig, 1, "127.0.0.1"},
		{"second", ig, 2, "127.0.0.2"},
		{"third", ig, 3, "127.0.0.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.ig.Next()
			if got != tt.want {
				t.Errorf("IPGenerator.Next() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("IPGenerator.Next() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}

	ig.id = 253
	ig.ip = []byte{0, 0, 253}

	tests = []struct {
		name  string
		ig    *IPGenerator
		want  int
		want1 string
	}{
		{"before", ig, 254, "127.0.0.254"},
		{"overlap", ig, 255, "127.0.1.1"},
		{"continue", ig, 256, "127.0.1.2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.ig.Next()
			if got != tt.want {
				t.Errorf("IPGenerator.Next() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("IPGenerator.Next() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
