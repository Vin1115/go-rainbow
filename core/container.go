package core

//finished
import (
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	clientV3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

//通过名字设置实例，不支持设置默认名如‘log’，‘db’，‘redis’，‘etcd’
//此4类容器框架已默认设置好，故此数据结构为禁止添加的实例列表
var unSafeList = map[string]interface{}{
	"log":   nil,
	"db":    nil,
	"redis": nil,
	"etcd":  nil,
}

//设置框架内置容器实例，即默认4类
func (r *Rainbow) setSafe(name string, val interface{}) {
	r.container.Store(name, val)
}

//Get 由名字获得容器中实例
func (r *Rainbow) Get(name string) (interface{}, error) {
	if res, ok := r.container.Load(name); ok {
		return res, nil
	}
	return nil, errors.New(fmt.Sprintf("Not found %s from container! ", name))
}

//Set (自定义添加容器实例)通过名字设置实例，不允许设置默认4类容器名字：log’，‘db’，‘redis’，‘etcd’
func (r *Rainbow) Set(name string, val interface{}) error {
	if _, ok := unSafeList[name]; ok {
		return errors.New("Cant's set unsafe name! ")
	}
	r.container.Store(name, val)
	return nil
}

//GetLog instance to write custom Logs
func (r *Rainbow) GetLog() *zap.SugaredLogger {
	res, _ := r.Get("log")
	return res.(*zap.SugaredLogger)
}

//GetDb instance to performing database operations
func (r *Rainbow) GetDb() *gorm.DB {
	res, _ := r.Get("db")
	return res.(*gorm.DB)
}

//GetRedis instance to performing redis operations
func (r *Rainbow) GetRedis() *redis.Client {
	res, _ := r.Get("redis")
	return res.(*redis.Client)
}

//GetEtcd instance to performing etcd operations
func (r *Rainbow) GetEtcd() *clientV3.Client {
	res, _ := r.Get("etcd")
	return res.(*clientV3.Client)
}
