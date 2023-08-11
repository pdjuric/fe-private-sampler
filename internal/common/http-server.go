package common

import (
	"github.com/gin-gonic/gin"
)

type Endpoint struct {
	Method   string
	Path     string
	Function func(c *gin.Context)
}

// todo what if machine is not online ????
type HttpServer struct {
	IP
	HttpLogger *Logger
	endpoints  []Endpoint
}

func InitHttpServer(logFilename string, ip IP, endpoints []Endpoint) *HttpServer {
	return &HttpServer{
		IP:         ip,
		HttpLogger: GetLoggerForFile("http server", logFilename),
		endpoints:  endpoints,
	}
}

// RunHttpServer ; prior to call of this function, SetEndpoints must be called
func (host *HttpServer) RunHttpServer() {
	if host.endpoints == nil {
		host.HttpLogger.Error("endpoints not set")
	}

	router := gin.Default()

	addLogging := func(fnToCall func(c *gin.Context)) func(c *gin.Context) {
		return func(c *gin.Context) {
			host.HttpLogger.Info("%s -->   %-6s   %s", c.RemoteIP(), c.Request.Method, c.Request.URL.String())
			fnToCall(c)
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

	//todo should here be localhost?
	err := router.Run(host.IPv4.String() + ":" + host.Port)
	if err != nil {
		host.HttpLogger.Err(err)
	}

}
