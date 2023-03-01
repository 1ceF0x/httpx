package httpx

import (
	"fmt"
	"testing"
)

func TestGet(t *testing.T) {
	config := &Client{
		SSLVerify: true,
		Proxy:     "192.168.1.248:8080",
	}
	// 使用自定义配置初始化Client
	InitClient(config)

	req := NewRequest()
	req.Url = "http://www.google.com"
	req.Method = GET
	//req.Proxy = "192.168.1.248:8080"
	resp, err := req.Request()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(resp.Headers["Server"])
	}
}
