/*
(c) Copyright 2019 Hewlett Packard Enterprise Development LP
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package model

import (
	"testing"
)

var argTests = []struct {
	name    string
	input   string
	results []string
}{
	{"nothing", "", nil},
	{"normal -a -b -c", "-a -b  -c", []string{"-a", "-b", "-c"}},
	{"some extra spaces", "-a -b  -c", []string{"-a", "-b", "-c"}},
	{"some extra spaces early", " -a -b  -c", []string{"-a", "-b", "-c"}},
	{"some extra spaces late", " -a -b  -c ", []string{"-a", "-b", "-c"}},
	{"danger ;", ";init 0", []string{"init", "0"}},
	{"danger &", "&init 0", []string{"init", "0"}},
	{"danger &&", "&&init 0", []string{"init", "0"}},
	{"danger ||", "||init 0", []string{"init", "0"}},
	{"xfs example", "-m crc=1 -K -i maxpct=0 -d agsize=9999999999999", []string{"-m", "crc=1", "-K", "-i", "maxpct=0", "-d", "agsize=9999999999999"}},
	{"ext example", "-E stride=16,stripe-width=64 -c", []string{"-E", "stride=16,stripe-width=64", "-c"}},
}

func TestGetFilesystemOpts(t *testing.T) {
	for _, tc := range argTests {
		t.Run(tc.name, func(t *testing.T) {
			fsopts := FilesystemOpts{"", "", "", tc.input}
			safe := fsopts.GetCreateOpts()
			if tc.results == nil && safe == nil {
				return
			}
			for i := range tc.results {
				if safe[i] != tc.results[i] {
					t.Error("For", tc.name, "expected", tc.results[i], "got", safe[i], "expected array", tc.results, "returned array", safe)
				}
			}
		})
	}

}
