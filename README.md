# Full Duplex Proxy (fdx-proxy)
`fdx-proxy` is a Go library that provides a full-duplex proxy solution to handle scenarios where an HTTP node is not directly accessible by external services but can initiate outbound connections. The library establishes a WebSocket-based full-duplex communication channel between a proxy (`fdx-proxy` client) running on the HTTP node and an external service (`fdx-proxy` server). This allows seamless proxying of requests from the external service to the HTTP node through the established WebSocket connection.

## Components
The `fdx-proxy` library consists of the following core components:
- **FullDuplexServer**: Manages WebSocket connections and forwards HTTP requests from external clients to the target HTTP service.
- **FullDuplexProxy**: Acts as a client that establishes a full-duplex WebSocket channel with the `FullDuplexServer`, enabling bidirectional communication.
- **Proxy Controller (`tools/main.go`)**: Provides a command-line interface to start the `fdx-proxy` client.

## Installation
### Using the Library in Your Go Project
To use `fdx-proxy` as a Go library in your own project, add it as a dependency:

```bash
go get github.com/yuuavwa/fdx-proxy
```

## Usage
### Integrating `fdx-proxy` with Your Service
The primary use case for `fdx-proxy` is to integrate it with an external service that requires access to an HTTP node which is not directly accessible. Here’s how to use `fdx-proxy`:

1. **Implement the Full Duplex Server in Your External Service:**
   In your external service, use `fdx-proxy` to implement a full-duplex server that listens for WebSocket connections from the proxy client. Here is an example:

   ```go
   import (
       "fmt"
       proxy "github.com/yuuavwa/fdx-proxy"
   )

   var serverCtrl *proxy.FullDuplexServerController
   serverCtrl, _ = proxy.NewFullDuplexServerController()

   // in your gin router handler
   func ProxyHandler(c *gin.Context) {
       targetID := c.Param("target_id")
       err := serverCtrl.AddFullDuplexConnController(c, targetID)
       if err != nil {
           panic(err)
       }
       // ...
   }

   func MyTask(target_id string) {
       if ctrl, err := serverCtrl.GetFullDuplexConnController(target_id); ctrl != nil && err == nil {
           ReqMethod := "GET"
           ReqHeaders := map[string]string{}
           ReqBody := ""
           ReqURL := "https://www.this-is-just-a-test.com/"
           // here to use CallAPI to proxy the request through the websocket connection
           status, res_body, err := serverCtrl.CallAPI(target_id, ReqMethod, ReqURL, ReqHeaders, ReqBody)
           if err != nil {
               panic(err)
           }
           fmt.Println(status, res_body)
           // ...
       }
   }

   func main() {
       router := gin.Default()
       router.GET("/api/EstablishFullDuplexChannel/:target_id", ProxyHandler)
       go MyTask("192.168.0.100:5000")
       router.Run(":8080")  // Start server on port 8080
   }
   ```

   This example sets up a server that listens for WebSocket connections at the endpoint `/api/EstablishFullDuplexChannel/:target_id` and handles incoming requests using the `CallAPI` function.

2. **Deploy the Full Duplex Proxy on the HTTP Node:**
   On the HTTP node that cannot be accessed directly but can access the external service, deploy the proxy client by running the following command:
   ```bash
   go run tools/main.go -m proxy -s <external_service_address> -t <target_address>
   ```
   - `<external_service_address>`: The address of the external service that runs the `fdx-proxy` server.
   - `<target_address>`: The identifier or address of the target HTTP service that the proxy client will forward requests to.
   This command starts the `fdx-proxy` client, which connects to the external service and establishes a WebSocket channel for full-duplex communication.

## How It Works
1. **fdx-server:**
   - Listens for WebSocket connections at the specified endpoint.
   - Creates a `FullDuplexServerController` to manage incoming and outgoing requests over the WebSocket channel.

2. **fdx-proxy:**
   - Actively connects to the proxy server using WebSocket and establishes a communication channel.
   - Uses `FullDuplexProxyController` to handle bidirectional communication, including reading requests from the WebSocket and forwarding them to the target HTTP service.

## Logging
The `fdx-proxy` library supports logging. Modify the `SetLogger` function in `util.go` to set the desired log file path.
