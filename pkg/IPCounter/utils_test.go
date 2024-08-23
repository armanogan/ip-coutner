package IPCounter

import (
	"context"
	"testing"
	"time"
)

func TestAsciNumbersToUint8(t *testing.T) {
	tests := []struct {
		name    string
		byte    []byte
		want    byte
		wantErr bool
	}{
		{
			name:    "Valid single digit 0",
			byte:    []byte("0"),
			want:    0,
			wantErr: false,
		},
		{
			name:    "Valid single digit 9",
			byte:    []byte("9"),
			want:    9,
			wantErr: false,
		},
		{
			name:    "Valid two-digit number 10",
			byte:    []byte("10"),
			want:    10,
			wantErr: false,
		},
		{
			name:    "Valid two-digit number 99",
			byte:    []byte("99"),
			want:    99,
			wantErr: false,
		},
		{
			name:    "Valid three-digit number 255",
			byte:    []byte("255"),
			want:    255,
			wantErr: false,
		},
		{
			name:    "Valid three-digit number with leading zeros 007",
			byte:    []byte("007"),
			want:    7,
			wantErr: false,
		},
		{
			name:    "Invalid number with four digits",
			byte:    []byte("1000"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "Invalid number with non-numeric characters",
			byte:    []byte("12a"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "Invalid number with empty input",
			byte:    []byte(""),
			want:    0,
			wantErr: true,
		},
		{
			name:    "Invalid number with mixed characters",
			byte:    []byte("1a2"),
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := asciNumbersToUint8(tt.byte)
			if (err != nil) != tt.wantErr {
				t.Errorf("asciNumbersToUint8() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("asciNumbersToUint8() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIp4BytesToUint32 tests the ip4BytesToUint32 function.
func TestIp4BytesToUint32(t *testing.T) {
	tests := []struct {
		name    string
		ipBytes []byte
		want    uint32
		wantErr bool
	}{

		{
			name:    "Valid IP address 10.0.0.255",
			ipBytes: []byte("10.0.0.255"),
			want:    167772415, // 10.0.0.255 in uint32
			wantErr: false,
		},
		{
			name:    "Valid IP address 1.1.1.1",
			ipBytes: []byte("1.1.1.1"),
			want:    16843009, // 1.1.1.1 in uint32
			wantErr: false,
		},
		{
			name:    "Valid IP address 255.255.255.255",
			ipBytes: []byte("255.255.255.255"),
			want:    4294967295, // 255.255.255.255 in uint32
			wantErr: false,
		},
		{
			name:    "Invalid IP address with missing octets",
			ipBytes: []byte("192.168.1"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "Invalid IP address with too many octets",
			ipBytes: []byte("192.168.1.1.1"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "Invalid IP address with non-numeric characters",
			ipBytes: []byte("192.168.abc.1"),
			want:    0,
			wantErr: true,
		},

		{
			name:    "Empty input",
			ipBytes: []byte(""),
			want:    0,
			wantErr: true,
		},
		{
			name:    "IP address with extra spaces",
			ipBytes: []byte(" 192.168.0.1 "),
			want:    3232235521, // 192.168.0.1 in uint32
			wantErr: false,
		},
		{
			name:    "IP address with embedded whitespace",
			ipBytes: []byte("192. 168.0.1"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "IP address with dot at the end",
			ipBytes: []byte("192.168.0.1."),
			want:    0,
			wantErr: true,
		},
		{
			name:    "IP address with dot at the start",
			ipBytes: []byte(".192.168.0.1"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "IP address with octets out of range",
			ipBytes: []byte("256.0.0.0"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "IP address with octets out of range (high)",
			ipBytes: []byte("0.0.0.256"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "IP address with valid max octets but with leading zero",
			ipBytes: []byte("0.0.0.001"),
			want:    0,
			wantErr: true,
		},
		{
			name:    "IP address with valid max octets but with spaces",
			ipBytes: []byte("0.0.0.255 "),
			want:    255,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ip4BytesToUint32(tt.ipBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("ip4BytesToUint32() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ip4BytesToUint32() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCheckContext tests the checkContext function.
func TestCheckContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantErr bool
	}{
		{
			name:    "Context not canceled",
			ctx:     context.Background(),
			wantErr: false,
		},
		{
			name: "Context canceled",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			wantErr: true,
		},
		{
			name: "Context with deadline exceeded",
			ctx: func() context.Context {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
				cancel()
				return ctx
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkContext(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIp4SortCount tests the ip4SortCount function.
func TestIp4SortCount(t *testing.T) {
	// Define test cases with input slices and expected results.
	tests := []struct {
		ips           []uint32
		expectedCount int64
	}{
		{[]uint32{1, 1, 1, 1}, 1},                   // All IPs are the same
		{[]uint32{1, 2, 3, 4}, 4},                   // All IPs are unique
		{[]uint32{1, 2, 2, 3, 3, 3, 4, 4, 4, 4}, 4}, // Some duplicate IPs
		{[]uint32{}, 0},                             // Empty slice
		{[]uint32{1}, 1},                            // Single IP
		{[]uint32{3, 1, 4, 2, 5}, 5},                // Unsorted unique IPs
		{[]uint32{5, 5, 5, 5, 5, 5, 5, 5, 5}, 1},    // All IPs are the same (repeated)
	}

	// Loop through each test case
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			// Call the function with the test input
			count := ip4SortCount(tt.ips)

			// Check if the result is correct
			if count != tt.expectedCount {
				t.Errorf("ip4SortCount(%v) = %d; want %d", tt.ips, count, tt.expectedCount)
			}
		})
	}
}
