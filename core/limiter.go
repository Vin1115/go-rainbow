package core

//finished
import (
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

//routes.yml字段limiter值：服务限流器，如5/100表示login接口5秒内最多接受100个请求，超出后限流
type limiterData struct {
	StartTimestamp int64
	Quantity       int64
}

//返回limiter字段的两个int64的值：second 和 quantity
func limiterAnalyze(limiter string) (int64, int64, error) {
	arr := strings.Split(limiter, "/")
	if len(arr) != 2 {
		return 0, 0, errors.New("route limiter format error")
	}
	second, err := strconv.Atoi(arr[0])
	if err != nil {
		return 0, 0, errors.New("route limiter format error")
	}
	quantity, err := strconv.Atoi(arr[1])
	if err != nil {
		return 0, 0, errors.New("route limiter format error")
	}
	return int64(second), int64(quantity), nil
}

//服务限流器监视
func (r *Rainbow) limiterInspect(path string, second, quantity int64) bool {
	l, ok := r.limiterMap.Load(path)
	if !ok {
		l = r.resetLimiterIndex(path)
	}
	ld := l.(*limiterData)

	now := time.Now().Unix()
	lost := now - ld.StartTimestamp
	if lost >= second { //超过限定时间则重新计时
		ld = r.resetLimiterIndex(path)
	}

	if atomic.LoadInt64(&ld.Quantity) >= quantity { //判断是否已超过限流值
		return false
	}

	atomic.AddInt64(&ld.Quantity, 1) //原子操作确保安全性，增加一次该服务访问次数

	return true
}

//到时重置或不存在时初始化设置
func (r *Rainbow) resetLimiterIndex(index string) *limiterData {
	ld := limiterData{
		StartTimestamp: time.Now().Unix(),
		Quantity:       0,
	}
	r.limiterMap.Store(index, &ld)
	return &ld
}
