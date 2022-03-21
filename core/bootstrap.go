/*
服务启动引导相关代码
*/
//finished
package core

import (
	"go-rainbow/drives/db"
	"go-rainbow/drives/etcd"
	"go-rainbow/drives/redis"
     //"gitee.com/vinzhang1115/go-rainbow"
)

func (r *Rainbow) bootstrap(configPath, runtimePath string) {
	r.cfg.ConfigsPath = configPath
	r.cfg.RuntimePath = runtimePath
	r.bootConfig("yml")
	r.checkConfig()
	r.bootLog()
	r.Log(InfoLevel, "bootstrap", r.cfg.Service.ServiceName+" running")
	r.bootEtcd()
	r.bootService()
	r.bootOpenTracing()
	r.bootDb()
	r.bootRedis()
}

//引导自定义etcd驱动程序，初始化etcd客户端，并装入容器中
func (r *Rainbow) bootEtcd() {
	etcdClient, err := etcd.Connect(r.cfg.Service.EtcdAddress)
	if err != nil {
		r.Log(FatalLevel, "bootetcd", err)
	}
	r.setSafe("etcd", etcdClient)
}

//引导自定义db驱动程序，初始化db客户端，并装入容器中
func (r *Rainbow) bootDb() {
	dbconf := r.GetConfigValueMap("db")
	if dbconf != nil {
		dbC, err := db.Connect(dbconf)
		if err != nil {
			r.Log(FatalLevel, "bootdb", err)
		}
		r.Log(InfoLevel, "db", "Connect successfully")
		r.setSafe("db", dbC)
	}
}

//引导自定义redis驱动程序，初始化redis客户端，并装入容器中
func (r *Rainbow) bootRedis() {
	redisConf := r.GetConfigValueMap("redis")
	if redisConf != nil {
		redisC, err := redis.Connect(redisConf, func(err interface{}) {
			r.Log(FatalLevel, "redis", err)
		})
		if err != nil {
			r.Log(FatalLevel, "database", err)
		}
		r.Log(InfoLevel, "redis", "Connect successfully")
		r.setSafe("redis", redisC)
	}
}

//检查service设置是否正确
func (r *Rainbow) checkConfig() {
	if r.cfg.Service.ServiceName == "" {
		r.Log(FatalLevel, "Config", "empty option serviceName")
	}
	if r.cfg.Service.HttpPort == "" {
		r.Log(FatalLevel, "Config", "empty option httpPort")
	}
	if r.cfg.Service.RpcPort == "" {
		r.Log(FatalLevel, "Config", "empty option httpPort")
	}
	if r.cfg.Service.CallKey == "" {
		r.Log(FatalLevel, "Config", "empty option callKey")
	}
	if r.cfg.Service.CallRetry == "" {
		r.Log(FatalLevel, "Config", "empty option callRetry")
	}
	if r.cfg.Service.EtcdKey == "" {
		r.Log(FatalLevel, "Config", "empty option etcdKey")
	}
	if len(r.cfg.Service.EtcdAddress) == 0 {
		r.Log(FatalLevel, "Config", "empty option etcdAddress")
	}
	if r.cfg.Service.TracerDrive != "zipkin" && r.cfg.Service.TracerDrive != "jaeger" {
		r.Log(FatalLevel, "Config", "traceDrive just support zipkin or jaeger")
	}
	if r.cfg.Service.TracerDrive == "zipkin" && r.cfg.Service.ZipkinAddress == "" {
		r.Log(FatalLevel, "Config", "empty option zipkinAddress")
	}
	if r.cfg.Service.TracerDrive == "jaeger" && r.cfg.Service.JaegerAddress == "" {
		r.Log(FatalLevel, "Config", "empty option jaegerAddress")
	}
}
