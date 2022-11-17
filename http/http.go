package httpx

import (
	"crypto/tls"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"io"
	"strings"
	"time"
)

type Client struct {
	ft *fasthttp.Client
}

type Requests struct {
	Url     string
	Method  string
	Headers map[string]string
	Cookies map[string]string
	Body    []byte
	Timeout int
	Retry   int
	Proxy   string
}

type Response struct {
	Headers map[string][]byte
	Cookies map[string]string
	Body    []byte
	Status  int
}

func NewClient() Client {
	config := &fasthttp.Client{
		TLSConfig:                 &tls.Config{InsecureSkipVerify: true},
		MaxConnsPerHost:           1024,
		ReadTimeout:               time.Duration(20) * time.Second,
		WriteTimeout:              time.Duration(3) * time.Second,
		NoDefaultUserAgentHeader:  true,
		MaxIdemponentCallAttempts: 1,
	}
	return Client{ft: config}
}

func NewRequest() *Requests {
	// 初始化基本的配置
	return &Requests{
		Headers: map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:103.0) Gecko/20100101 Firefox/103.0"},
		Timeout: 60,
		Retry:   3,
	}
}

func (c *Client) Request(r *Requests) (*Response, error) {
	if len(strings.TrimSpace(r.Proxy)) > 0 {
		if strings.HasPrefix(r.Proxy, "socks4://") || strings.HasPrefix(r.Proxy, "socks5://") {
			c.ft.Dial = fasthttpproxy.FasthttpSocksDialer(r.Proxy)
		} else {
			c.ft.Dial = fasthttpproxy.FasthttpHTTPDialer(r.Proxy)
		}
	}

	fastReq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(fastReq)

	fastReq.Header.SetMethod(r.Method)
	fastReq.SetRequestURI(r.Url)

	if len(r.Cookies) > 0 {
		for k, v := range r.Cookies {
			fastReq.Header.SetCookie(k, v)
		}
	}

	fastReq.SetBody(r.Body)

	fastResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(fastResp)

	attempts := 0
	for {
		err := c.ft.DoTimeout(fastReq, fastResp, time.Duration(r.Timeout)*time.Second)
		if err == nil || !isIdempotent(fastReq) && err != io.EOF {
			break
		}
		if attempts >= r.Retry {
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
