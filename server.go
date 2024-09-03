package fdxproxy

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    "github.com/google/uuid"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
)


type FullDuplexServer interface {
    // Call an API through a FullDuplex websocket connection
    CallAPI(addr, method, url string, headers map[string]string, req_body string) (status int, res_body string, err error)
}

type FullDuplexConnController struct {
    conn          *websocket.Conn
    // Request buffer read by the write conn goroutine and written to the connection
    reqBuffer     chan RequestMessage
}

type FullDuplexServerController struct {
    // CurrentMap: map[string]FullDuplexConnController
    connCtrls     *CurrentMap
    // CurrentMap: map[string]chan ResponseMessage
    resMsgs       *CurrentMap
}


func NewFullDuplexServerController(c *gin.Context, targetID string) (*FullDuplexServerController, error) {
    // new and initialize FullDuplex websocket connection controller
    if targetID == "" {
        logger.Printf("no target address found for this call")
        return nil, fmt.Errorf("no target address found for this call")
    }
	var upgrader = websocket.Upgrader{} 
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        logger.Printf("upgrade:", err)
        return nil, err
    }
    // defer conn.Close()
    connCtrls := NewCurrentMap()
    connCtrls.Set(targetID, FullDuplexConnController {
        conn: conn,
        reqBuffer: make(chan RequestMessage, ConnReqQueueSize),
    })
    ctrl := &FullDuplexServerController {
        connCtrls: connCtrls,
        resMsgs: NewCurrentMap(),
    }
    logger.Printf("websocket upgrade for %v", targetID)
    go ctrl.loopWriteRequestToConn(targetID)
    go ctrl.loopReadResponseFromConn(targetID)
    return ctrl, nil
}

func (s *FullDuplexServerController) loopReadResponseFromConn(connKey string) error {
    // Loop to read messages from the websocket connection and deserialize into ResponseMessage
    connCtrl, exists := s.connCtrls.Get(connKey)
    if !exists {
        logger.Printf("Connection controller for %s is not found.", connKey)
        return fmt.Errorf("connection controller for %s is not found", connKey)
    }
    for {
        _, message, err := connCtrl.(FullDuplexConnController).conn.ReadMessage()
        if err != nil {
            logger.Println("read:", err)
            return err
        }

        var resp ResponseMessage
        err = json.Unmarshal(message, &resp)
        if err != nil {
            logger.Println("unmarshal:", err)
            continue
        }
        // Do not create a new channel and set it here, 
        // as it could cause the resMsgs reading goroutine which may run ahead failing to get the channel.
        // The channel should be created and set by the resMsgs reading goroutine
        resCh, exists := s.resMsgs.Get(resp.ReqID)
        if !exists {
            logger.Printf("Get no res msg channel for %v", resp.ReqID)
            continue
        }
        // here to write into the gotten channel
        resCh.(chan ResponseMessage) <- resp
	}
}

func (s *FullDuplexServerController) loopWriteRequestToConn(connKey string) error {
    // Loop to write messages from the request channel into the websocket connection  
    connCtrl, exists := s.connCtrls.Get(connKey)
    if !exists {
        logger.Printf("Connection controller for %s is not found.", connKey)
        return fmt.Errorf("connection controller for %s is not found", connKey)
    }
    for req := range connCtrl.(FullDuplexConnController).reqBuffer {
        reqJSON, err := json.Marshal(req)
        if err != nil {
            logger.Printf("marshal: %v", err)
        }
        err = connCtrl.(FullDuplexConnController).conn.WriteMessage(websocket.TextMessage, reqJSON)
        if err != nil {
            logger.Println("write:", err)
            return err
        }
    }
    logger.Printf("Request channel for %s is closed, exiting loop", connKey)
    return nil
}

func (s *FullDuplexServerController) CallAPI(addr, method, url string, headers map[string]string, 
        req_body string) (status int, res_body string, err error) {

    rid := uuid.New().String()
    reqMsg := RequestMessage {
        ReqID: rid,
        Method: method,
        URL: url,
        Headers: headers,
        Body: req_body,
    }
    // Reserve an extra buffer to prevent blocking in loopReadResponseFromConn on a single channel write
    s.resMsgs.Set(rid, make(chan ResponseMessage, 1))
    connCtrl, exists := s.connCtrls.Get(addr)
    if !exists {
        logger.Printf("Connection controller for %s is not found.", addr)
        return http.StatusInternalServerError, "", fmt.Errorf("connection controller for %s is not found", addr)
    }
    connCtrl.(FullDuplexConnController).reqBuffer <- reqMsg

    // Read the response from the corresponding channel
    resCh, exists := s.resMsgs.Get(rid)
    if !exists {
        logger.Printf("Get no res msg channel for %v", rid)
        return http.StatusInternalServerError, "", fmt.Errorf("get no res msg channel for %v", rid)
    }
    select {
        case resMsg := <-resCh.(chan ResponseMessage):
            // Successfully retrieve data from the channel
            // Clear the corresponding request dictionary entry to free up space
            close(resCh.(chan ResponseMessage))
            s.resMsgs.Delete(resMsg.ReqID)
            return resMsg.Status, resMsg.Body, nil
        case <-time.After(time.Duration(ResponseTimeout) * time.Second):
            // Timeout occurred without retrieving data from the channel
            close(resCh.(chan ResponseMessage))
            s.resMsgs.Delete(rid)
            return http.StatusInternalServerError, "", fmt.Errorf("timeout: no data received from %s within %v seconds", addr, ResponseTimeout)
        }
}
