package main

import (
	"fmt"
	"log"

	"github.com/zan8in/pyxis/pkg/pyxis"
)

func main() {
	scanner, err := pyxis.NewScanner(&pyxis.Options{
		HostsFile: "./target.txt",
	})
	if err != nil {
		log.Fatalf("Failed to create scanner: %v", err)
	}
	scanner.Run()

	if scanner.Result.HasHostResult() {
		for hostResult := range scanner.Result.GetHostResult() {
			fmt.Println(
				hostResult.FullUrl,
				hostResult.Title,
				hostResult.FaviconHash,
				hostResult.FingerPrint,
			)
		}
	}
}
