package core

//finished
import "github.com/gin-gonic/gin"

//Run 启动服务
func (r *Rainbow) Run(route func(e *gin.Engine), rpc interface{}, auth func() gin.HandlerFunc) {
	go r.runHttpServer(route, auth)
	go r.runRpcServer(rpc)
	forever := make(chan int, 0)
	<-forever //只有读端的无缓存的channel，会一直阻塞
}

func (r *Rainbow) runHttpServer(route func(e *gin.Engine), auth func() gin.HandlerFunc) {
	address := r.GetServiceIp()
	if r.cfg.Service.HttpOut { //http端口是否允许外网访问
		address = "0.0.0.0"
	}
	listenAddress := address + ":" + r.cfg.Service.HttpPort
	if err := r.ginListen(listenAddress, route, auth); err != nil { //在listen函数处run起server engine，并配置好gin日志服务和中间件
		r.Log(FatalLevel, "ginRun", err)
	}
}

func (r *Rainbow) runRpcServer(rpc interface{}) {
	address := r.GetServiceIp()
	if r.cfg.Service.RpcOut { //rpc端口是否允许外部访问
		address = "0.0.0.0"
	}
	rpcAddress := address + ":" + r.cfg.Service.RpcPort
	if err := r.rpcListen(r.cfg.Service.ServiceName, "tcp", rpcAddress, rpc, ""); err != nil {
		r.Log(FatalLevel, "rpcRun", err)
	}
}

//RebootFunc 当panic时自动reboot
func (r *Rainbow) RebootFunc(label string, f func()) {
	defer func() {
		if err := recover(); err != nil {
			r.Log(ErrorLevel, label, err)
			f()
		}
	}()
	f()
}
