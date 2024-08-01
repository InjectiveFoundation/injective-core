package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type DepsMap struct {
	Repo string
	Deps map[string]string
}

func main() {
	chainDeps := extractDepsMap("CHAIN", "./go.mod")
	sdkGoDeps := extractDepsMap("SDK", "../sdk-go/go.mod")
	indexerDeps := extractDepsMap("INDEXER", "../injective-indexer/go.mod")

	compareDeps(chainDeps, sdkGoDeps)
	compareDeps(chainDeps, indexerDeps)
}

func extractDepsMap(name, path string) *DepsMap {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	depMaps := map[string]string{}

	pattern := `^\t([A-Za-z0-9-._/=> ]*)`
	r := regexp.MustCompile(pattern)
	for scanner.Scan() {
		line := scanner.Text()
		match := r.FindStringSubmatch(line)
		if len(match) > 0 {
			parts := strings.Split(match[1], " ")
			switch len(parts) {
			case 2:
				depMaps[parts[0]] = parts[1]
			case 4:
				depMaps[parts[0]] = parts[1] + " " + parts[2] + " " + parts[3]
			default:
				panic("invalid dep parts")
			}
		}
	}

	return &DepsMap{
		Repo: name,
		Deps: depMaps,
	}
}

func compareDeps(parent, child *DepsMap) {
	fmt.Println("====================DEPS COMPARISON====================")
	for n, v := range parent.Deps {
		if child.Deps[n] != "" {
			if !strings.HasPrefix(v, child.Deps[n]) && !strings.HasPrefix(child.Deps[n], v) {
				fmt.Printf("PACKAGE: %s --- %s %s >< %s %s\n", n, parent.Repo, v, child.Repo, child.Deps[n])
			}
		}
	}
}
