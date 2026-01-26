package network

import (
	"fmt"
	"io"
	"strings"
)

/***** CONSTANT ********************************/

const (
	ONE_BYTE     = 1
	ONE_KILOBYTE = 1024 * ONE_BYTE
	ONE_MEGABYTE = 1024 * ONE_KILOBYTE
	ONE_GIGABYTE = 1024 * ONE_MEGABYTE
	ONE_TERABYTE = 1024 * ONE_GIGABYTE
)

/***** FUNCTION ********************************/

// Display the szie of a file in human-friendly form, such as "xx B", "xx KB", ..., "xx TB".
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

/***** CONSTANT ********************************/

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
	HTTPUserAgent               = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.0.3 Safari/605.1.15"
)

/***** STRUCT **********************************/

type TaskError interface {
	Error() string
	IsTemporary() bool
	IsEOF() bool
}

/***********************************************/

type taskError struct {
	err  error
	flag bool
}

/***** FUNCTION ********************************/

func NewTaskError(err error, flag bool) TaskError {
	return taskError{err: err, flag: flag}
}

/***********************************************/

func (e taskError) Error() string {
	return e.err.Error()
}

/***********************************************/

func (e taskError) IsTemporary() bool {
	return e.flag
}

/***********************************************/

func (e taskError) IsEOF() bool {
	return e.err == io.EOF
}

/***** STRUCT **********************************/

type NetworkInfo struct {
	Url      string `json:"url"`
	UserName string `json:"username"`
	Password string `json:"password"`
}

/***** FUNCTION ********************************/

func (s *NetworkInfo) IsHttp() bool {
	return strings.Contains(s.Url, "http://")
}

/***********************************************/

func (s *NetworkInfo) IsHttps() bool {
	return strings.Contains(s.Url, "https://") &&
		(!strings.Contains(s.Url, "https://cddis"))
}

/***********************************************/

func (s *NetworkInfo) IsHttpsCddis() bool {
	return strings.Contains(s.Url, "https://cddis")
}

/***********************************************/

func (s *NetworkInfo) IsFtp() bool {
	return strings.Contains(s.Url, "ftp://")
}

/***********************************************/

func (s *NetworkInfo) IsFtps() bool {
	return strings.Contains(s.Url, "ftps://")
}

/***** STRUCT **********************************/

type NetworkTask struct {
	Source   NetworkInfo // URL, username and password
	Path     string      // path of the file to be saved
	Size     int64       // size of downloaded part
	Continue bool        // whether to resume getting a partially-downloaded file or not
}

/***********************************************/
