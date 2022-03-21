package core

//finished
import (
	"errors"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

//call retry：服务重试策略，格式timer1/timer2/timer3/...（单位毫秒）
//retry数据分析
func retryAnalyze(retry string) ([]int, error) {
	retrySlice := make([]int, 0)
	arr := strings.Split(retry, "/")
	if len(arr) == 0 {
		return []int{}, errors.New("config retry format error")
	}
	for _, sec := range arr {
		s, err := strconv.Atoi(sec)
		if err != nil {
			return []int{}, errors.New("config retry format error")
		}
		retrySlice = append(retrySlice, s)
	}

	retrySlice = append(retrySlice, 0) //服务重试时间切片后添0

	return retrySlice, nil
}

func (r *Rainbow) retryGo(service, action string, retry []int, nodeIndex int, span opentracing.Span, route routeCfg, request *req, rpcArgs, rpcReply interface{}) (int, string, http.Header, error) {
	code := httpOk
	result := infoSuccess
	addr := ""
	var err error
	var header http.Header

	for i, rtr := range retry {
		atomic.AddInt64(&r.services[service].Nodes[nodeIndex].Waiting, 1) //等待队列

		if route.Type == "http" {
			addr, err = r.getServiceHttpAddr(service, nodeIndex)
			if err != nil {
				code = httpFail
				break
			}
			addr = "http://" + addr + route.Path                                                 //http路由类型时需要此配置，path：路由
			code, result, header, err = r.requestServiceHttp(span, addr, request, route.Timeout) //如果有err交给后面处理
		} else if route.Type == "rpc" {
			addr, err = r.getServiceRpcAddr(service, nodeIndex)
			if err != nil {
				code = httpFail
				break
			}
			action = capitalize(action) //使首字母大写
			err = rpcCall(span, addr, service, action, rpcArgs, rpcReply, route.Timeout)
			if err != nil {
				code = httpFail
			}
		}

		atomic.AddInt64(&r.services[service].Nodes[nodeIndex].Waiting, -1)

		//requestServiceHttp产生的错误的处理
		if err != nil {
			r.Log(ErrorLevel, "callService", err)
			r.addFusingQuantity(r.services[service].Nodes[nodeIndex].Addr + "/" + service + "/" + action)

			// call timeout don't retry
			if strings.Contains(err.Error(), "Timeout") || strings.Contains(err.Error(), "deadline") {
				err = errors.New(fmt.Sprintf("Call %s %s %s timeout", route.Type, service, action))
				return code, infoTimeout, nil, err
			}

			// call 404 don't retry
			if code == httpNotFound {
				return code, infoNotFound, nil, err
			}

			if i == len(retry)-1 { //retry切片最后面补了个0，当遍历到最后时不用再重试，直接返回错误
				return code, infoServerError, nil, err
			}
			time.Sleep(time.Millisecond * time.Duration(rtr))
			continue
		}

		break
	}

	atomic.AddInt64(&r.services[service].Nodes[nodeIndex].Finish, 1) //完成

	return code, result, header, err
}
