package common

import (
	"github.com/gin-gonic/gin"
)

type Endpoint struct {
	Method  string
	Path    string
	Handler func(c *gin.Context) (ResponseType, int, any)
}

type HttpServer struct {
	*IP
	HttpLogger *Logger
	endpoints  []Endpoint
}

func InitHttpServer(logger *Logger, endpoints []Endpoint) *HttpServer {
	return &HttpServer{
		IP:         nil,
		HttpLogger: GetLogger("http server", logger),
		endpoints:  endpoints,
	}
}

// todo what if machine is not online ????
func (host *HttpServer) RunHttpServer(ip IP) {
	if host.endpoints == nil {
		host.HttpLogger.Error("endpoints not set")
		return
	}

	host.IP = &ip
	router := gin.Default()

	addLogging := func(fnToCall func(c *gin.Context) (ResponseType, int, any)) func(c *gin.Context) {
		return func(c *gin.Context) {
			host.HttpLogger.Info("%s -->   %-6s   %s", c.RemoteIP(), c.Request.Method, c.Request.URL.String())
			responseType, code, body := fnToCall(c)
			switch responseType {
			case StringResponse:
				c.String(code, body.(string))
			case JSONResponse:
				c.JSON(code, body)
			case DataResponse:
				c.Header("Content-Type", string(DataResponse))
				c.Data(code, "application/octet-stream", body.([]byte))
			case NoResponse:
				c.Status(code)
			case ErrorResponse:

				switch body.(type) {
				case error:
					host.HttpLogger.Err(body.(error))
					c.JSON(code, gin.H{"error": body})
				case string:
					host.HttpLogger.Error(body.(string))
					c.JSON(code, gin.H{"error": body})

				case []error:
					for _, err := range body.([]error) {
						host.HttpLogger.Err(err)
					}
					c.JSON(code, gin.H{"errors": body})
				default:
					panic("unknown response type")
				}

			}

		}
	}

	for _, endpoint := range host.endpoints {
		switch endpoint.Method {
		case "POST":
			router.POST(endpoint.Path, addLogging(endpoint.Handler))
		case "GET":
			router.GET(endpoint.Path, addLogging(endpoint.Handler))
		}
	}

	err := router.Run(host.IPv4.String() + ":" + host.Port)
	if err != nil {
		host.HttpLogger.Err(err)
	}

}
