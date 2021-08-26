// (c) Copyright 2018 Hewlett Packard Enterprise Development LP

package conversion

import (
	"fmt"
	"hash/fnv"
)

// ConvertGiBToMiB converts GiB into MiB
func ConvertGiBToMiB(value uint64) uint64 {
	return value * 1024
}

// ConvertMiBToGiB converts MiB into GiB
func ConvertMiBToGiB(value uint64) uint64 {
	if value != 0 {
		value = value / 1024
	}
	return value
}

// ConvertBytesToMiB converts bytes into MiB
func ConvertBytesToMiB(value uint64) uint64 {
	if value != 0 {
		value = value / (1024 * 1024)
	}
	return value
}

// ConvertBytesToGiB converts bytes into GiB
func ConvertBytesToGiB(value uint64) uint64 {
	if value != 0 {
		value = value / (1024 * 1024 * 1024)
	}
	return value
}

// GetMiBToGiBStr to convert MiB into GiB string. Ex: If value is 1.464844, then returns 1.465
func GetMiBToGiBStr(value uint64, fractionalPartSize uint8) string {
	val := (float64(value) / 1024)
	var str string
	if value%1024 != 0 {
		// Truncate the fractional value upto 'fractionalPartSize' number of digits
		str = fmt.Sprintf(fmt.Sprintf("%%.%df", fractionalPartSize), val)
	} else {
		str = fmt.Sprintf("%v", value/1024)
	}
	return str
}

// GenerateHash to convert a string value into unsigned 32-bit hash value
func GenerateHash(s *string) uint32 {
	if s == nil {
		return 0
	}
	h := fnv.New32a()
	h.Write([]byte(*s))
	return h.Sum32()
}
