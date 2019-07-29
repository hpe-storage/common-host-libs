#!/bin/bash

# (c) Copyright 2018 Hewlett Packard Enterprise Development LP

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

# http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

RC=0
for pkg in $@; do
    echo "Testing $pkg:"
    DIR=`echo $pkg| awk -F "/" '{ s = ""; for (i = 4; i <= NF; i++) s = s $i "/"; print s }'`

    echo 🚀 go test -cover ./${DIR}
    go test -cover ./${DIR}

    if [ ${?} -ne 0 ]; then
        RC=1
        break
    fi
done

exit $RC
