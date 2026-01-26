package network

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	archiveCDDISUrl = "https://cddis.nasa.gov/archive/"
	loginCDDISUrl   = "https://urs.earthdata.nasa.gov/login"
)

var (
	mutexAuthCDDIS sync.Mutex
	proxyAuthCDDIS atomic.Value
	proxyTimeCDDIS time.Time
)

func noRedirectFunc(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func GetCDDISProxyAuth(username, password string) TaskError {
	mutexAuthCDDIS.Lock()
	defer mutexAuthCDDIS.Unlock()

	// update every 2 hours
	_, ok := proxyAuthCDDIS.Load().(string)

	if ok && time.Since(proxyTimeCDDIS) <= time.Hour*2 {
		return nil
	}

	client := http.Client{Timeout: time.Minute}

	// 1st request
	request, _ := http.NewRequest(http.MethodGet, archiveCDDISUrl, nil)
	request.Header.Add("User-Agent", HTTPUserAgent)
	response, err := client.Do(request)

	if err != nil {
		err = fmt.Errorf("in the 1st request, %s", err)
		return NewTaskError(err, true)
	} else if response.StatusCode != http.StatusOK {
		response.Body.Close()
		err = fmt.Errorf("in the 1st request, invalid response status code %d", response.StatusCode)
		return NewTaskError(err, true)
	}

	cookies := response.Header["Set-Cookie"]

	if len(cookies) == 0 {
		response.Body.Close()
		err = fmt.Errorf(`in the 1st request, no "Set-Cookie"`)
		return NewTaskError(err, false)
	}

	body, err := io.ReadAll(response.Body)

	if err != nil || len(body) == 0 {
		response.Body.Close()
		err = fmt.Errorf("in the 1st request, response body fault")
		return NewTaskError(err, false)
	}

	bodyStr := strings.ReplaceAll(string(body), "\n", "")
	response.Body.Close()

	formExp := regexp.MustCompile("<form(.*?)</form>")
	formStr := formExp.FindString(bodyStr)

	authExp := regexp.MustCompile(`name="authenticity_token" value="(.*?)"`)
	authStr := strings.ReplaceAll(authExp.FindString(formStr), `"`, "")
	authVal := authStr[strings.LastIndexByte(authStr, '=')+1:]

	clientExp := regexp.MustCompile(`name="client_id" id="client_id" value="(.*?)"`)
	clientStr := strings.ReplaceAll(clientExp.FindString(formStr), `"`, "")
	clientVal := clientStr[strings.LastIndexByte(clientStr, '=')+1:]

	redirectExp := regexp.MustCompile(`name="redirect_uri" id="redirect_uri" value="(.*?)"`)
	redirectStr := strings.ReplaceAll(redirectExp.FindString(formStr), `"`, "")
	redirectVal := redirectStr[strings.LastIndexByte(redirectStr, '=')+1:]

	responseExp := regexp.MustCompile(`name="response_type" id="response_type" value="(.*?)"`)
	responseStr := strings.ReplaceAll(responseExp.FindString(formStr), `"`, "")
	responseVal := responseStr[strings.LastIndexByte(responseStr, '=')+1:]

	stateExp := regexp.MustCompile(`name="state" id="state" value="(.*?)"`)
	stateStr := strings.ReplaceAll(stateExp.FindString(formStr), `"`, "")
	stateVal := stateStr[strings.LastIndexByte(stateStr, '=')+1:]

	stayExp := regexp.MustCompile(`name="stay_in" id="stay_in" value="(.*?)"`)
	stayStr := strings.ReplaceAll(stayExp.FindString(formStr), `"`, "")
	stayVal := stayStr[strings.LastIndexByte(stayStr, '=')+1:]

	commitExp := regexp.MustCompile(`name="commit" value="(.*?)"`)
	commitStr := strings.ReplaceAll(commitExp.FindString(formStr), `"`, "")
	commitVal := commitStr[strings.LastIndexByte(commitStr, '=')+1:]

	if authVal == "" || clientVal == "" || redirectVal == "" || responseVal == "" ||
		stateVal == "" || stayVal == "" || commitVal == "" {
		err = fmt.Errorf("in the 1st request, html parsing fault")
		return NewTaskError(err, false)
	}

	// 2nd request
	data := url.Values{}
	data.Set("authenticity_token", authVal)
	data.Set("username", username)
	data.Set("password", password)
	data.Set("client_id", clientVal)
	data.Set("redirect_uri", redirectVal)
	data.Set("response_type", responseVal)
	data.Set("state", stateVal)
	data.Set("stay_in", stayVal)
	data.Set("commit", commitVal)
	dataStr := data.Encode()

	request, _ = http.NewRequest(http.MethodPost, loginCDDISUrl, strings.NewReader(dataStr))
	request.Header.Set("User-Agent", HTTPUserAgent)
	request.Header.Set("content-type", "application/x-www-form-urlencoded")

	for _, cookie := range cookies {
		request.Header.Add("Cookie", cookie)
	}

	// if redirect is enabled here, the final response id wrong, and the reason has not be founded,
	// maybe because cookies is not passed correctly to the redirect
	client = http.Client{Timeout: time.Minute, CheckRedirect: noRedirectFunc}
	response, err = client.Do(request)

	if err != nil {
		err = fmt.Errorf("in the 2nd request (1st half), %s", err)
		return NewTaskError(err, true)
	} else if response.StatusCode != http.StatusFound {
		response.Body.Close()
		err = fmt.Errorf("in the 2nd request (1st half), invalid response status code %d", response.StatusCode)
		return NewTaskError(err, true)
	}

	response.Body.Close()
	cookies = response.Header["Set-Cookie"]

	if len(cookies) == 0 {
		err = fmt.Errorf(`in the 2nd request (1st half), no "Set-Cookie"`)
		return NewTaskError(err, false)
	}

	urlsTmp := response.Header["Location"]

	if len(urlsTmp) == 0 {
		err = fmt.Errorf(`in the 2nd request (1st half), no "Location"`)
		return NewTaskError(err, false)
	}

	client = http.Client{Timeout: time.Minute}
	request, _ = http.NewRequest(http.MethodGet, urlsTmp[0], nil)
	request.Header.Set("User-Agent", HTTPUserAgent)

	var cookieTmp string

	for idx, cookie := range cookies {
		if idx < len(cookies)-1 {
			cookieTmp += strings.Split(cookie, ";")[0] + "; "
		} else {
			cookieTmp += strings.Split(cookie, ";")[0]
		}

	}

	request.Header.Set("Cookie", cookieTmp)
	response, err = client.Do(request)

	if err != nil {
		err = fmt.Errorf("in the 2nd request (2nd half), %s", err)
		return NewTaskError(err, true)
	} else if response.StatusCode != http.StatusOK {
		response.Body.Close()
		err = fmt.Errorf("in the 2nd request (2nd half), invalid response status code %d", response.StatusCode)
		return NewTaskError(err, true)
	}

	body, err = io.ReadAll(response.Body)

	if err != nil || len(body) == 0 {
		response.Body.Close()
		err = fmt.Errorf("in the 2nd request (2nd half), response body fault")
		return NewTaskError(err, false)
	}

	bodyStr = strings.ReplaceAll(string(body), "\n", "")
	response.Body.Close()

	urlExp := regexp.MustCompile(`var redirectURL = "(.*?)"`)
	urlStr := urlExp.FindString(bodyStr)
	urlVal := urlStr[strings.IndexByte(urlStr, '"')+1 : len(urlStr)-1]

	if urlVal == "" {
		err = fmt.Errorf("in the 2nd request (2nd half), body parsing fault")
		return NewTaskError(err, false)
	}

	// 3rd request
	client = http.Client{Timeout: time.Minute, CheckRedirect: noRedirectFunc}
	request, _ = http.NewRequest(http.MethodGet, urlVal, nil)
	request.Header.Set("User-Agent", HTTPUserAgent)
	response, err = client.Do(request)

	if err != nil {
		err = fmt.Errorf("in the 3rd request, %s", err)
		return NewTaskError(err, true)
	} else if response.StatusCode != http.StatusFound {
		response.Body.Close()
		err = fmt.Errorf("in the 3rd request, invalid response status code %d", response.StatusCode)
		return NewTaskError(err, true)
	}

	response.Body.Close()
	cookies = response.Header["Set-Cookie"]

	if len(cookies) == 0 {
		err = fmt.Errorf(`in the 3rd request, no "Set-Cookie"`)
		return NewTaskError(err, false)
	}

	for _, cookie := range cookies {
		vals := strings.Split(cookie, ";")

		for _, val := range vals {
			if strings.Contains(val, "ProxyAuth=") {
				proxyAuthCDDIS.Store(strings.TrimSpace(val))
				proxyTimeCDDIS = time.Now()
				return nil
			}
		}
	}

	return NewTaskError(fmt.Errorf("ProxyAuth not found"), false)
}

func CDDISDownLoad(f *NetworkTask) TaskError {
	// initialize status of the task
	var idx int64
	var flag int

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

	// make request to download the file
	client := http.Client{CheckRedirect: noRedirectFunc}
	ctx, cancel := context.WithCancel(context.TODO())
	timer := time.AfterFunc(time.Minute, func() { cancel() })
	request, _ := http.NewRequest(http.MethodGet, f.Source.Url, nil)
	request.Header.Add("User-Agent", HTTPUserAgent)
	request.Header.Add("Range", fmt.Sprintf("bytes=%d-", idx))
	request = request.WithContext(ctx)
	timer.Reset(time.Minute)

	var terr TaskError

	if _, ok := proxyAuthCDDIS.Load().(string); !ok || time.Since(proxyTimeCDDIS) > 2*time.Hour {
		for i := 0; i < 5; i++ {
			terr = GetCDDISProxyAuth(f.Source.UserName, f.Source.Password)

			if terr == nil || !terr.IsTemporary() {
				break
			}
		}
	}

	if terr != nil {
		err := fmt.Errorf("failed to get ProxyAuth of CDDIS, %s", terr)
		return NewTaskError(err, false)
	}

	cookie, _ := proxyAuthCDDIS.Load().(string)

	request.Header.Set("Cookie", cookie)
	response, err := client.Do(request)

	if err != nil {
		return NewTaskError(err, true)
	} else if response.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		response.Body.Close()
		return nil
	} else if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent {
		response.Body.Close()
		err = fmt.Errorf("invalid response status %d", response.StatusCode)
		return NewTaskError(err, true)
	}

	defer response.Body.Close()

	fp, err := os.OpenFile(f.Path, flag, 0664)

	if err != nil {
		return NewTaskError(err, false)
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
			if strings.Contains(err.Error(), "context canceled") {
				return NewTaskError(err, true)
			} else {
				return NewTaskError(err, false)
			}
		}
	}
}
