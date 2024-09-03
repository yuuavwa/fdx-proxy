package fdxproxy

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "context"
    "github.com/gorilla/websocket"
)


type FullDuplexProxy interface {
    // close websocket connection
    CloseProxyWSConn() error

    // run Proxy controller
    RunProxyController() error
}

type FullDuplexProxyController struct {
    conn          *websocket.Conn
}

func NewFullDuplexProxyController (serverAddr, serverURL string) (*FullDuplexProxyController, error) {
    // Use scheme 'ws' to establish a full deplux channel based on websocket connection
    u := url.URL{Scheme: "ws", Host: serverAddr, Path: serverURL}
    logger.Printf("connecting to %s", u.String())

    conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        logger.Printf("dial: %v", err)
        return nil, err
    }
    logger.Printf("full duplex channel established.")
    // defer conn.Close()
    ctrl := &FullDuplexProxyController{
        conn: conn,
    }
    return ctrl, nil
}

func (c *FullDuplexProxyController) CloseProxyWSConn() error {
    logger.Println("proxy full duplex channel closing...")
    return c.conn.Close()
}

func (c *FullDuplexProxyController) RunProxyController(ctx context.Context) error {
    reqBuffer := make(chan RequestMessage, ConnReqQueueSize)
    resBuffer := make(chan ResponseMessage, ConnResQueueSize)
    done := ctx.Done()
    go func() {
        defer close(reqBuffer)
        c.loopReadRequestFromConn(ctx, reqBuffer)
    }()
    go func() {
        c.loopProxyProcess(ctx, reqBuffer, resBuffer)
    }()
    go func() {
        defer close(resBuffer)
        c.loopWriteResponseToConn(ctx, resBuffer)
    }()
    <-done
    return nil
}

func (c *FullDuplexProxyController) loopReadRequestFromConn(ctx context.Context, reqBuffer chan<- RequestMessage) error {
    // Loop to read request msg from websocket connection and deserialize into RequestMessage
    for {
        select {  
            case <-ctx.Done():  
                return fmt.Errorf("context cancelled, exiting loopReadRequestFromConn")
            default:  
                _, message, err := c.conn.ReadMessage()
                if err != nil {
                    logger.Println("read:", err)
                    break
                }
                var reqMsg RequestMessage
                err = json.Unmarshal(message, &reqMsg)
                if err != nil {
                    logger.Println("unmarshal:", err)
                    continue
                }
                reqBuffer <- reqMsg
            }
    }
}

func (c *FullDuplexProxyController) loopWriteResponseToConn(ctx context.Context, resBuffer <-chan ResponseMessage) error {
    // Loop to read response msg and write it into websocket conneciton
    for {
        select {
        case res, ok := <-resBuffer:  
            if !ok {  
                return fmt.Errorf("resBuffer channel was closed")
            }
            resJSON, err := json.Marshal(res)
            if err != nil {
                logger.Printf("marshal: %v", err)
                continue
            }
            err = c.conn.WriteMessage(websocket.TextMessage, resJSON)
            if err != nil {
                logger.Println("write:", err)
                return err
            }
        case <-ctx.Done():
            logger.Println("context cancelled, exiting loopWriteResponseToConn")
            return fmt.Errorf("context cancelled, exiting loopWriteResponseToConn")
        }
    }
}

func (c *FullDuplexProxyController) loopProxyProcess(ctx context.Context, reqBuffer <-chan RequestMessage, 
        resBuffer chan<- ResponseMessage) error {
    // Process and redirect request from proxy server
    for {
        select {
        case req, ok := <-reqBuffer:
            if !ok {  
                return fmt.Errorf("reqBuffer channel was closed")
            }
            // Start a goroutine for each single RequestMessage
            go func() error {
                // Start a new goroutine to forward request
                res, err := c.forwardRequest(req)
                if err != nil {
                    logger.Printf("forwardRequest err:%v", err)
                }
                resBuffer <- res
                return err
            }()
        case <-ctx.Done():
            logger.Println("context cancelled, exiting loopProxyProcess")
            return fmt.Errorf("context cancelled, exiting loopProxyProcess")
        }
    }
}

func (c *FullDuplexProxyController) forwardRequest(reqMsg RequestMessage) (ResponseMessage, error) {
    var resMsg = ResponseMessage{
        // MUST reuse the ReqID from request
        ReqID: reqMsg.ReqID,
        Headers: make(map[string]string),
        Status: http.StatusInternalServerError,
    }
    req, err := http.NewRequest(reqMsg.Method, reqMsg.URL, bytes.NewBuffer([]byte(reqMsg.Body)))
    if err != nil {
        logger.Printf("NewRequest err: %v", err)
        return resMsg, err
    }
    for key, value := range reqMsg.Headers {
        req.Header.Add(key, value)
    }

    client := &http.Client{}
    if reqMsg.Method == http.MethodGet {
        logger.Printf("forward---> %v %v", reqMsg.Method, reqMsg.URL)
    } else {
        logger.Printf("forward---> %v %v -d \"%v\"", reqMsg.Method, reqMsg.URL, reqMsg.Body)
    }
    resp, err := client.Do(req)
    if err != nil {
        logger.Printf("forwardRequest err: %v", err)
        return resMsg, err
    }
    defer resp.Body.Close()

    resMsg.Status = resp.StatusCode
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        logger.Printf("forwardRequest err: %v", err)
        return resMsg, err
    }
    resMsg.Body = string(body)
    for key, values := range resp.Header {
        resMsg.Headers[key] = strings.Join(values, ", ")
    }
    return resMsg, nil
}
