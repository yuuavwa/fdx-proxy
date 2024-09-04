package main

import (
    "context"
    "flag"
    "fmt"
    "time"

    "github.com/gin-gonic/gin"
    proxy "github.com/yuuavwa/fdx-proxy"
)

var serverCtrl *proxy.FullDuplexServerController

func start_proxy(serverAddr, targetID string) {
    serverURL := "/api/EstablishFullDuplexChannel/" + targetID
    ctrl, err := proxy.NewFullDuplexProxyController(serverAddr, serverURL)
    if err != nil {
        panic(err)
    }
    defer ctrl.CloseProxyWSConn()
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    ctrl.RunProxyController(ctx)
}

func serverProxyHandler(c *gin.Context) {
    fmt.Println(c.Request.URL.String())
    targetID := c.Param("target_id")
    fmt.Println("proxy targetID:", targetID)
    if ctrl, err := serverCtrl.GetFullDuplexConnController(targetID); ctrl != nil && err == nil {
        fmt.Printf("FDxConnController for %v already exists.", targetID)
    }
    err := serverCtrl.AddFullDuplexConnController(c, targetID)
    if err != nil {
        panic(err)
    }
    go func() {
        for {
            ReqMethod := "GET"
            ReqHeaders := map[string]string{}
            ReqBody := ""
            go func() {
                ReqURL := "https://www.fault.xxxxxxxxxxxx/"
                status, res_body, err := serverCtrl.CallAPI(targetID, ReqMethod, ReqURL, ReqHeaders, ReqBody)
                if err != nil {
                    fmt.Println("========CallAPI err")
                    // panic(err)
                }
                fmt.Printf("req: %v, code: %v, res_body: %v\n", ReqURL, status, res_body)
            }()

            go func() {
                ReqURL := "https://www.baidu.com/"
                status, _, err := serverCtrl.CallAPI(targetID, ReqMethod, ReqURL, ReqHeaders, ReqBody)
                if err != nil {
                    fmt.Println("========CallAPI err")
                    // panic(err)
                }
                fmt.Printf("req: %v, code: %v\n", ReqURL, status)
            }()
            // fmt.Println("sleep for 1 seconds...")
            time.Sleep(time.Millisecond * 500)
        }
    }()
    fmt.Println("exit serverProxyHandler =================")
}

func start_test_server() {
    serverCtrl, _ = proxy.NewFullDuplexServerController()
    router := gin.Default()
    router.GET("/api/EstablishFullDuplexChannel/:target_id", serverProxyHandler)
    router.Run(":8080")
}

func main() {
    mode := flag.String("m", "test-server", "start mode (test-server or proxy)")
    serverAddr := flag.String("s", "localhost:8080", "server address")
    targetAddr := flag.String("t", "localhost:5000", "target identifier(target address commonly) registered to server")
    flag.Parse()

    switch *mode {
    case "test-server":
        fmt.Println("Starting server...")
        start_test_server()
    case "proxy":
        fmt.Println("Starting client...")
        start_proxy(*serverAddr, *targetAddr)
    default:
        fmt.Println("Invalid mode, use 'server' or 'client'")
    }
}
