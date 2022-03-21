package core

//finished
import (
	"go.uber.org/atomic"
	"net/http"
	"sync"
)

type (
	logLevel int8

	//MapData 任何类似map的数据类型
	MapData map[string]interface{}

	//Rainbow go-rainbow框架类
	Rainbow struct {
		container      sync.Map
		cfg            cfg
		logBoot        uint
		serviceType    uint //0:service 1:gateway
		services       map[string]*service
		serviceManager chan serviceOperate
		syncCache      []byte //同步路由配置的缓存
		fusingMap      sync.Map
		limiterMap     sync.Map

		metrics sync.Map

		// /metrics接口提供采集指标，默认实现了RequestProcess
		//和RequestFinish两个指标，分别表示处理中请求数和已完成
		//请求数，可通过表达式查询；
		requestProcess atomic.Int64
		requestFinish  atomic.Int64
	}
)

//日志等级
const (
	DebugLevel logLevel = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
	DPanicLevel
	PanicLevel
	FatalLevel
)

const (
	httpOk       = http.StatusOK
	httpFail     = http.StatusInternalServerError
	httpNotFound = http.StatusNotFound

	infoSuccess       = "Success"
	infoServerError   = "Server Error"
	infoServerLimiter = "Server limit flow"
	infoServerFusing  = "Server fusing flow"
	infoNoAuth        = "No access permission"
	infoNotFound      = "The resource could not be found"
	infoTimeout       = "Request timeout"
)
