package httpx

import (
	"crypto/tls"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
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
	Headers map[string]string
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
		Retry:   0,
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
		if err == nil {
			break
		}
		if attempts >= request.Retry {
			return nil, err
		}
		attempts++
	}

	httpResponse := &Response{}
	var err error
	contentEncoding := strings.ToLower(string(fastResp.Header.Peek("Content-Encoding")))
	switch contentEncoding {
	case "", "none", "identity":
		httpResponse.Body = fastResp.Body()
	case "gzip":
		httpResponse.Body, err = fastResp.BodyGunzip()
	case "br":
		httpResponse.Body, err = fastResp.BodyUnbrotli()
	case "deflate":
		httpResponse.Body, err = fastResp.BodyInflate()
	default:
		httpResponse.Body = fastResp.Body()
	}
	if err != nil {
		return nil, err
	}

	httpResponse.Status = fastResp.StatusCode()

	fastRespHeader := make(map[string]string)
	fastResp.Header.VisitAll(func(key, value []byte) {
		if fastRespHeader[string(key)] != "" {
			fastRespHeader[string(key)] += string(value)
		} else {
			fastRespHeader[string(key)] = string(value)
		}
	})
	httpResponse.Headers = fastRespHeader

	cookie := make(map[string]string)
	fastResp.Header.VisitAllCookie(func(key, value []byte) {
		cookie[string(key)] = strings.Split(strings.Split(string(value), ";")[0], "=")[1]
	})
	httpResponse.Cookies = cookie

	return httpResponse, nil
}
