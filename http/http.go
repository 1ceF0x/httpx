package httpx

import (
	"crypto/tls"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"io"
	"strings"
	"time"
)

type Request struct {
	Url     string
	Method  string
	Headers map[string]string
	Body    []byte
	Timeout int
	Retry   int
	Proxy   string
}

type Response struct {
	Headers map[string]string
	Cookies map[string]string
	Body    []byte
	Status  int32
}

var FT *fasthttp.Client

// TODO 初始化参数
func Init() {
	FT = &fasthttp.Client{
		TLSConfig:                 &tls.Config{InsecureSkipVerify: true},
		MaxConnsPerHost:           1024,
		ReadTimeout:               time.Duration(20) * time.Second,
		WriteTimeout:              time.Duration(3) * time.Second,
		NoDefaultUserAgentHeader:  true,
		MaxIdemponentCallAttempts: 1,
	}
}

func NewRequest() *Request {
	return &Request{
		Headers: map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:103.0) Gecko/20100101 Firefox/103.0"},
		Timeout: 3,
		Retry:   3,
	}
}

func HTTPRequest(httpRequest *Request) (*Response, error) {
	if len(strings.TrimSpace(httpRequest.Proxy)) > 0 {
		if strings.HasPrefix(httpRequest.Proxy, "socks4://") || strings.HasPrefix(httpRequest.Proxy, "socks5://") {
			FT.Dial = fasthttpproxy.FasthttpSocksDialer(httpRequest.Proxy)
		} else {
			FT.Dial = fasthttpproxy.FasthttpHTTPDialer(httpRequest.Proxy)
		}
	}

	fastReq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(fastReq)

	fastReq.Header.SetMethod(httpRequest.Method)
	fastReq.SetRequestURI(httpRequest.Url)

	var rawHeader strings.Builder
	newHeader := make(map[string]string)
	for k, v := range httpRequest.Headers {
		fastReq.Header.Set(k, v)
		newHeader[k] = v
		rawHeader.WriteString(k)
		rawHeader.WriteString(": ")
		rawHeader.WriteString(v)
		rawHeader.WriteString("\n")
	}

	fastReq.SetBody(httpRequest.Body)

	fastResp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(fastResp)

	attempts := 0
	for {
		err := FT.DoTimeout(fastReq, fastResp, time.Duration(httpRequest.Timeout)*time.Second)
		if err == nil || !isIdempotent(fastReq) && err != io.EOF {
			break
		}
		if attempts >= httpRequest.Retry {
			return nil, err
		}
		attempts++
	}

	httpResponse := &Response{}
	httpResponse.Status = int32(fastResp.StatusCode())
	newHeader2 := make(map[string]string)
	respHeaderSlice := strings.Split(fastResp.Header.String(), "\r\n")
	for _, h := range respHeaderSlice {
		hslice := strings.SplitN(h, ":", 2)
		if len(hslice) != 2 {
			continue
		}
		k := strings.ToLower(hslice[0])
		v := strings.TrimLeft(hslice[1], " ")
		if newHeader2[k] != "" {
			newHeader2[k] += v
		} else {
			newHeader2[k] = v
		}
	}
	httpResponse.Headers = newHeader2

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
