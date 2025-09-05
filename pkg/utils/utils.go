package utils

import (
	"net"
	"strconv"
	"strings"
)

// InterfaceToFloat64 converts various interface types to float64
func InterfaceToFloat64(n interface{}) (float64, error) {
	switch x := n.(type) {
	case string:
		return strconv.ParseFloat(x, 64)
	case float32:
		return float64(x), nil
	case float64:
		return x, nil
	case int64:
		return float64(x), nil
	case int32:
		return float64(x), nil
	case int:
		return float64(x), nil
	case uint64:
		return float64(x), nil
	case uint32:
		return float64(x), nil
	case uint:
		return float64(x), nil
	default:
		return 0.0, nil
	}
}

// SubNetMaskToLen converts subnet mask to CIDR notation length
func SubNetMaskToLen(netmask string) (int, error) {
	ipSplitArr := strings.Split(netmask, ".")
	if len(ipSplitArr) != 4 {
		return 0, netmaskError(netmask)
	}
	
	ipv4MaskArr := make([]byte, 4)
	for i, value := range ipSplitArr {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return 0, conversionError("strconv.Atoi", value, err)
		}
		if intValue > 255 {
			return 0, invalidNetmaskError(value)
		}
		ipv4MaskArr[i] = byte(intValue)
	}

	ones, _ := net.IPv4Mask(ipv4MaskArr[0], ipv4MaskArr[1], ipv4MaskArr[2], ipv4MaskArr[3]).Size()
	return ones, nil
}

// ParseCPUFrequency parses CPU frequency string to MHz
func ParseCPUFrequency(freqStr string) float64 {
	switch {
	case strings.HasSuffix(freqStr, "GHz"):
		if value, err := strconv.ParseFloat(strings.TrimSuffix(freqStr, "GHz"), 64); err == nil {
			return value * 1000
		}
	case strings.HasSuffix(freqStr, "MHz"):
		if value, err := strconv.ParseFloat(strings.TrimSuffix(freqStr, "MHz"), 64); err == nil {
			return value
		}
	}
	return 0.0
}

// ParseMemorySize parses memory size string to MB
func ParseMemorySize(memStr string) float64 {
	if value, err := strconv.ParseFloat(strings.TrimSuffix(memStr, "MB"), 64); err == nil {
		return value
	}
	return 0.0
}

// Helper functions for error messages
func netmaskError(netmask string) error {
	return netmaskFormatError(netmask)
}

func netmaskFormatError(netmask string) error {
	return invalidNetmaskPatternError(netmask)
}

func invalidNetmaskPatternError(netmask string) error {
	return &netmaskValidationError{
		message:  "invalid netmask format",
		netmask:  netmask,
		expected: "255.255.255.0 pattern",
	}
}

type netmaskValidationError struct {
	message  string
	netmask  string
	expected string
}

func (e *netmaskValidationError) Error() string {
	return e.message
}

func conversionError(operation, value string, err error) error {
	return &conversionErrorType{
		operation: operation,
		value:     value,
		err:       err,
	}
}

type conversionErrorType struct {
	operation string
	value     string
	err       error
}

func (e *conversionErrorType) Error() string {
	return e.operation + " error: " + e.err.Error()
}

func (e *conversionErrorType) Unwrap() error {
	return e.err
}

func invalidNetmaskError(value string) error {
	return &netmaskRangeError{
		message: "netmask value exceeds maximum",
		value:   value,
		max:     255,
	}
}

type netmaskRangeError struct {
	message string
	value   string
	max     int
}

func (e *netmaskRangeError) Error() string {
	return e.message
}