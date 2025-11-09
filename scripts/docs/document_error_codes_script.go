// Package main provides a tool to generate JSON documentation for all registered error codes
// in the Injective Core application. It extracts error information from the cosmos-sdk
// error registry and generates separate JSON files for each codespace.
//
// Usage:
//
// Run with default settings (generates files in current directory):
//
//	go run scripts/docs/document_error_codes_script.go
//
// Run with custom destination folder:
//
//	go run scripts/docs/document_error_codes_script.go -dest /path/to/output/folder
//
// Show help information:
//
//	go run scripts/docs/document_error_codes_script.go -h
//
// Build and run as executable:
//
//	go build -o error_docs_generator scripts/docs/document_error_codes_script.go
//	./error_docs_generator -dest ./docs/errors/
//
// The script generates one JSON file per codespace, named "<codespace>_errors.json".
// Each file contains an array of error objects with module_name, error_code, and description.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	sdkerrors "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	simapp "github.com/InjectiveLabs/injective-core/injective-chain/app"
)

const (
	// Default values
	defaultDestFolder = "."
	defaultFileMode   = 0o644
	defaultDirMode    = 0o755

	// Template strings
	tempDirPrefix = "injective-core-test"
	filenameFmt   = "%s_errors.json"
	jsonIndent    = "  "
)

// ErrorInfo represents the structure of error information in JSON output.
type ErrorInfo struct {
	ModuleName  string `json:"module_name"`
	ErrorCode   uint32 `json:"error_code"`
	Description string `json:"description"`
}

// Config holds the configuration for the error documentation generator.
type Config struct {
	DestFolder string
}

// parseFlags parses command line arguments and returns the configuration.
func parseFlags() *Config {
	var destFolder string
	flag.StringVar(&destFolder, "dest", defaultDestFolder, "Destination folder for generated JSON files (default: current directory)")
	flag.Parse()

	return &Config{
		DestFolder: destFolder,
	}
}

// setupApp initializes the Injective application for error extraction.
func setupApp() (*simapp.InjectiveApp, func(), error) {
	tempDir, err := os.MkdirTemp("", tempDirPrefix)
	if err != nil {
		if fileError := os.RemoveAll(tempDir); fileError != nil {
			log.Printf("Warning: failed to remove temp directory %s: %v", tempDir, fileError)
		}
		return nil, nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	injectiveApp := simapp.Setup(false, simtestutil.AppOptionsMap{
		flags.FlagHome: tempDir,
	})

	cleanup := func() {
		simapp.Cleanup(injectiveApp)
		if err := os.RemoveAll(tempDir); err != nil {
			log.Printf("Warning: failed to remove temp directory %s: %v", tempDir, err)
		}
	}

	return injectiveApp, cleanup, nil
}

// collectErrors retrieves all registered errors and groups them by codespace.
func collectErrors() (map[string][]ErrorInfo, error) {
	allErrors := sdkerrors.AllRegisteredErrors()
	if len(allErrors) == 0 {
		return nil, errors.New("no registered errors found")
	}

	codespaceGroups := make(map[string][]ErrorInfo)

	for _, err := range allErrors {
		codespace := err.Codespace()
		if codespace == "" {
			log.Printf("Warning: error with empty codespace: %v", err)
			continue
		}

		errorInfo := ErrorInfo{
			ModuleName:  codespace,
			ErrorCode:   err.ABCICode(),
			Description: err.Error(),
		}

		codespaceGroups[codespace] = append(codespaceGroups[codespace], errorInfo)
	}

	return codespaceGroups, nil
}

// sortErrorsByCode sorts errors within each codespace by error code.
func sortErrorsByCode(codespaceGroups map[string][]ErrorInfo) {
	for _, errorInfos := range codespaceGroups {
		sort.Slice(errorInfos, func(i, j int) bool {
			return errorInfos[i].ErrorCode < errorInfos[j].ErrorCode
		})
	}
}

// generateJSON creates JSON bytes for the given error infos without HTML escaping.
func generateJSON(errorInfos []ErrorInfo) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", jsonIndent)

	if err := encoder.Encode(errorInfos); err != nil {
		return nil, fmt.Errorf("failed to encode JSON: %w", err)
	}

	return buf.Bytes(), nil
}

// writeJSONFiles generates JSON files for each codespace in the specified destination folder.
func writeJSONFiles(codespaceGroups map[string][]ErrorInfo, destFolder string) error {
	if err := os.MkdirAll(destFolder, defaultDirMode); err != nil {
		return fmt.Errorf("failed to create destination folder %s: %w", destFolder, err)
	}

	var successCount int
	var generationErrors []error

	for codespace, errorInfos := range codespaceGroups {
		filename := fmt.Sprintf(filenameFmt, codespace)
		filePath := filepath.Join(destFolder, filename)

		jsonData, err := generateJSON(errorInfos)
		if err != nil {
			generationErrors = append(generationErrors, fmt.Errorf("failed to generate JSON for codespace %s: %w", codespace, err))
			continue
		}

		if err := os.WriteFile(filePath, jsonData, defaultFileMode); err != nil {
			generationErrors = append(generationErrors, fmt.Errorf("failed to write file %s: %w", filePath, err))
			continue
		}

		log.Printf("Generated %s with %d errors\n", filePath, len(errorInfos))
		successCount++
	}

	// Report any errors that occurred
	for _, err := range generationErrors {
		log.Printf("Error: %v", err)
	}

	if successCount == 0 {
		return errors.New("failed to generate any JSON files")
	}

	log.Printf("Successfully generated JSON files for %d codespaces in %s\n", successCount, destFolder)
	return nil
}

// run executes the main logic of the error documentation generator.
func run() error {
	config := parseFlags()

	_, cleanup, err := setupApp()
	if err != nil {
		return fmt.Errorf("failed to setup app: %w", err)
	}
	defer cleanup()

	codespaceGroups, err := collectErrors()
	if err != nil {
		return fmt.Errorf("failed to collect errors: %w", err)
	}

	sortErrorsByCode(codespaceGroups)

	if err := writeJSONFiles(codespaceGroups, config.DestFolder); err != nil {
		return fmt.Errorf("failed to write JSON files: %w", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
