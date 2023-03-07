package pyxis

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/zan8in/gologger"
	"github.com/zan8in/pyxis/pkg/util/fileutil"
)

type OutputResult struct {
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

	if fileType == fileutil.FILE_CSV {
		csvutil = csv.NewWriter(file)
		file.WriteString("\xEF\xBB\xBF")
		csvutil.Write([]string{"FullURL", "Title", "StatusCode", "Faviconhash", "ContentLength", "ResponseTime", "Host", "IP", "Port", "TLS"})
	}

	for result := range r.Result.GetHostResult() {
		or := &OutputResult{
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

}

func (or *OutputResult) JSON() ([]byte, error) {
	return json.Marshal(or)
}

func (or *OutputResult) TXT() string {
	return fmt.Sprintf("%s\t%s\t%s\n", or.FullUrl, or.Title, or.FaviconHash)
}

func (or *OutputResult) CSV() []string {
	// {"FullURL", "Title", "StatusCode", "Faviconhash", "ContentLength", "ResponseTime", "Host", "IP", "Port", "TLS"})
	return []string{
		or.FullUrl,
		or.Title,
		strconv.Itoa(or.StatusCode),
		or.FaviconHash,
		fmt.Sprintf("%d", or.ContentLength),
		fmt.Sprintf("%d", or.ResponseTime),
		or.Host,
		or.IP,
		strconv.Itoa(or.Port),
		fmt.Sprintf("%t", or.TLS),
	}
}
