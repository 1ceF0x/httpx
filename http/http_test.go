package httpx

import (
	"fmt"
	"testing"
)

func TestRequest(t *testing.T) {
	c := NewClient()
	req := Requests{
		Url:     "",
		Method:  "",
		Headers: nil,
		Cookies: nil,
		Body:    nil,
		Timeout: 0,
		Retry:   0,
		Proxy:   "",
	}
	//req := NewRequest()
	req.Url = "https://www.baidu.com"
	req.Method = GET
	req.Proxy = "127.0.0.1:8080"
	fmt.Println(req)
	resp, err := c.Request(&req)
	if err != nil {
		fmt.Println(err)
	} else {
		for k, v := range resp.Headers {
			fmt.Println(k, ":", string(v))
		}
	}
	req.Url = "https://www.bing.com"
	fmt.Println(req)
	resp, err = c.Request(&req)
	if err != nil {
		fmt.Println(err)
	} else {
		for k, v := range resp.Headers {
			fmt.Println(k, ":", string(v))
		}
	}
}
