package network

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func HTTPDownload(f *NetworkTask) TaskError {
	var idx int64
	var flag int
	var err error

	if f.Continue {
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
		info, err := os.Stat(f.Path)

		if err != nil {
			idx = 0
		} else {
			if info.Size() != f.Size { // some error occurs, redownload
				idx = 0
				f.Size = 0
				flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
			} else {
				idx = f.Size
			}
		}
	} else {
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		idx = 0
	}

	client := http.Client{}
	ctx, cancel := context.WithCancel(context.TODO())
	timer := time.AfterFunc(time.Minute, func() { cancel() })

	request, _ := http.NewRequest(http.MethodGet, f.Source.Url, nil)
	request.Header.Add("User-Agent", HTTPUserAgent)
	request.Header.Add("Range", fmt.Sprintf("bytes=%d-", idx))
	request = request.WithContext(ctx)

	response, err := client.Do(request)

	if err != nil {
		return taskError{err: err, flag: true}
	} else if response.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		request.Body.Close()
		return nil
	} else if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent {
		response.Body.Close()
		err = fmt.Errorf("invalid response status code %d", response.StatusCode)
		return taskError{err: err, flag: true}
	}

	defer response.Body.Close()

	fp, err := os.OpenFile(f.Path, flag, 0664)

	if err != nil {
		return taskError{err: err, flag: false}
	}

	defer fp.Close()

	var n int64

	for {
		timer.Reset(30 * time.Second)
		n, err = io.CopyN(fp, response.Body, 1024)
		f.Size += n

		if err == io.EOF {
			return nil
		} else if err != nil {
			return taskError{err: err, flag: true}
		}
	}
}
