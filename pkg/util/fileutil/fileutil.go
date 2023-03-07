package fileutil

import (
	"bufio"
	"os"
	"path"
)

// FileExists checks if the file exists in the provided path
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// FolderExists checks if the folder exists
func FolderExists(foldername string) bool {
	info, err := os.Stat(foldername)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false
	}
	return info.IsDir()
}

// FileOrFolderExists checks if the file/folder exists
func FileOrFolderExists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

type FileType = uint8

const (
	FILE_TXT = iota
	FILE_JSON
	FILE_CSV
	NOT_FOUND
)

func FileExt(filename string) FileType {
	ext := path.Ext(filename)
	switch ext {
	case ".txt":
		return FILE_TXT
	case ".json":
		return FILE_JSON
	case ".csv":
		return FILE_CSV
	default:
		return NOT_FOUND
	}
}

func BufferWriteAppend(file *os.File, content string) error {
	buf := bufio.NewWriter(file)
	buf.WriteString(content)
	return buf.Flush()
}
