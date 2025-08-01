package pyxis

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/zan8in/gologger"
	fileutil2 "github.com/zan8in/pins/file"
	"github.com/zan8in/pyxis/pkg/logcolor"
	"github.com/zan8in/pyxis/pkg/result"
	"github.com/zan8in/pyxis/pkg/util/fileutil"
)

type OutputResult struct {
	Flag          int    `json:"flag" csv:"flag"`
	FullUrl       string `json:"fullurl,omitempty" csv:"fullurl"`
	Host          string `json:"host,omitempty" csv:"host"`
	IP            string `json:"ip,omitempty" csv:"ip"`
	Port          int    `json:"port" csv:"port"`
	TLS           bool   `json:"tls" csv:"tls"`
	Title         string `json:"title,omitempty" csv:"title"`
	StatusCode    int    `json:"statuscode,omitempty" csv:"statuscode"`
	ContentLength int64  `json:"contentlength,omitempty" csv:"contentlength"`
	ResponseTime  int64  `json:"responsetime,omitempty" csv:"responsetime"`
	FaviconHash   string `json:"faviconhash,omitempty" csv:"faviconhash"`
	Fingerprint   string `json:"fingerprint,omitempty" csv:"fingerprint"`
	Cdn           string `json:"cdn,omitempty" csv:"cdn"` // 新增CDN字段
}

func (r *Runner) print(result *result.HostResult) {
	// 如果启用了CDN选项，只显示CDN检测结果
	if r.Options.Cdn {
		if result.Flag == 0 {
			if result.Cdn != "" {
				fmt.Printf("%s [%s][%s]\n",
					result.Host,
					logcolor.LogColor.IP(result.IP),
					logcolor.LogColor.Cdn(result.Cdn),
				)
			} else {
				fmt.Printf("%s [%s][%s]\n",
					result.Host,
					logcolor.LogColor.IP(result.IP),
					logcolor.LogColor.Failed("Not CDN"),
				)
			}
		} else {
			fmt.Printf("%s [%s]\n",
				result.Host,
				logcolor.LogColor.Failed("Failed to detect"),
			)
		}
		return
	}

	if result.Flag == 0 {
		fmt.Printf("%s [%s][%s][%s][%s][%s][%s][%s]\n",
			result.FullUrl,
			logcolor.LogColor.Status(result.StatusCode),
			logcolor.LogColor.ContentLength(FormatFileSize(result.ContentLength)),
			logcolor.LogColor.Title(result.Title),
			logcolor.LogColor.Fingerprint(result.FingerPrint),
			logcolor.LogColor.Faviconhash(result.FaviconHash),
			logcolor.LogColor.IP(result.IP),
			logcolor.LogColor.Cdn(result.Cdn),

			// result.TLS,
			// result.Host,
			// result.Port,
			// result.IP,
			// result.StatusCode,
			// result.ResponseTime,
			// result.ContentLength,
			// result.FaviconHash,
			// result.FingerPrint,
		)
	} else {
		fmt.Printf("%s [%s%s]\n",
			result.Host,
			logcolor.LogColor.Failed("Failed to access "),
			logcolor.LogColor.Failed(result.Host),
		)
	}

}

func (r *Runner) WriteOutput() {
	if !r.Result.HasHostResult() || len(r.Options.Output) == 0 {
		return
	}

	var (
		file     *os.File
		output   string
		err      error
		fileType uint8
		csvutil  *csv.Writer
	)

	output = r.Options.Output

	fileType = fileutil.FileExt(output)

	outputFolder := filepath.Dir(output)
	if fileutil.FolderExists(outputFolder) {
		mkdirErr := os.MkdirAll(outputFolder, 0700)
		if mkdirErr != nil {
			gologger.Error().Msgf("Could not create output folder %s: %s\n", outputFolder, mkdirErr)
			return
		}
	}

	file, err = os.Create(output)
	if err != nil {
		gologger.Error().Msgf("Could not create file %s: %s\n", output, err)
		return
	}
	defer file.Close()

	switch fileType {
	case fileutil.FILE_CSV:
		csvutil = csv.NewWriter(file)
		file.WriteString("\xEF\xBB\xBF")
	// csvutil.Write([]string{"FullURL", "Title", "StatusCode", "Faviconhash", "Fingerprint", "ContentLength", "ResponseTime", "Host", "IP", "Port", "TLS"})
	case fileutil.FILE_JSON:
		fileutil.BufferWriteAppend(file, "[")
	}

	for result := range r.Result.GetHostResult() {
		or := &OutputResult{
			Flag:          result.Flag,
			FullUrl:       result.FullUrl,
			Host:          result.Host,
			IP:            result.IP,
			Port:          result.Port,
			TLS:           result.TLS,
			Title:         result.Title,
			StatusCode:    result.StatusCode,
			FaviconHash:   result.FaviconHash,
			ContentLength: result.ContentLength,
			ResponseTime:  result.ResponseTime,
			Fingerprint:   result.FingerPrint,
			Cdn:           result.Cdn, // 添加CDN字段
		}

		if or.Flag == 1 {
			continue
		}

		switch fileType {
		case fileutil.FILE_TXT:
			fileutil.BufferWriteAppend(file, or.TXT())
		case fileutil.FILE_JSON:
			b, marshallErr := or.JSON()
			if marshallErr != nil {
				continue
			}
			fileutil.BufferWriteAppend(file, string(b)+",")
		case fileutil.FILE_CSV:
			csvutil.Write(or.CSV())
		}

		switch fileType {
		case fileutil.FILE_CSV:
			csvutil.Flush()
		}
	}

	switch fileType {
	case fileutil.FILE_JSON:
		fileutil.BufferWriteAppend(file, "]")
		fileutil2.CoverFile(output, ",]", "]")
	}

}

func (or *OutputResult) JSON() ([]byte, error) {
	return json.Marshal(or)
}

func (or *OutputResult) TXT() string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\n", or.Host, or.IP, or.Cdn, or.FullUrl)
}

func (or *OutputResult) CSV() []string {
	return []string{
		or.Host,
		or.IP,
		or.Cdn,
		or.FullUrl,
		or.Title,
		strconv.Itoa(or.StatusCode),
		or.FaviconHash,
		or.Fingerprint,
		FormatFileSize(or.ContentLength),
		fmt.Sprintf("%d", or.ResponseTime),
		strconv.Itoa(or.Port),
		fmt.Sprintf("%t", or.TLS),
	}
}

func FormatFileSize(fileSize int64) (size string) {
	if fileSize < 1024 {
		//return strconv.FormatInt(fileSize, 10) + "B"
		return fmt.Sprintf("%.2fB", float64(fileSize)/float64(1))
	} else if fileSize < (1024 * 1024) {
		return fmt.Sprintf("%.2fKB", float64(fileSize)/float64(1024))
	} else if fileSize < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fMB", float64(fileSize)/float64(1024*1024))
	} else if fileSize < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fGB", float64(fileSize)/float64(1024*1024*1024))
	} else if fileSize < (1024 * 1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fTB", float64(fileSize)/float64(1024*1024*1024*1024))
	} else { //if fileSize < (1024 * 1024 * 1024 * 1024 * 1024 * 1024)
		return fmt.Sprintf("%.2fPB", float64(fileSize)/float64(1024*1024*1024*1024*1024))
	}
}
