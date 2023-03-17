# remove old gen
rm -rf client/proto-ts/gen
mkdir -p client/proto-ts/gen/cjs
mkdir -p client/proto-ts/gen/esm

# collecting proto files
mkdir -p client/proto-ts/temp
cp -r proto/injective client/proto-ts/temp/
cp -r third_party/proto/ client/proto-ts/temp/

proto_dirs=$(find ./client/proto-ts/temp -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)

# gen cjs
for dir in $proto_dirs; do
    protoc \
    -I "./client/proto-ts/temp" \
    --js_out="import_style=commonjs,binary:client/proto-ts/gen/cjs" \
    --ts_out="service=grpc-web:client/proto-ts/gen/cjs" \
    $(find "${dir}" -maxdepth 1 -name '*.proto')
done

# gen esm
for dir in $proto_dirs; do
    protoc \
    --plugin="./node_modules/.bin/protoc-gen-ts_proto" \
    --ts_proto_opt="esModuleInterop=true" \
    --ts_proto_opt="forceLong=string" \
    --ts_proto_opt="env=both" \
    --ts_proto_opt="outputClientImpl=grpc-web" \
    --ts_proto_out="client/proto-ts/gen/esm" \
    -I "./client/proto-ts/temp" \
    $(find "${dir}" -maxdepth 1 -name '*.proto')
done

# clean up
cp ./client/proto-ts/package.json.template ./client/proto-ts/gen/cjs/package.json
cp ./client/proto-ts/package.json.esm.template ./client/proto-ts/gen/esm/package.json
rm -rf client/proto-ts/temp
