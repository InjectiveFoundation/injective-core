package testwasmx

import (
	"fmt"
	"os"
)

type TestWasmX struct{}

func (t TestWasmX) GetValidWasmByteCode(relativePath *string) []byte {
	rp := "./"
	if relativePath != nil {
		rp = *relativePath
	}
	bytes, err := os.ReadFile(fmt.Sprintf("%vtest/contracts/dummy.wasm", rp))
	if err != nil {
		panic(err)
	}

	return bytes
}
