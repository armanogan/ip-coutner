package IPCounter

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sort"
)

// asciNumbersToUint8 converts a slice of bytes representing a decimal number to byte.
func asciNumbersToUint8(bytes []byte) (byte, error) {
	size := len(bytes)
	if size == 0 {
		return 0, fmt.Errorf("byte too short: %s", bytes)
	}
	if size > 3 || (size == 3 && (bytes[0] > '2' || (bytes[0] == '2' && (bytes[1] > '5' || (bytes[1] == '5' && bytes[2] > '5'))))) {
		return 0, fmt.Errorf("byte too long: %s", bytes)
	}
	var num uint8
	for i := 0; i < size; i++ {
		if bytes[i] < '0' || bytes[i] > '9' {
			return 0, fmt.Errorf("wrong byte: %s", bytes)
		}
		num = num*10 + bytes[i] - '0' // Convert ASCII character to its numeric value

	}
	return num, nil
}

// ip4BytesToUint32 converts a slice of bytes representing an IPv4 address to an uint32 value.
func ip4BytesToUint32(ipBytes []byte) (uint32, error) {
	var (
		j, k  uint8
		ip32  uint32
		octet [4]byte
	)
	ipBytes = bytes.TrimSpace(ipBytes)
	length := len(ipBytes)
	for i := 0; i < length; i++ {
		if ipBytes[i] != '.' && (ipBytes[i] < '0' || ipBytes[i] > '9') {
			return 0, fmt.Errorf("incorrect ip address: %s", ipBytes)
		}
		if ipBytes[i] != '.' || i == length-1 { //46 is dot
			if k == 4 {
				j = 0
				break
			}
			octet[k] = ipBytes[i]
			k++
		}
		if ipBytes[i] == '.' || i == length-1 {
			if k > 1 && octet[0] == '0' {
				return 0, fmt.Errorf("incorrect ip address %s", ipBytes)
			}
			valUint8, err := asciNumbersToUint8(octet[:k])
			if err != nil {
				return 0, fmt.Errorf("ip4 bytes not correct in ip address %s with error: %w", ipBytes, err)
			}
			ip32 |= uint32(valUint8) << (24 - j*8) // byte to uint32 with formula 256^3,256^2,256^1,256^0
			if j > net.IPv4len {
				break
			}
			j++
			k = 0
		}
	}
	if j != net.IPv4len {
		return 0, fmt.Errorf("ip4 bytes not correct: %s", ipBytes)
	}
	return ip32, nil
}

// checkContext checks if the context has been canceled or done.
func checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return nil
}

// ip4SortCount counts unique IPv4 addresses in a sorted slice.
func ip4SortCount(ips []uint32) int64 {
	if len(ips) == 0 {
		return 0
	}
	var uniqueCount int64 = 1
	sort.Slice(ips, func(i, j int) bool {
		return ips[i] < ips[j]
	})
	for i := 1; i < len(ips); i++ {
		if ips[i] != ips[i-1] {
			uniqueCount++
		}
	}
	return uniqueCount
}
