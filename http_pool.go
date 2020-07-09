package frame

import (
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
)

var httpLock sync.Mutex

type curlClientConfig struct {
	KeepAlive          bool   `toml:"keepAlive"`
	TimeOut            int    `toml:"timeout"`
	MaxIdleConn        int    `toml:"maxIdleConns"`
	MaxIdleConnPerHost int    `toml:"maxIdleConnsPerHost"`
	IdleConnTimeout    int    `toml:"idleConnTimeout"`
	ProxyUrl           string `toml:"proxyUrl"`
}

var httpClientPool map[string]*http.Client

func getHttpClient(clientGroup string) *http.Client {
	if httpClientPool == nil {
		httpClientPool = make(map[string]*http.Client)
	}
	clientConfig := &curlClientConfig{}
	err := App().Env(clientGroup, clientConfig)
	if err != nil {
		panic(HttpError.Error() + ":" + err.Error())
	}
	keepAlive := clientConfig.KeepAlive
	timeout := clientConfig.TimeOut
	maxIdleConns := clientConfig.MaxIdleConn
	maxIdleConnsPerHost := clientConfig.MaxIdleConnPerHost
	idleConnTimeout := clientConfig.IdleConnTimeout
	proxyUrl := clientConfig.ProxyUrl

	keepAliveStr := "0"
	disableKeepAlive := false
	if keepAlive {
		keepAliveStr = "1"
		disableKeepAlive = true
	}
	cacheKey := keepAliveStr + strconv.Itoa(timeout) + strconv.Itoa(maxIdleConns) +
		strconv.Itoa(maxIdleConnsPerHost) + strconv.Itoa(idleConnTimeout) + proxyUrl
	cache, ok := httpClientPool[cacheKey]
	if ok {
		return cache
	}
	httpLock.Lock()
	defer httpLock.Unlock()
	var clientObj *http.Client
	if proxyUrl != "" {
		proxy, err := url.Parse(proxyUrl)
		if err != nil {
			//报错
			msg := map[string]interface{}{
				"error": err.Error(),
				"stack": string(debug.Stack()),
			}
			App().Log.Error(msg, LogClientError)
			return nil
		}
		clientObj = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxy), //使用代理的方法
				DialContext: (&net.Dialer{ //用来创建http（非https）连接
					Timeout:   30 * time.Second, //建立Tcp链接超时时间
					KeepAlive: 30 * time.Second, //维持keepalive 多久发送一次Keep-Alive报文
				}).DialContext,
				MaxIdleConns:        maxIdleConns,                                 //连接池对所有host的最大链接数量
				MaxIdleConnsPerHost: maxIdleConnsPerHost,                          //连接池对每个host的最大链接数量
				IdleConnTimeout:     time.Duration(idleConnTimeout) * time.Second, //空闲timeout设置，也即socket在该时间内没有交互则自动关闭连接,有交互则重新计时
				DisableKeepAlives:   disableKeepAlive,                             //是否关闭重用连接
			},
			Timeout: time.Duration(timeout) * time.Second, //处理单个请求最长时间
		}
	} else {
		clientObj = &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{ //用来创建http（非https）连接
					Timeout:   30 * time.Second, //建立Tcp链接超时时间
					KeepAlive: 30 * time.Second, //维持keepalive 多久发送一次Keep-Alive报文
				}).DialContext,
				MaxIdleConns:        maxIdleConns,                                 //连接池对所有host的最大链接数量
				MaxIdleConnsPerHost: maxIdleConnsPerHost,                          //连接池对每个host的最大链接数量
				IdleConnTimeout:     time.Duration(idleConnTimeout) * time.Second, //空闲timeout设置，也即socket在该时间内没有交互则自动关闭连接,有交互则重新计时
				DisableKeepAlives:   disableKeepAlive,                             //是否关闭重用连接
			},
			Timeout: time.Duration(timeout) * time.Second, //处理单个请求最长时间
		}
	}
	httpClientPool[cacheKey] = clientObj
	return httpClientPool[cacheKey]
}

//关闭连接池中所有闲置连接
func closeHttpClient() {
	if httpClientPool == nil {
		return
	}
	for _, v := range httpClientPool {
		v.CloseIdleConnections()
	}
}
