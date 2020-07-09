package frame

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type requestInfo struct {
	body   string
	header map[string]interface{}
}

//curl结构体
type Curl struct {
	httpClient     *http.Client //调用 GetHttpClient 方法构造出来
	httpClientOnce sync.Once
	httpCode       int
	header         map[string][]string
	body           string
	requestInfo    *requestInfo
	ClientGroup    string //client配置 如 curl/client_default
}

func (curl *Curl) Get(uri string, requestMapHeaders ...map[string]interface{}) (string, error) {
	requestMap := make(map[string]interface{}, 0)
	header := make(map[string]interface{}, 0)
	if len(requestMapHeaders) > 0 {
		requestMap = requestMapHeaders[0]
	}
	if len(requestMapHeaders) > 1 {
		header = requestMapHeaders[1]
	}
	return curl.request(http.MethodGet, uri, requestMap, header, false)
}

func (curl *Curl) Post(uri string, requestMapHeaders ...map[string]interface{}) (string, error) {
	requestMap := make(map[string]interface{}, 0)
	header := make(map[string]interface{}, 0)
	if len(requestMapHeaders) > 0 {
		requestMap = requestMapHeaders[0]
	}
	if len(requestMapHeaders) > 1 {
		header = requestMapHeaders[1]
	}
	return curl.request(http.MethodPost, uri, requestMap, header, false)
}

func (curl *Curl) PostByMultiPart(uri string, requestMapHeaders ...map[string]interface{}) (string, error) {
	requestMap := make(map[string]interface{}, 0)
	header := make(map[string]interface{}, 0)
	if len(requestMapHeaders) > 0 {
		requestMap = requestMapHeaders[0]
	}
	if len(requestMapHeaders) > 1 {
		header = requestMapHeaders[1]
	}
	return curl.request(http.MethodPost, uri, requestMap, header, true)
}

func (curl *Curl) HttpCode() int {
	return curl.httpCode
}

func (curl *Curl) Header() map[string][]string {
	return curl.header
}

func (curl *Curl) request(method string, uri string, requestMap map[string]interface{}, header map[string]interface{}, multipart bool) (string, error) {
	curl.resetBefore()
	//构造url参数
	remoteUrl, err := url.Parse(uri)
	defer func() {
		curlErrorHandle(err, curl)
	}()
	if err != nil {
		return "", err
	}
	bodyString := ""
	if method == http.MethodGet || method == http.MethodPost {
		if requestMap != nil && !multipart {
			queryValues := url.Values{}
			for k, v := range requestMap {
				switch reflect.TypeOf(v).Kind() {
				case reflect.Slice:
					queryValues[k+"[]"] = convertSliceToStrings(v)
				default:
					queryValues[k] = []string{convertToString(v)}
				}
			}
			if method == http.MethodGet {
				values := remoteUrl.Query()
				if values != nil {
					for k, v := range values {
						queryValues[k] = v
					}
				}
				remoteUrl.RawQuery = queryValues.Encode()
			} else {
				bodyString = queryValues.Encode()
			}
		}

		if multipart {
			mJson, _ := json.Marshal(requestMap)
			bodyString = string(mJson)
		}
	}

	req, err := http.NewRequest(method, remoteUrl.String(), strings.NewReader(bodyString))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Expect", "")
	for h, v := range header {
		req.Header.Set(h, v.(string))
	}
	curl.requestInfo.body = bodyString
	curl.requestInfo.header = header
	//没有设置 HttpClient 默认调用 defaultHttpClient()
	if curl.httpClient == nil {
		curl.httpClientOnce.Do(func() {
			if curl.ClientGroup == "" {
				curl.ClientGroup = "curl/client_default"
			}
			curl.httpClient = getHttpClient(curl.ClientGroup)
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //后发送报文的最长时间,文件上传时可能很长 超时客户端就自己取消报504
	defer cancel()
	resp, err := curl.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	curl.httpCode = resp.StatusCode
	curl.header = resp.Header
	defer func() {
		if resp != nil {
			_ = resp.Body.Close()
		}
	}()
	respBody, err := ioutil.ReadAll(resp.Body)
	curl.body = string(respBody)
	if err != nil {
		return "", err
	}
	//非200 人为构造错误,方便记录日志
	if curl.httpCode != 200 {
		err = HttpFailError
	}
	return string(respBody), nil
}

func (curl *Curl) resetBefore() {
	curl.httpCode = 0
	curl.header = make(map[string][]string)
	curl.body = ""
	curl.requestInfo = &requestInfo{}
}

func curlErrorHandle(err error, curl *Curl) {
	if err != nil {
		msg := map[string]interface{}{
			"request":   curl.requestInfo,
			"http_code": curl.httpCode,
			"header":    curl.header,
			"body":      curl.body,
			"error":     err.Error(),
			"stack":     string(debug.Stack()),
		}
		App().Log.Error(msg, LogCurlError)
	}
}
