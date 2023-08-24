package common

import (
	"github.com/gin-gonic/gin"
)

type Endpoint struct {
	Method   string
	Path     string
	Function func(c *gin.Context) (ResponseType, int, any)
}

// todo what if machine is not online ????
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

// RunHttpServer ; prior to call of this function, SetEndpoints must be called
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
			router.POST(endpoint.Path, addLogging(endpoint.Function))
		case "GET":
			router.GET(endpoint.Path, addLogging(endpoint.Function))
		}
	}

	err := router.Run(host.IPv4.String() + ":" + host.Port)
	if err != nil {
		host.HttpLogger.Err(err)
	}

}
