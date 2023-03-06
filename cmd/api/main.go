package main

import "github.com/zan8in/pyxis/pkg/pyxis"

func main() {
	scanner, err := pyxis.NewScanner(&pyxis.Options{
		HostsFile: "./target.txt",
	})
	if err != nil {
		panic(err)
	}
	scanner.Run()
}
