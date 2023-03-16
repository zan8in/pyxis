## Pyxis

<p align="center">
    <img width="120" src="image/pyxis.png"/>
<p>

pyxis can automatically identify http and https requests, and get response headers, status codes, response size, response time, tools for fingerprinting (favicon has, service, CMS, framework, etc.)

## Features

☐ Automatically identify http/https<br/>
☐ Response title/status code/response size/response time<br/>
☐ Favicon Hash<br/>
☐ Fingerprinting (2000+)<br/>

## Example

```
pyxis -t example.com
pyxis -t example.com,scanme.nmap.org
pyxis -t 192.168.88.168:8080,192.168.66.200

pyxis -T urls.txt
cat ./urls.txt
example.com
scanme.nmap.org
...

pyxis -T urls.txt -o result.csv
pyxis -T urls.txt -o result.json
pyxis -T urls.txt -o result.txt
```

