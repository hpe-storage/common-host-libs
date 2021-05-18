package util

import "testing"

func TestGetMacAddressHostNameMD5Hash(t *testing.T) {
	testCases := []struct {
		name        string
		macAddress1 string
		hostName1   string
		macAddress2 string
		hostName2   string
	}{
		{
			name:        "same mac address but different hostname",
			macAddress1: "0a58a9fe0001",
			macAddress2: "0a58a9fe0001",
			hostName1:   "host-dev",
			hostName2:   "host-prod",
		},
		{
			name:        "same hostname but different mac address",
			macAddress1: "0a58a9fe0001",
			macAddress2: "0a58a9fe0002",
			hostName1:   "host-dev",
			hostName2:   "host-dev",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idStr1 := GetMD5HashOfTwoStrings(tc.macAddress1, tc.hostName1)
			idStr2 := GetMD5HashOfTwoStrings(tc.macAddress2, tc.hostName2)
			if idStr1 == idStr2 {
				t.Fatalf("Expected %s to be different from %s for test %s", idStr1, idStr2, tc.name)
			}
		})
	}
}
