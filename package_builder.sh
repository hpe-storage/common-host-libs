#!/bin/bash

# (c) Copyright 2019 Hewlett Packard Enterprise Development LP

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
    echo "Building $pkg:"
    DIR=`echo $pkg | awk -F "/" '{ s = ""; for (i = 4; i <= NF; i++) s = s $i "/"; print s }'`

    if [[ "${DIR}" == *"cmd"* ]]; then
       
        # Get the name of the binary we are building
        BIN=`echo $pkg| awk -F "/" '{print $5}'`
        
        echo 🚀 go build -o build/${BIN} -ldflags -X main.Version=${VERSION} -X main.Commit=${COMMIT} ./${DIR}
        go build -o build/${BIN} -ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT}" ./${DIR}

        if [ "${BIN}" == "dockerplugind" ] || [ "${BIN}" == "ndockeradm" ] || [ "${BIN}" == "chapid" ]; then
            echo "► Go generate ${BIN} resource.syso for Windows"
            (export GOOS=windows && go generate ./${DIR})
            echo "Building ${BIN} for Windows"
            echo ► export GOOS=windows go build -o build/${BIN}.exe -ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT}" ./${DIR}
            (export GOOS=windows && go build -o build/${BIN}.exe -ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT}" ./${DIR})
            # Remove any Windows resource.syso file created from the Windows build
            rm -f "./${DIR}/resource.syso"
        fi
    else 
        echo 🚀 go build -ldflags -X main.Version=${VERSION} -X main.Commit=${COMMIT} ./${DIR}
        go build -ldflags "-X main.Version=${VERSION} -X main.Commit=${COMMIT}" ./${DIR}
    fi

    if [ ${?} -ne 0 ]; then
        RC=1
        break
    fi
done

exit $RC
