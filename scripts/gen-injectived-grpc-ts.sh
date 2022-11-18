# remove old gen
rm -rf client/gen/grpc
mkdir -p client/gen/grpc/web

# collecting proto files
mkdir -p temp
cp -r proto/injective temp/
cp -r third_party/proto/ temp/

# gen
proto_dirs=$(find ./temp -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
    protoc \
    -I "temp" \
    --js_out="import_style=commonjs,binary:client/gen/grpc/web" \
    --ts_out="service=grpc-web:client/gen/grpc/web" \
    $(find "${dir}" -maxdepth 1 -name '*.proto')
done

# clean up
cp ./client/package.json.template ./client/gen/grpc/web/package.json
rm -rf temp
