package core

//finished
import (
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

//服务熔断器，second/quantity表示XX接口second秒
//内最多允许quantity次错误，超出后熔断

type fusingData struct {
	StartTimestamp int64 //开始时间戳
	Quantity       int64 //最大允许错误次数
}

//分析route配置中熔断的数据，并返回路由配置中fusing的两个值：second/quantity
func (r *Rainbow) fusingAnalyze(limiter string) (int64, int64, error) {
	arr := strings.Split(limiter, "/")
	if len(arr) != 2 {
		return 0, 0, errors.New("route fusing format error")
	}
	second, err := strconv.Atoi(arr[0])
	if err != nil {
		return 0, 0, errors.New("route fusing format error")
	}
	quantity, err := strconv.Atoi(arr[1])
	if err != nil {
		return 0, 0, errors.New("route fusing format error")
	}
	return int64(second), int64(quantity), nil
}

//熔断监视，如果限定时间内错误数超过最大允许数则熔断
func (r *Rainbow) fusingInspect(path string, second, quantity int64) bool {
	f, ok := r.fusingMap.Load(path) //sync.map实现
	if !ok {
		f = r.resetFusingIndex(path) //不存在则初始化
	}
	fd := f.(*fusingData)

	//若大于设置时间，则熔断重设
	now := time.Now().Unix()
	lost := now - fd.StartTimestamp
	if lost >= second { //超时重置,重新计算
		fd = r.resetFusingIndex(path)
	}

	q := atomic.LoadInt64(&fd.Quantity)
	if q >= quantity {
		return false
	}

	return true
}

//到时重置或不存在时初始化设置
func (r *Rainbow) resetFusingIndex(index string) *fusingData {
	fd := fusingData{
		StartTimestamp: time.Now().Unix(),
		Quantity:       0,
	}
	r.fusingMap.Store(index, &fd)
	return &fd
}

//当前路由接口限定时间内发生的错误数+1
func (r *Rainbow) addFusingQuantity(index string) {
	f, ok := r.fusingMap.Load(index)
	if !ok {
		f = r.resetFusingIndex(index)
	}
	fd := f.(*fusingData)

	atomic.AddInt64(&fd.Quantity, 1)
}
