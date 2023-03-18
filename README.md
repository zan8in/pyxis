## Pyxis

<p align="center">
    <img width="120" src="image/pyxis.png"/>
<p>

pyxis can automatically identify http and https requests, and get response headers, status codes, response size, response time, tools for fingerprinting (favicon has, service, CMS, framework, etc.)

## Features

* [x] Automatically identify http/https<br/>
* [x] Response title/status code/response size/response time<br/>
* [x] Favicon Hash<br/>
* [x] Fingerprinting (2000+)<br/>

## Example

URL Input
```
pyxis -t example.com
```

Multiple URLs Input (comma-separ)
```
pyxis -t example.com,scanme.nmap.org
pyxis -t 192.168.88.168:8080,192.168.66.200
```

List of URLs Input
```
$ cat url_list.txt

http://example.com
scanme.nmap.org
..
```

Output files (csv/json/txt)
```
pyxis -T url_list.txt -o result.csv
pyxis -T url_list.txt -o result.json
pyxis -T url_list.txt -o result.txt
```

## Pyxis as a library
```
package main

import (
	"fmt"

	"github.com/zan8in/pyxis/pkg/pyxis"
)

func main() {
	scanner, err := pyxis.NewScanner(&pyxis.Options{
		HostsFile: "./target.txt",
	})
	if err != nil {
		panic(err)
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
```