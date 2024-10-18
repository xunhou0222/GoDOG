package network

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/url"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	mutexTLS   sync.Mutex
	configTLS  tls.Config
	ifInitTLS  atomic.Bool
)

type ftpsConn struct {
	rawConn   net.Conn
	tlsConn   *tls.Conn
	timeout   time.Duration
	ctrlConn  net.Conn
	reader    *textproto.Reader
	writer    *textproto.Writer
	features  map[string]string
}

func getTLSConfig() (err error) {
	mutexTLS.Lock()
	defer mutexTLS.Unlock()

	if ok := ifInitTLS.Load(); ok {
		return nil
	}

	configTLS.InsecureSkipVerify = true
	configTLS.ClientAuth = tls.VerifyClientCertIfGiven
	configTLS.CipherSuites = []uint16{
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	}

	configTLS.RootCAs, err = x509.SystemCertPool()

	if err != nil {
		return fmt.Errorf( "failed to get root CAs, %s", err.Error() )
	}

	configTLS.ClientCAs = configTLS.RootCAs

	return nil
}

func NewFTPSConn(addr string, timeout time.Duration) (*ftpsConn, error) {
	c := new(ftpsConn)
	var err error

	c.rawConn, err = net.DialTimeout("tcp", addr, timeout)

	if err != nil {
		return nil, err
	}

	c.timeout = timeout
	c.setCtrlConn(c.rawConn)
	c.features = make(map[string]string)
	c.ctrlConn.SetReadDeadline(time.Now().Add(c.timeout))
	_, _, err = c.reader.ReadResponse(FTPCodeServiceReady)

	if err != nil {
		c.Close()
		err = fmt.Errorf("failed to get welcome message, %s", err)
		return nil, err
	}

	_, _, err = c.SendCommand(FTPCodeAuthOk, "AUTH TLS")

	if err != nil {
		c.Close()
		err = fmt.Errorf("failed to send AUTH command, %s", err)
		return nil, err
	}

	if ok := ifInitTLS.Load(); ! ok {
		err = getTLSConfig()

		if err != nil {
			c.Close()
			err = fmt.Errorf("failed to get TLS config, %s", err)
			return nil, err
		}
	}

	c.tlsConn = tls.Client(c.rawConn, &configTLS)
	c.setCtrlConn(c.tlsConn)

	err = c.fetchFeatures()

	if err != nil {
		c.Close()
		err = fmt.Errorf("failed to fetch features, %s", err)
		return nil, err
	}

	return c, nil
}

func (c *ftpsConn) setCtrlConn(conn net.Conn) {
	c.ctrlConn = conn
	c.reader = textproto.NewReader(bufio.NewReader(c.ctrlConn))
	c.writer = textproto.NewWriter(bufio.NewWriter(c.ctrlConn))
}

func (conn *ftpsConn) fetchFeatures() error {
	code, msg, err := conn.SendCommand(FTPCodePositive, "FEAT")

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
				conn.features[strings.ToUpper(parts[0])] = ""
			} else if len(parts) == 2 {
				conn.features[strings.ToUpper(parts[0])] = parts[1]
			}
		}
	}

	return nil
}

func (conn *ftpsConn) IsResumable() bool {
	val, ok := conn.features["REST"]

	return ok && val == "STREAM"
}

func (c *ftpsConn) SendCommand(expectCode int, format string, args ...interface{}) (int, string, error) {
	c.ctrlConn.SetWriteDeadline(time.Now().Add(c.timeout))
	err := c.writer.PrintfLine(format, args...)

	if err != nil {
		return 0, "", err
	}

	c.ctrlConn.SetReadDeadline(time.Now().Add(c.timeout))
	code, msg, err := c.reader.ReadResponse(expectCode)

	if err != nil {
		return 0, "", err
	}

	return code, msg, err
}

func (c *ftpsConn) Close() error {
	return c.ctrlConn.Close()
}

func FTPSDownload(f *NetTask) TaskError {
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

	rawURL := f.Source.URL
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

	conn, err := NewFTPSConn(addr, time.Minute)

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

	_, _, err = conn.SendCommand(FTPCodePositive, "PBSZ 0")

	if err != nil {
		err = fmt.Errorf("failed to send PBSZ command, %s", err)
		return taskError{err: err, flag: false}
	}

	_, _, err = conn.SendCommand(FTPCodePositive, "PROT P")

	if err != nil {
		err = fmt.Errorf("failed to send PORT command, %s", err)
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

	DataAddr := fmt.Sprintf("%s:%d", host, port)
	dconn, err := net.DialTimeout("tcp", DataAddr, conn.timeout)

	if err != nil {
		err = fmt.Errorf("failed to active data connection")
		return taskError{err: err, flag: false}
	}

	dconn = tls.Client(dconn, &configTLS)
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