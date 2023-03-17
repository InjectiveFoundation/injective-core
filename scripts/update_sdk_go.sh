# On Mac OS: brew install gnu-sed
# On Linux: change command to sed

cp injective-chain/modules/auction/types/*.go ../sdk-go/chain/auction/types/
cp injective-chain/modules/exchange/types/*.go ../sdk-go/chain/exchange/types/
cp injective-chain/modules/ocr/types/*.go ../sdk-go/chain/ocr/types/
cp injective-chain/modules/peggy/types/*.go ../sdk-go/chain/peggy/types/
cp injective-chain/modules/wasmx/types/*.go ../sdk-go/chain/wasmx/types/
cp injective-chain/modules/insurance/types/*.go ../sdk-go/chain/insurance/types/
cp injective-chain/modules/oracle/types/*.go ../sdk-go/chain/oracle/types/
cp injective-chain/modules/tokenfactory/types/*.go ../sdk-go/chain/tokenfactory/types/
cp -r proto/ ../sdk-go/proto


cd ../sdk-go/chain/auction/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/InjectiveLabs\/injective-core\/injective-/github.com\/InjectiveLabs\/sdk-go\//g" *.go

cd ../../exchange/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/InjectiveLabs\/injective-core\/injective-chain\/modules/github.com\/InjectiveLabs\/sdk-go\/chain/g" *.go

cd ../../ocr/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/InjectiveLabs\/injective-core\/injective-chain\/modules/github.com\/InjectiveLabs\/sdk-go\/chain/g" *.go

cd ../../peggy/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/InjectiveLabs\/injective-core\/injective-chain\/modules/github.com\/InjectiveLabs\/sdk-go\/chain/g" *.go

cd ../../wasmx/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/InjectiveLabs\/injective-core\/injective-chain\/modules/github.com\/InjectiveLabs\/sdk-go\/chain/g" *.go

cd ../../insurance/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/InjectiveLabs\/injective-core\/injective-chain\/modules/github.com\/InjectiveLabs\/sdk-go\/chain/g" *.go

cd ../../oracle/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/InjectiveLabs\/injective-core\/injective-chain\/modules/github.com\/InjectiveLabs\/sdk-go\/chain/g" *.go

cd ../../tokenfactory/types/
rm -f *test.go
rm -f *gw.go
gsed -i "s/github.com\/InjectiveLabs\/injective-core\/injective-chain\/modules/github.com\/InjectiveLabs\/sdk-go\/chain/g" *.go
