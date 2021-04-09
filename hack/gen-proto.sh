#!/bin/bash
set -o errexit
set -o pipefail
set -o nounset

# Get up to repository root
pushd "$(dirname "$0")/../"

GOGOPROTO_SERVICES=(
    "pkg/proute"
)

command -v protoc > /dev/null 2>&1 || {
    echo "protoc not detected. Please install the following:"
    echo https://github.com/protocolbuffers/protobuf/releases
    echo 'Run these outside of a gomodule directory (e.g. /tmp)'
    echo go get github.com/gogo/protobuf/{proto,protoc-gen-gogo,gogoproto,protoc-gen-gofast,protoc-gen-gogoslick}
}

for service in "${GOGOPROTO_SERVICES[@]}"; do
    regex="./${service}/*.proto"
    if compgen -G "${regex}"; then
        protoc \
            -I="${service}" \
            -I=. \
            -I="$GOPATH/src" \
            -I="$GOPATH/src/github.com/gogo/protobuf/types" \
            -I="$GOPATH/src/github.com/gogo/protobuf/gogoproto" \
            -I="$GOPATH/src/github.com/gogo/protobuf/protobuf" \
            --gogoslick_out=paths=source_relative,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,\
Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api,\
Mgoogle/protobuf/field_mask.proto=github.com/gogo/protobuf/types:\
"${service}/" \
            ./"${service}"/*.proto
            goimports -w ./"${service}"/*.pb.go
    fi
done
