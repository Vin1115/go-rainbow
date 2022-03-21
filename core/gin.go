package core

//finished
import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

//启动一个gin客户端并监听
func (r *Rainbow) ginListen(listenAddress string, route func(e *gin.Engine), auth func() gin.HandlerFunc) error {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	//如果服务设置了debug选项，使用gin自带log中间件生成gin框架的日志文件
	if r.cfg.Service.Debug {
		if err := createDir(r.cfg.RuntimePath); err != nil {
			return err
		}
		file, err := os.Create(fmt.Sprintf("%s/gin.log", r.cfg.RuntimePath))
		if err != nil {
			return err
		}
		gin.DefaultWriter = file
		engine.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
				param.ClientIP,
				param.TimeStamp.Format(time.RFC1123),
				param.Method,
				param.Path,
				param.Request.Proto,
				param.StatusCode,
				param.Latency,
				param.Request.UserAgent(),
				param.ErrorMessage)
		}))
		//集成pprof性能监控，路由：/debug/pprof（需要开启debug调试模式，因为安全问题不建议生产环境开启）
		pprof.Register(engine)
	} else {
		gin.DefaultWriter = ioutil.Discard
		//Discard is an io.Writer on which all Write calls succeed without doing anything.
	}

	engine.Use(r.openTracingMiddleware()) //调用了c.next()
	if r.cfg.Service.AllowCors {
		engine.Use(cors)
	}
	r.prometheus(engine) //gin服务监控路由
	if auth != nil {
		engine.Use(auth())
	}
	notFound(engine) //The resource could not be found 404时的处理情况
	route(engine)

	r.Log(InfoLevel, "http", fmt.Sprintf("listen on: %s", listenAddress))
	return engine.Run(listenAddress)
}

// GatewayRoute create gateway service type to use this
func (r *Rainbow) GatewayRoute(e *gin.Engine) {
	r.serviceType = 1
	e.Any("api/:service/:action", func(c *gin.Context) {
		r.gateway(c)
	})
}

//404 notfound：error The resource could not be found
func notFound(e *gin.Engine) {
	e.NoRoute(func(c *gin.Context) {
		c.JSON(httpNotFound, gatewayFail(infoNotFound)) //gateway文件统一处理gateway函数
	})
	e.NoMethod(func(c *gin.Context) {
		c.JSON(httpNotFound, gatewayFail(infoNotFound))
	})
}

//跨域设置
func cors(ctx *gin.Context) {
	method := ctx.Request.Method
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Access-Control-Allow-Headers", "*")
	ctx.Header("Access-Control-Allow-Methods", "*")
	ctx.Header("Access-Control-Expose-Headers", "*")
	ctx.Header("Access-Control-Allow-Credentials", "true")
	if method == "OPTIONS" {
		ctx.AbortWithStatus(http.StatusNoContent)
	}
}

//gin监控路由
func (r *Rainbow) prometheus(e *gin.Engine) {
	e.GET("/metrics", func(c *gin.Context) {
		data := MapData{
			"RequestProcess": r.requestProcess.String(),
			"RequestFinish":  r.requestFinish.String(),
		}
		r.metrics.Range(func(k, v interface{}) bool {
			data[k.(string)] = v
			return true
		})
		c.String(200, metricFormat(data))
	})
}

func (r *Rainbow) openTracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		r.requestProcess.Inc() // 处理中请求数+1

		span := StartSpanFromHeader(c.Request.Header, c.Request.RequestURI)
		span.SetTag("CallType", "Http")
		span.SetTag("ServiceIp", r.GetServiceIp())
		span.SetTag("ServiceId", r.GetServiceId())
		span.SetTag("Status", "unfinished")

		request := req{
			getMethod(c),
			getUrl(c),
			getUrlParam(c),
			getClientIp(c),
			getHeaders(c),
			getBody(c)}
		s, _ := json.Marshal(&request)
		span.SetTag("Request", string(s))

		c.Set("span", span)
		c.Set("request", &request)

		c.Next() //调用后续的中间件处理函数

		span.SetTag("Status", "finished")
		span.Finish()

		r.requestProcess.Dec()
		r.requestFinish.Inc()
	}
}

func getMethod(c *gin.Context) string {
	return strings.ToUpper(c.Request.Method)
}

func getUrl(c *gin.Context) string {
	return c.Request.URL.Path
}

func getUrlParam(c *gin.Context) string {
	requestUrl := c.Request.RequestURI
	urlSplit := strings.Split(requestUrl, "?")
	if len(urlSplit) > 1 {
		requestUrl = "?" + urlSplit[1]
	} else {
		requestUrl = ""
	}
	return requestUrl
}

func getClientIp(c *gin.Context) string {
	return c.ClientIP()
}

func getHeaders(c *gin.Context) MapData {
	headers := MapData{}
	for k, v := range c.Request.Header {
		headers[k] = v[0]
	}
	return headers
}

func getBody(c *gin.Context) MapData {
	body := MapData{}
	h := c.GetHeader("Content-Type")
	// 获取表单格式请求参数
	if strings.Contains(h, "multipart/form-data") || strings.Contains(h, "application/x-www-form-urlencoded") {
		c.PostForm("get_params_bug_fix")
		for k, v := range c.Request.PostForm {
			body[k] = v[0]
		}
	} else if strings.Contains(h, "application/json") {
		c.BindJSON(&body)
	}
	return body
}

//GetContext get custom context
func GetContext(c *gin.Context, name string) (interface{}, error) {
	t, success := c.Get(name)
	if !success {
		return nil, errors.New(name + " is nil")
	}
	return t, nil
}

//SetContext store a new key/value pair exclusively for this context.
func SetContext(c *gin.Context, name string, val interface{}) {
	c.Set(name, val)
}

//GetRequest get request datatype from context
func GetRequest(c *gin.Context) *req {
	t, _ := GetContext(c, "request")
	r := t.(*req)
	return r
}

// GetSpan get opentracing span from context
func GetSpan(c *gin.Context) opentracing.Span {
	t, _ := GetContext(c, "span")
	r := t.(opentracing.Span)
	return r
}

// CheckCallSafeMiddleware from call service safe check
// 微服务开发代码中使用
func (r *Rainbow) CheckCallSafeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !r.checkCallSafe(c.GetHeader("Call-Key")) {
			c.JSON(httpNotFound, gatewayFail(infoNoAuth))
			c.Abort()
		}
	}
}

//检查callkey是否安全
func (r *Rainbow) checkCallSafe(key string) bool { //原security文件
	if strings.Compare(key, r.cfg.Service.CallKey) != 0 {
		return false
	}
	return true
}
