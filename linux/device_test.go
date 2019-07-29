package linux

import (
	"testing"
)

var parseHctlTests = []struct {
	name      string
	lunString string
	host      string
	channel   string
	target    string
	lun       string
	err       bool
}{
	{"parse test/lun0", "Host: scsi7 Channel: 00 Id: 00 Lun: 00", "7", "0", "0", "0", false},
	{"parse test/multi digits", "Host: scsi10 Channel: 20 Id: 30 Lun: 255", "10", "20", "30", "255", false},
	{"parse test/begin with space", " Host: scsi7 Channel: 00 Id: 00 Lun: 00", "7", "0", "0", "0", false},
	{"parse test/no match", "not a valid lun string", "", "", "", "", true},
}

func TestParseHctl(t *testing.T) {
	for _, tc := range parseHctlTests {
		t.Run(tc.name, func(t *testing.T) {
			host, channel, target, lun, err := parseHctl(tc.lunString)
			if (err != nil) != tc.err {
				t.Error(
					"For:", tc.name,
					"expected:", "No error",
					"got:", err,
				)
			}
			if host != tc.host {
				t.Error(
					"For:", tc.name,
					"expected host:", tc.host,
					"got host:", host,
				)
			}
			if channel != tc.channel {
				t.Error(
					"For:", tc.name,
					"expected channel:", tc.channel,
					"got channel:", channel,
				)
			}
			if target != tc.target {
				t.Error(
					"For:", tc.name,
					"expected target:", tc.target,
					"got target:", target,
				)
			}
			if lun != tc.lun {
				t.Error(
					"For:", tc.name,
					"expected lun:", tc.lun,
					"got lun:", lun,
				)
			}
		})
	}
}
