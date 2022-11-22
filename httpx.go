package httpx

import (
	"crypto/tls"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"io"
	"strings"
	"time"
)

type Requests struct {
	Url     string
	Method  string
	Headers map[string]string
	Cookies map[string]string
	Body    []byte
	Timeout int
	Retry   int
	// Requests配置代理 单独为请求配置代理 (随机代理或需要随时切换代理时使用)
	Proxy string
}

type Response struct {
	Headers map[string][]byte
	Cookies map[string]string
	Body    []byte
	Status  int
}

type Client struct {
	SSLVerify                bool
	MaxConnsPerHost          int
	ReadTimeout              int
	WriteTimeout             int
	NoDefaultUserAgentHeader bool
	// Client配置代理 全局使用 (动态代理或无需切换代理时使用)
	Proxy string
}

// HTTPX Client默认配置
var FT = &fasthttp.Client{
	TLSConfig:                 &tls.Config{InsecureSkipVerify: true},
	MaxConnsPerHost:           1024,
	ReadTimeout:               time.Duration(20) * time.Second,
	WriteTimeout:              time.Duration(3) * time.Second,
	NoDefaultUserAgentHeader:  true,
	MaxIdemponentCallAttempts: 1,
}

// 初始化Client自定义配置
func InitClient(config *Client) {
	FT.TLSConfig = &tls.Config{InsecureSkipVerify: !config.SSLVerify}
	FT.MaxConnsPerHost = config.MaxConnsPerHost
	FT.ReadTimeout = time.Duration(config.ReadTimeout) * time.Second
	FT.WriteTimeout = time.Duration(config.WriteTimeout) * time.Second
	FT.NoDefaultUserAgentHeader = config.NoDefaultUserAgentHeader

	if len(strings.TrimSpace(config.Proxy)) > 0 {
		if strings.HasPrefix(config.Proxy, "socks4://") || strings.HasPrefix(config.Proxy, "socks5://") {
			FT.Dial = fasthttpproxy.FasthttpSocksDialer(config.Proxy)
		} else {
			FT.Dial = fasthttpproxy.FasthttpHTTPDialer(config.Proxy)
		}
	}
}

// 创建request
func NewRequest() *Requests {
	// 初始化默认配置
	return &Requests{
		Headers: map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:103.0) Gecko/20100101 Firefox/103.0"},
		Timeout: 60,
		Retry:   3,
	}
}

func (request *Requests) Request() (*Response, error) {
	if len(strings.TrimSpace(request.Proxy)) > 0 {
		if strings.HasPrefix(request.Proxy, "socks4://") || strings.HasPrefix(request.Proxy, "socks5://") {
			FT.Dial = fasthttpproxy.FasthttpSocksDialer(request.Proxy)
		} else {
			FT.Dial = fasthttpproxy.FasthttpHTTPDialer(request.Proxy)
		}
	}

	fastReq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(fastReq)

	fastReq.Header.SetMethod(request.Method)
	fastReq.SetRequestURI(request.Url)

	for k, v := range request.Headers {
		fastReq.Header.Set(k, v)
	}

	if len(request.Cookies) > 0 {
		for k, v := range request.Cookies {
			fastReq.Header.SetCookie(k, v)
		}
	}

	fastReq.SetBody(request.Body)

	fastResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(fastResp)

	attempts := 0
	for {
		err := FT.DoTimeout(fastReq, fastResp, time.Duration(request.Timeout)*time.Second)
		if err == nil || !isIdempotent(fastReq) && err != io.EOF {
			break
		}
		if attempts >= request.Retry {
			return nil, err
		}
		attempts++
	}

	var fastRespHeader map[string][]byte
	fastRespHeader = make(map[string][]byte)

	// 响应头转换map
	fastResp.Header.VisitAll(func(key, value []byte) {
		if fastRespHeader[string(key)] != nil {
			fastRespHeader[string(key)] = append(fastRespHeader[string(key)], value...)
		} else {
			fastRespHeader[string(key)] = value
		}
	})

	httpResponse := &Response{}
	httpResponse.Status = fastResp.StatusCode()
	httpResponse.Headers = fastRespHeader

	cookie := make(map[string]string)
	fastResp.Header.VisitAllCookie(func(key, value []byte) {
		cookie[string(key)] = strings.Split(strings.Split(string(value), ";")[0], "=")[1]
	})
	httpResponse.Cookies = cookie

	httpResponse.Body = fastResp.Body()
	return httpResponse, nil
}

func isIdempotent(req *fasthttp.Request) bool {
	return req.Header.IsGet() || req.Header.IsHead() || req.Header.IsPut()
}
