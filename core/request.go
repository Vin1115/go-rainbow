package core

//finished
import (
	"errors"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Request datatype
type req struct {
	Method   string  `json:"method"`
	Url      string  `json:"url"`
	UrlParam string  `json:"urlParam"`
	ClientIp string  `json:"clientIp"`
	Headers  MapData `json:"headers"`
	Body     MapData `json:"body"`
}

// CallRpc call other service rpc method
// 微服务开发接口
func (r *Rainbow) CallRpc(span opentracing.Span, service, action string, args, reply interface{}) error {
	_, _, _, err := r.callService(span, service, action, nil, &args, &reply)
	if err != nil {
		return err
	}
	return nil
}

func (r *Rainbow) callService(span opentracing.Span, service, action string, request *req, args, reply interface{}) (int, string, http.Header, error) {
	s := r.cfg.Routes[service] //路由设置中的服务
	if len(s) == 0 {
		return httpNotFound, infoNotFound, nil, errors.New("service not found")
	}

	route := s[action] //路由设置中的服务的路由接口，如exists等
	if (route.Type != "http" && route.Type != "rpc") ||
		(route.Type == "http" && len(route.Path) == 0) ||
		(route.Type == "rpc" && (args == nil || reply == nil)) {
		return httpNotFound, infoNotFound, nil, errors.New("service route not found")
	}

	serviceAddr, nodeIndex, err := r.selectService(service)
	if err != nil {
		return httpNotFound, infoNotFound, nil, err
	}

	// service limiter 服务限流
	if route.Limiter != "" {
		second, quantity, err := limiterAnalyze(route.Limiter) //获取路由配置中限流的两个配置参数，such as 5/10000
		if err != nil {
			r.Log(DebugLevel, "Limiter", err)
		} else if !r.limiterInspect(serviceAddr+"/"+service+"/"+action, second, quantity) { //监视服务限流情况，并加一次服务访问次数
			//超出最大连接限制数量则返回错误
			span.SetTag("break", "service limiter")
			return httpNotFound, infoServerLimiter, nil, errors.New("server limiter")
		}
	}

	// service fusing 服务熔断
	if route.Fusing != "" {
		second, quantity, err := r.fusingAnalyze(route.Fusing) //获取路由配置中熔断的两个配置参数，such as 5/100
		if err != nil {
			r.Log(ErrorLevel, "Fusing", err)
		} else if !r.fusingInspect(serviceAddr+"/"+service+"/"+action, second, quantity) {
			span.SetTag("break", "service fusing")
			return httpNotFound, infoServerFusing, nil, errors.New("server fusing")
		}
	}

	// service call retry：服务重试策略，格式timer1/timer2/timer3/...（单位毫秒）
	retry, err := retryAnalyze(r.cfg.Service.CallRetry)
	if err != nil {
		r.Log(DebugLevel, "Retry", err)
		retry = []int{0}
	}

	//通过retryGo调用服务，只有当requestServiceHttp发生错误时再遍历retry数组执行等待时间并重试，使用了if语句判断，没有错误则直接break
	code, result, header, err := r.retryGo(service, action, retry, nodeIndex, span, route, request, args, reply)

	return code, result, header, err
}

func (r *Rainbow) requestServiceHttp(span opentracing.Span, url string, request *req, timeout int) (int, string, http.Header, error) {
	client := &http.Client{
		Timeout: time.Millisecond * time.Duration(timeout),
	}

	// encapsulation request body
	var s string
	for k, v := range request.Body {
		s += fmt.Sprintf("%v=%v&", k, v)
	}
	s = strings.Trim(s, "&")
	req, err := http.NewRequest(request.Method, url, strings.NewReader(s))
	if err != nil {
		return httpFail, "", nil, err
	}

	// New request request
	for k, v := range request.Headers {
		req.Header.Add(k, v.(string))
	}
	// Add the body format header
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Increase calls to the downstream service security validation key
	req.Header.Set("Call-Key", r.cfg.Service.CallKey)

	// add request opentracing span header
	opentracing.GlobalTracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header))

	res, err := client.Do(req)
	if err != nil {
		return httpFail, "", nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != httpOk {
		return res.StatusCode, "", nil, errors.New("http status " + strconv.Itoa(res.StatusCode))
	}
	body2, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return httpFail, "", nil, err
	}
	return httpOk, string(body2), res.Header, nil //返回response
}
