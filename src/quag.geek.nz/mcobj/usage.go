package main

import "runtime"

var (
	exampleWorldMap = map[string]string{
		"darwin":  "~/Library/Application\\ Support/minecraft/saves/World1",
		"linux":   "~/.minecraft/saves/World1",
		"windows": "%AppData%\\.minecraft\\saves\\World1",
	}
)

func ExampleWorldPath() string {
	return exampleWorldMap[runtime.GOOS]
}
