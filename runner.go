package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	proxy2 "golang.org/x/net/proxy"
	"golang.org/x/xerrors"
	"io"
	"log"
	"net/http"
	"os"
	"socks5_proxy/socksmitm"
)

func HmHook() {

	err := socksmitm.PacListenAndServe(context.TODO(), pacPort, socksPort)
	if err != nil {
		log.Printf("%+v\n", err)
		return
	}

	// TODO: 设置证书认证
	pkcs12Data, err := os.ReadFile("更改为你所需的证书文件!")
	if err != nil {
		log.Printf("%+v\n", err)
		return
	}
	var dialer = proxy2.FromEnvironment()
	mux := socksmitm.NewMux(dialer)
	mux.SetDefaultHTTPRoundTrip(socksmitm.NormalRoundTrip)

	// TODO: 对指定url自定义handler进行处理
	mux.Register("指定url", ACSHandlerFunc)

	// TODO: 设置证书对应的密码
	server, err := socksmitm.NewSocks5Server(mux, pkcs12Data, "证书对应的密码")
	if err != nil {
		log.Printf("%+v\n", err)
		return
	}
	server.RegisterRootCa() // 注册 root.ca 处理器, 用于浏览器获取ca证书
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	err = server.Run(ctx, fmt.Sprintf("0.0.0.0:%d", socksPort))
	if err != nil {
		log.Printf("%+v\n", err)
		return
	}

}

// ACSHandlerFunc 自定义 handler 用于处理需要记录或更改的字段
var ACSHandlerFunc socksmitm.HTTPRoundTrip = func(req *http.Request) (*http.Response, error) {
	//log.Println("req:", req.Method, req.Proto, req.URL.Scheme, req.Host, req.URL.Path)
	//if req.Body == nil || req.Body == http.NoBody {
	//	return nil, xerrors.New("block")
	//}
	reqBodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, xerrors.Errorf("%w", err)
	}
	//log.Println("req:", string(reqBodyBytes))
	req.ContentLength = int64(len(reqBodyBytes))
	req.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
	resp, err := socksmitm.NormalRoundTrip(req)
	if err != nil {
		return nil, xerrors.Errorf("%w\n", err)
	}
	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Errorf("%w\n", err)
	}
	decodeResp := respBodyBytes
	if resp.Header.Get("Content-Encoding") == "gzip" {
		greader, err := gzip.NewReader(bytes.NewReader(respBodyBytes))
		if err == nil {
			decodeResp, err = io.ReadAll(greader)
			if err != nil {
				log.Printf("%+v\n", err)
			}
		} else {
			log.Printf("%+v\n", err)
		}

	}
	_ = decodeResp
	url := req.URL.RequestURI()
	// 如果匹配到了路径
	_ = url

	req.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
	resp.Body = io.NopCloser(bytes.NewReader(respBodyBytes))
	if err != nil {
		log.Printf("%+v\n", err)
		return resp, nil
	}
	return resp, nil
}
