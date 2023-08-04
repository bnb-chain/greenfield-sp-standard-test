package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"strings"
)

type File struct {
	Reader    *bytes.Reader
	Path      string
	Size      int64
	Name      string
	Extension string
}

func GetFile(path string) (*File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	split := strings.Split(file.Name(), ".")
	readAll, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(readAll)
	return &File{reader, path, stat.Size(), stat.Name(), split[1]}, nil
}

func GetFileBufferBySize(size uint64) bytes.Buffer {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	var buffer bytes.Buffer
	buffer.Write(b)
	return buffer
}

func AppendFile(buf bytes.Buffer, pre string) *bytes.Buffer {
	if buf.Len() < len(pre) {
		return bytes.NewBuffer([]byte(pre[0:buf.Len()]))
	} else {
		return bytes.NewBuffer(append(buf.Bytes()[0:buf.Len()-len(pre)], []byte(pre)...))
	}
}
func GetSizeByString(size string) int64 {
	len := len(size)
	mul := int64(1)
	switch size[len-1] {
	case 'g':
		{
			mul = 1024 * 1024 * 1024
		}
	case 'm':
		{
			mul = 1024 * 1024
		}
	case 'k':
		{
			mul = 1024
		}
	case 'b':
		{
			mul = 1
		}
	}
	i, _ := strconv.Atoi(size[0 : len-1])
	return int64(i) * mul
}
func GetParamFromCSV(path string, skipFirstRow bool) [][]string {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	record, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}
	if skipFirstRow {
		record = record[1:]
	}
	return record
}
func WriteToCSV(data [][]string, path string) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	for _, d := range data {
		err := writer.Write(d)
		if err != nil {
			return err
		}
	}
	writer.Flush()
	return err
}

func IsContain(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}
