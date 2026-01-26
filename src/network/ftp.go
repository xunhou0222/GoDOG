package network

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type ftpConn struct {
	conn     net.Conn
	timeout  time.Duration
	reader   *textproto.Reader
	writer   *textproto.Writer
	features map[string]string
}

func NewFTPConn(addr string, timeout time.Duration) (*ftpConn, error) {
	c := new(ftpConn)
	var err error

	c.conn, err = net.DialTimeout("tcp", addr, timeout)

	if err != nil {
		return nil, err
	}

	c.timeout = timeout
	c.reader = textproto.NewReader(bufio.NewReader(c.conn))
	c.writer = textproto.NewWriter(bufio.NewWriter(c.conn))
	c.features = make(map[string]string)

	c.conn.SetReadDeadline(time.Now().Add(c.timeout))
	_, _, err = c.reader.ReadResponse(FTPCodeServiceReady)

	if err != nil {
		c.Close()
		err = fmt.Errorf("failed to get welcome message, %s", err)
		return nil, err
	}

	err = c.fetchFeatures()

	if err != nil {
		c.Close()
		err = fmt.Errorf("failed to fetch features, %s", err)
		return nil, err
	}

	return c, nil
}

// close the connection.
func (c *ftpConn) Close() error {
	return c.conn.Close()
}

func (c *ftpConn) SendCommand(expectCode int, format string, args ...interface{}) (int, string, error) {
	c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	err := c.writer.PrintfLine(format, args...)

	if err != nil {
		return 0, "", err
	}

	c.conn.SetReadDeadline(time.Now().Add(c.timeout))
	code, msg, err := c.reader.ReadResponse(expectCode)

	if err != nil {
		return 0, "", err
	}

	return code, msg, err
}

func (c *ftpConn) fetchFeatures() error {
	code, msg, err := c.SendCommand(FTPCodePositive, "FEAT")

	if err != nil {
		return err
	}

	if code != FTPCodeSystemStatus {
		return nil
	}

	for _, line := range strings.Split(msg, "\n") {
		if len(line) > 0 && line[0] == ' ' {
			parts := strings.SplitN(strings.Trim(line, " \r\n"), " ", 2)

			if len(parts) == 1 {
				c.features[strings.ToUpper(parts[0])] = ""
			} else if len(parts) == 2 {
				c.features[strings.ToUpper(parts[0])] = parts[1]
			}
		}
	}

	return nil
}

func (c *ftpConn) IsResumable() bool {
	val, ok := c.features["REST"]

	return ok && val == "STREAM"
}

func FTPDownload(f *NetworkTask) TaskError {
	var offset int64
	var flag int
	var err error

	if f.Continue {
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
		info, err := os.Stat(f.Path)

		if err != nil {
			offset = 0
		} else {
			if info.Size() != f.Size { // some error occurs, redownload
				offset = 0
				f.Size = 0
				flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
			} else {
				offset = f.Size
			}
		}
	} else {
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		offset = 0
	}

	username := f.Source.UserName
	password := f.Source.Password

	rawURL := f.Source.Url
	pURL, err := url.Parse(rawURL)

	if err != nil {
		err = fmt.Errorf("falied to parse URL, %s", err)
		return taskError{err: err, flag: false}
	}

	addr := pURL.Host
	path := pURL.Path

	if pURL.Port() == "" {
		addr += ":21"
	}

	conn, err := NewFTPConn(addr, time.Minute)

	if err != nil {
		err = fmt.Errorf("failed to connect to the server, %s", err)
		return taskError{err: err, flag: true}
	}

	defer conn.Close()

	_, _, err = conn.SendCommand(FTPCodeNeedPassword, "USER %s", username)

	if err != nil {
		err = fmt.Errorf("failed to send USER command, %s", err)
		return taskError{err: err, flag: false}
	}

	_, _, err = conn.SendCommand(FTPCodeLoggedIn, "PASS %s", password)

	if err != nil {
		err = fmt.Errorf("failed to send PASS command, %s", err)
		return taskError{err: err, flag: false}
	}

	_, _, err = conn.SendCommand(FTPCodeCommandOk, "TYPE I")

	if err != nil {
		err = fmt.Errorf("failed to send TYPE command, %s", err)
		return taskError{err: err, flag: false}
	}

	_, msg, err := conn.SendCommand(FTPCodePassiveMode, "PASV")

	if err != nil {
		err = fmt.Errorf("failed to send PASV command, %s", err)
		return taskError{err: err, flag: false}
	}

	startIdx := strings.Index(msg, "(")
	endIdx := strings.LastIndex(msg, ")")

	if startIdx == -1 || endIdx == -1 || startIdx > endIdx {
		err = fmt.Errorf("failed to get the address of data connection")
		return taskError{err: err, flag: false}
	}

	addrParts := strings.Split(msg[startIdx+1:endIdx], ",")

	if len(addrParts) != 6 {
		err = fmt.Errorf("failed to get the address of data connection, invalid host")
		return taskError{err: err, flag: false}
	}

	host := strings.Join(addrParts[0:4], ".")

	port := 0

	for i, part := range addrParts[4:6] {
		iport, err := strconv.Atoi(part)

		if err != nil {
			err = fmt.Errorf("failed to get the address of data connection, invalid port")
			return taskError{err: err, flag: false}
		}

		port |= iport << (byte(1-i) * 8)
	}

	if conn.IsResumable() {
		conn.SendCommand(FTPCodeFileActionPending, "REST %d", offset)
	}

	DataAddr := fmt.Sprintf("[%s]:%d", host, port)
	dconn, err := net.DialTimeout("tcp", DataAddr, conn.timeout)

	if err != nil {
		err = fmt.Errorf("failed to active data connection")
		return taskError{err: err, flag: false}
	}

	defer dconn.Close()

	_, _, err = conn.SendCommand(FTPCodeFileStatusOk, "RETR %s", path)

	if err != nil {
		err = fmt.Errorf("failed to send RETR command, %s", err)
		return taskError{err: err, flag: false}
	}

	fp, err := os.OpenFile(f.Path, flag, 0664)

	if err != nil {
		return taskError{err: err, flag: false}
	}

	defer fp.Close()

	var n int64
	timer := time.AfterFunc(time.Minute, func() { dconn.Close() })

	for {
		timer.Reset(30 * time.Second)
		n, err = io.CopyN(fp, dconn, 1024)
		f.Size += n

		if err == io.EOF {
			return nil
		} else if err != nil {
			return taskError{err: err, flag: true}
		}
	}
}
