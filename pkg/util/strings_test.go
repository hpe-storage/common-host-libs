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
