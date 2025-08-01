## Pyxis

<p align="center">
    <img width="120" src="image/pyxis.png"/>
<p>

<p align="center">
    <a href="https://github.com/zan8in/pyxis/releases"><img src="https://img.shields.io/github/release/zan8in/pyxis"></a>
    <a href="https://github.com/zan8in/pyxis"><img src="https://img.shields.io/badge/language-go-blue"></a>
    <a href="https://github.com/zan8in/pyxis/blob/main/LICENSE"><img src="https://img.shields.io/github/license/zan8in/pyxis"></a>
</p>

Pyxis 是一个强大的 Web 指纹识别工具，能够自动识别 HTTP 和 HTTPS 请求，获取响应头、状态码、响应大小、响应时间，并进行指纹识别（favicon hash、服务、CMS、框架等）。

## ✨ 功能特性

* [x] **自动协议识别** - 自动识别 HTTP/HTTPS 协议
* [x] **响应信息获取** - 获取标题/状态码/响应大小/响应时间
* [x] **Favicon Hash** - 计算网站 favicon 的 hash 值
* [x] **指纹识别** - 支持 10000+ 指纹库识别
* [x] **CDN 检测** - 检测目标是否使用 CDN 服务
* [x] **多种输出格式** - 支持 TXT、CSV、JSON 格式输出
* [x] **代理支持** - 支持 HTTP/SOCKS5 代理
* [x] **并发扫描** - 支持高并发扫描提升效率
* [x] **灵活输入** - 支持单个目标、多个目标、文件输入

## 📦 安装

### 从源码安装

```bash
go install github.com/zan8in/pyxis/cmd/pyxis@latest
```

### 从 Release 下载

前往 [Releases](https://github.com/zan8in/pyxis/releases) 页面下载对应平台的二进制文件。

### 从源码编译

```bash
git clone https://github.com/zan8in/pyxis.git
cd pyxis
go build -o pyxis cmd/pyxis/main.go
```

## 🚀 使用方法

### 基本用法

**单个目标扫描**
```bash
pyxis -t example.com
```

**多个目标扫描（逗号分隔）**
```bash
pyxis -t example.com,scanme.nmap.org
pyxis -t 192.168.88.168:8080,192.168.66.200
```

**从文件读取目标列表**
```bash
pyxis -T url_list.txt
```

### 输出选项

**输出到文件（支持多种格式）**
```bash
pyxis -T url_list.txt -o result.csv   # CSV 格式
pyxis -T url_list.txt -o result.json  # JSON 格式
pyxis -T url_list.txt -o result.txt   # TXT 格式
```

### CDN 检测

**仅进行 CDN 检测**
```bash
pyxis -t example.com -cdn
pyxis -T url_list.txt -cdn -o cdn_results.txt
```

### 代理设置

**HTTP 代理**
```bash
pyxis -t example.com -proxy http://127.0.0.1:1082
```

**SOCKS5 代理**
```bash
pyxis -t example.com -proxy socks5://127.0.0.1:1081
```

### 高级选项

**设置超时时间**
```bash
pyxis -t example.com -timeout 30
```

**设置重试次数**
```bash
pyxis -t example.com -retries 3
```

**设置并发速率**
```bash
pyxis -t example.com -rate 100
```

**静默模式（仅显示结果）**
```bash
pyxis -t example.com -silent
```

## 📋 参数说明

### 输入选项
| 参数 | 简写 | 描述 | 示例 |
|------|------|------|------|
| `-target` | `-t` | 要扫描的目标主机（逗号分隔） | `-t example.com,google.com` |
| `-target-file` | `-T` | 包含目标列表的文件 | `-T targets.txt` |

### 输出选项
| 参数 | 简写 | 描述 | 示例 |
|------|------|------|------|
| `-output` | `-o` | 输出文件路径（支持 txt/csv/json） | `-o results.json` |
| `-silent` | | 静默模式，仅显示结果 | `-silent` |

### 优化选项
| 参数 | 默认值 | 描述 | 示例 |
|------|--------|------|------|
| `-retries` | 1 | 重试次数 | `-retries 3` |
| `-timeout` | 10 | 超时时间（秒） | `-timeout 30` |
| `-cdn` | false | 仅进行 CDN 检测 | `-cdn` |
| `-rate` | 150 | 每秒发送的数据包数量 | `-rate 100` |

### 代理选项
| 参数 | 描述 | 示例 |
|------|------|------|
| `-proxy` | HTTP/SOCKS5 代理设置 | `-proxy socks5://127.0.0.1:1080` |

## 📄 输出格式

### TXT 格式
```
example.com 93.184.216.34 false http://example.com google.com 142.250.191.14 true https://google.com
```


### CSV 格式
```csv
Host,IP,CDN,FullUrl,Title,StatusCode,Fingerprint
example.com,93.184.216.34,false,http://example.com,Example Domain,200,nginx
```

### JSON 格式
```json
[
  {
    "host": "example.com",
    "ip": "93.184.216.34",
    "cdn": false,
    "full_url": "http://example.com",
    "title": "Example Domain",
    "status_code": 200,
    "fingerprint": "nginx"
  }
]
```

## 📝 使用示例

### 示例 1: 基本扫描
```bash
# 扫描单个网站
pyxis -t https://example.com

# 扫描多个网站并保存结果
pyxis -t example.com,google.com,github.com -o scan_results.json
```

### 示例 2: 批量扫描
```bash
# 批量扫描并输出到 CSV
pyxis -T targets.txt -o results.csv
```

### 示例 3: CDN 检测
```bash
# 仅检测 CDN
pyxis -T targets.txt -cdn -o cdn_check.txt

# 通过代理进行 CDN 检测
pyxis -t cloudflare.com -cdn -proxy socks5://127.0.0.1:1080
```

### 示例 4: 高级配置
```bash
# 高并发扫描，设置超时和重试
pyxis -T large_targets.txt -rate 200 -timeout 15 -retries 2 -o results.json
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request 来帮助改进 Pyxis！

## 📄 许可证

本项目采用 [MIT License](LICENSE) 许可证。

## ⭐ Star History

如果这个项目对你有帮助，请给它一个 Star！

---

**注意**: 请确保在使用本工具时遵守相关法律法规，仅对授权的目标进行扫描。