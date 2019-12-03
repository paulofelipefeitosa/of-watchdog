package executor

import (
	"bytes"
	"io"
	"log"
	"math"
	"os"
)

func tail(filepath string) []byte {
	f, fLen, err := openFile(filepath)
	if err != nil {
		return []byte{}
	}
	bufLen := int64(512)
	buf := make([]byte, bufLen)
	var line bytes.Buffer
	for fPointer := fLen; fPointer >= 0; fPointer -= bufLen {
		offset := fPointer - bufLen
		realOffset := int64(math.Max(0, float64(offset)))
		readBytes, err := f.ReadAt(buf, realOffset)
		if offset < 0 { // Do not read the same content twice
			readBytes = int(fLen % bufLen)
		}
		if err != nil && err != io.EOF {
			log.Printf("Cannot read %s file seeked at %d, operation received %v", filepath, realOffset, err.Error())
			break
		}
		last, end := getLastLineInitPos(readBytes, buf)
		reverse(buf, last, readBytes-1)
		line.Write(buf[last:readBytes])
		if end {
			break
		}
	}
	s := line.Bytes()
	reverse(s, 0, len(s) - 1)
	return s
}

func reverse(arr []byte, init int, end int) {
	for ; init < end; init, end = init+1, end-1 {
		arr[init], arr[end] = arr[end], arr[init]
	}
}

func getLastLineInitPos(readBytes int, buf []byte) (int, bool) {
	if readBytes == 0 {
		return 0, true
	}
	i := readBytes - 1
	if buf[i] == byte(10) || buf[i] == byte(13) {
		i--
	}
	for ; i >= 0; i-- {
		if buf[i] == byte(10) || buf[i] == byte(13) { // if is a new line
			return i + 1, true
		}
	}
	return 0, false
}

func openFile(filepath string) (*os.File, int64, error) {
	f, err := os.Open(filepath)
	if err != nil {
		log.Printf("Cannot open %s file, operation received %v", filepath, err.Error())
		return nil, 0, err
	}
	fStat, err := os.Stat(filepath)
	if err != nil {
		log.Printf("Cannot get %s file statistics, operation received %v", filepath, err.Error())
		return nil, 0, err
	}
	fLen := fStat.Size()
	return f, fLen, nil
}
