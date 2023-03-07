package main

import (
	"fmt"
	"net"

	"github.com/zan8in/pyxis/pkg/pyxis"
)

func main() {
	// arr, err := iputil.ToFQDN("lankegp.com")
	addr, err := net.ResolveIPAddr("ip", "http://lankegp.com")
	fmt.Println(addr, err)

}

func main2() {
	scanner, err := pyxis.NewScanner(&pyxis.Options{
		HostsFile: "./target.txt",
	})
	if err != nil {
		panic(err)
	}
	scanner.Run()

	if scanner.Result.HasHostResult() {
		for hostResult := range scanner.Result.GetHostResult() {
			fmt.Println(hostResult.FullUrl, hostResult.Title, hostResult.FaviconHash)
		}
	}
}
