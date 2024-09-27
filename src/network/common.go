package network

import (
	"fmt"
	"strings"
)

const (
	OneByte     = 1
	OneKiloByte = 1024 * OneByte
	OneMegaByte = 1024 * OneKiloByte
	OneGigaByte = 1024 * OneMegaByte
	OneTeraByte = 1024 * OneGigaByte
)

const (
	FTPCodePositive             = 2

	FTPCodeFileStatusOk         = 150 // about to open data connection
	FTPCodeCommandOk            = 200
	FTPCodeSystemStatus         = 211
	FTPCodeFileStatus           = 213
	FTPCodeServiceReady         = 220
	FTPCodePassiveMode          = 227
	FTPCodeLoggedIn             = 230
	FTPCodeAuthOk               = 234
	FTPCodeFileActionOk         = 250
	FTPCodeNeedPassword         = 331
	FTPCodeFileActionPending    = 350 // pending further information
	FTPCodeDataConnectionFailed = 425
	FTPCodeConnectionClosed     = 426

	HTTPUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.0.3 Safari/605.1.15"
)

type TaskError interface {
	Error() string
	Temporary() bool
}

type taskError struct {
	err  error
	flag bool
}

func NewTaskError(err error, flag bool) taskError {
	return taskError{err: err, flag: flag}
}

func (e taskError) Error() string {
	return e.err.Error()
}

func (e taskError) Temporary() bool {
	return e.flag
}

type NetInfo struct {
	URL      string `json:"url"`
	UserName string `json:"username"`
	Password string `json:"password"`
}

func (s *NetInfo) IsHttp() bool {
		return strings.Contains(s.URL, "http://")
}

func (s *NetInfo) IsHttpsCddis() bool {
	return strings.Contains(s.URL, "https://cddis")
}

func (s *NetInfo) IsFtp() bool {
	return strings.Contains(s.URL, "ftp://")
}

func (s *NetInfo) IsFtps() bool {
	return strings.Contains(s.URL, "ftps://")
}

type NetTask struct {
	Source   NetInfo // URL, username and password
	Path     string  // path of the file to be saved
	Size     int64   // size of downloaded part
	Continue bool    // whether to resume getting a partially-downloaded file or not
}

/*
Display the szie of a file in human-friendly form, such as "xx B", "xx KB", ..., "xx TB".
*/
func SizeRepr(size int64) string {
	var sizeF float64 = float64(size)
	var idx int
	var unit string

	for idx = 1; idx < 6; idx++ {
		if sizeF < 1024 {
			break
		}

		sizeF /= 1024
	}

	switch idx {
	case 1:
		unit = "B"
	case 2:
		unit = "KB"
	case 3:
		unit = "MB"
	case 4:
		unit = "GB"
	default:
		unit = "TB"
	}

	if idx == 1 {
		return fmt.Sprintf("%d %s", size, unit)
	} else {
		return fmt.Sprintf("%.2f %s", sizeF, unit)
	}
}
