package main

import (
    "context"
    "flag"
    "fmt"
    "time"
    "github.com/gin-gonic/gin"
    proxy "github.com/yuuavwa/fdx-proxy"
)


func start_client(serverAddr, targetID string) {
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
    ctrl, err := proxy.NewFullDuplexServerController(c, targetID)
    if err != nil {
        panic(err)
    }
    
    for {
        ReqMethod := "GET"        
        ReqHeaders := map[string]string{}
        ReqBody := ""
        go func() {
            ReqURL := "https://www.baiduaa.com/"
            status, res_body, err := ctrl.CallAPI(targetID, ReqMethod, ReqURL, ReqHeaders, ReqBody)
            if err != nil {
                fmt.Println("========CallAPI err")
                // panic(err)
            }
            fmt.Printf("req: %v, code: %v, res_body: %v\n", ReqURL, status, res_body)
        }()
        
        go func() {
            ReqURL := "https://www.baidu.com/"
            status, _, err := ctrl.CallAPI(targetID, ReqMethod, ReqURL, ReqHeaders, ReqBody)
            if err != nil {
                
                fmt.Println("========CallAPI err")
                // panic(err)
            }
            fmt.Printf("req: %v, code: %v\n", ReqURL, status)
        }()
        // fmt.Println("sleep for 1 seconds...")
        time.Sleep(time.Millisecond * 100)
    }
}

func start_test_server() {
    router := gin.Default()
    router.GET("/api/EstablishFullDuplexChannel/:target_id", serverProxyHandler)
    router.Run(":8080")
}

func main() {
    mode := flag.String("m", "test-server", "start mode (test-server or client)")
    serverAddr := flag.String("s", "localhost:8080", "server address")
    targetAddr := flag.String("t", "localhost:5000", "target identifier(target address commonly) registered to server")
    flag.Parse()
  
    switch *mode {
    case "test-server":
        fmt.Println("Starting server...")
        start_test_server()
    case "client":  
        fmt.Println("Starting client...")
        start_client(*serverAddr, *targetAddr)
    default:  
        fmt.Println("Invalid mode, use 'server' or 'client'")
    }
}
