// (c) Copyright 2021 Hewlett Packard Enterprise Development LP

package util

import "testing"

func TestGetMacAddressHostNameMD5Hash(t *testing.T) {
	testCases := []struct {
		name        string
		macAddress1 string
		hostName1   string
		macAddress2 string
		hostName2   string
		expectSame  bool
	}{
		{
			name:        "same mac address but different hostname",
			macAddress1: "0a58a9fe0001",
			macAddress2: "0a58a9fe0001",
			hostName1:   "host-dev",
			hostName2:   "host-prod",
			expectSame:  false,
		},
		{
			name:        "same hostname but different mac address",
			macAddress1: "0a58a9fe0001",
			macAddress2: "0a58a9fe0002",
			hostName1:   "host-dev",
			hostName2:   "host-dev",
			expectSame:  false,
		},
		{
			name:        "same hostname and same mac address",
			macAddress1: "0a58a9fe0001",
			macAddress2: "0a58a9fe0001",
			hostName1:   "host-dev",
			hostName2:   "host-dev",
			expectSame:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idStr1 := GetMD5HashOfTwoStrings(tc.macAddress1, tc.hostName1)
			idStr2 := GetMD5HashOfTwoStrings(tc.macAddress2, tc.hostName2)
			if idStr1 == idStr2 && !tc.expectSame {
				t.Fatalf("Expected %s to be different from %s for test %s", idStr1, idStr2, tc.name)
			}
			if idStr1 != idStr2 && tc.expectSame {
				t.Fatalf("Expected %s to be same as %s for test %s", idStr1, idStr2, tc.name)
			}
		})
	}
}

func TestSanitizeIPAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// IPv4 tests
		{
			name:     "Clean IPv4 address",
			input:    "192.168.1.10",
			expected: "192.168.1.10",
		},
		{
			name:     "IPv4 with trailing asterisk",
			input:    "172.28.66.88*",
			expected: "172.28.66.88",
		},
		{
			name:     "IPv4 with leading asterisk",
			input:    "*192.168.1.1",
			expected: "192.168.1.1",
		},
		{
			name:     "IPv4 with spaces",
			input:    " 10.0.0.1 ",
			expected: "10.0.0.1",
		},
		{
			name:     "IPv4 with multiple invalid characters",
			input:    "192.168*.1@.10#",
			expected: "192.168.1.10",
		},
		// IPv6 tests
		{
			name:     "Clean IPv6 address",
			input:    "2001:db8::1",
			expected: "2001:db8::1",
		},
		{
			name:     "IPv6 with trailing asterisk",
			input:    "2001:db8:b000::15*",
			expected: "2001:db8:b000::15",
		},
		{
			name:     "IPv6 full address",
			input:    "2001:0db8:0000:0000:0000:0000:0000:0001",
			expected: "2001:0db8:0000:0000:0000:0000:0000:0001",
		},
		{
			name:     "IPv6 with brackets (for port notation)",
			input:    "[2001:db8::1]",
			expected: "2001:db8::1",
		},
		{
			name:     "IPv6 with invalid characters",
			input:    "fe80::1%eth0",
			expected: "fe80::1e0",
		},
		{
			name:     "IPv6 loopback",
			input:    "::1",
			expected: "::1",
		},
		{
			name:     "IPv6 with multiple asterisks",
			input:    "**2001:db8::a**",
			expected: "2001:db8::a",
		},

		{
			name:     "IP with tabs and newlines",
			input:    "192.168.1.10\t\n",
			expected: "192.168.1.10",
		},
		{
			name:     "Uppercase hex in IPv6",
			input:    "2001:DB8:ABCD::1",
			expected: "2001:DB8:ABCD::1",
		},
		{
			name:     "Mixed case hex in IPv6",
			input:    "2001:dB8:AbCd::1*",
			expected: "2001:dB8:AbCd::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeIPAddress(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeIPAddress(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
