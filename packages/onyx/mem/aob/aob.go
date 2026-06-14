package aob

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
)

func ExactMatch(name string) []byte {
	return append(RangedMatch(name), 0x00)
}
func RangedMatch(name string) []byte {
	return []byte(name)
}

func ParseAoB(aob string) ([]byte, error) {
	var pattern []byte
	fields := strings.FieldsSeq(aob)
	for field := range fields {
		if field == "?" || field == "??" {
			pattern = append(pattern, WildcardByte)
			continue
		}
		b, err := hex.DecodeString(field)
		if err != nil || len(b) != 1 {
			return nil, fmt.Errorf("invalid hex byte %q in pattern", field)
		}
		pattern = append(pattern, b[0])
	}
	return pattern, nil
}

func NewPatternFromBytes(raw []byte) *OptimizedCompiledPattern {
	if len(raw) == 0 {
		return nil
	}
	mask := make([]byte, len(raw))
	for i := range mask {
		mask[i] = 0xFF
	}
	return &OptimizedCompiledPattern{
		FullPattern: raw,
		Mask:        mask,
		Pivot:       raw,
		PivotOffset: 0,
	}
}

func NewOptimizedPattern(aob string) (*OptimizedCompiledPattern, error) {
	raw, err := ParseAoB(aob)
	if err != nil {
		return nil, err
	}

	mask := make([]byte, len(raw))
	for i, b := range raw {
		if b == WildcardByte {
			mask[i] = 0x00
		} else {
			mask[i] = 0xFF
		}
	}

	bestStart, bestLen, curStart, curLen := 0, 0, 0, 0
	for i, b := range raw {
		if b != WildcardByte {
			curLen++
			if curLen > bestLen {
				bestLen = curLen
				bestStart = curStart
			}
		} else {
			curLen = 0
			curStart = i + 1
		}
	}

	return &OptimizedCompiledPattern{
		FullPattern: raw,
		Mask:        mask,
		Pivot:       raw[bestStart : bestStart+bestLen],
		PivotOffset: bestStart,
	}, nil
}

func (cp *OptimizedCompiledPattern) FastScan(data []byte) []uintptr {
	var results []uintptr
	patLen := len(cp.FullPattern)
	dataLen := len(data)

	if len(cp.Pivot) == 0 {
		return nil
	}

	searchOffset := 0
	for {
		idx := bytes.Index(data[searchOffset:], cp.Pivot)
		if idx == -1 {
			break
		}

		matchStart := searchOffset + idx - cp.PivotOffset
		searchOffset += idx + 1

		if matchStart < 0 || matchStart+patLen > dataLen {
			continue
		}

		matched := true
		for i := range patLen {
			if (data[matchStart+i] & cp.Mask[i]) != (cp.FullPattern[i] & cp.Mask[i]) {
				matched = false
				break
			}
		}

		if matched {
			results = append(results, uintptr(matchStart))
		}
	}
	return results
}
