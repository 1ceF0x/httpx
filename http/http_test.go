package httpx

import (
	"fmt"
	httpx "github.com/1ceF0x/httpx/types"
	"testing"
)

func TestGet(t *testing.T) {
	Init()
	req := NewRequest()
	req.Url = "https://www.google.com"
	req.Method = httpx.MethodGet
	//req.Proxy = "127.0.0.1:8080"
	fmt.Println(req)
	resp, err := HTTPRequest(req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(resp.Headers["server"])
	}
}
