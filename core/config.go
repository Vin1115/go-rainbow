package core

//finished
import (
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"path/filepath"
	"strings"
	"fmt"
)

type routeCfg struct {
	Type    string
	Path    string
	Limiter string
	Fusing  string
	Timeout int
}

type serviceCfg struct {
	Debug              bool
	ServiceName        string
	ServiceIp          string
	HttpOut            bool
	HttpPort           string
	AllowCors          bool
	RpcOut             bool
	RpcPort            string
	CallKey            string
	CallRetry          string
	EtcdKey            string
	EtcdAddress        []string
	TracerDrive        string
	ZipkinAddress      string
	JaegerAddress      string
	PushGatewayAddress string
}

type cfg struct {
	Service     serviceCfg
	Routes      map[string]map[string]routeCfg
	Config      map[string]interface{} //config.yml配置文件中的自定义配置
	RuntimePath string
	ConfigsPath string
}

//获取该rainbow微服务对象的配置
func (r *Rainbow) GetCfg() cfg {
	return r.cfg
}

//初始化引导config并解析与监控
func (r *Rainbow) bootConfig(fileType string) {
	viper.AddConfigPath(r.cfg.ConfigsPath)
	viper.SetConfigType(fileType) //设置配置文件类型

	viper.SetConfigName("config")
	if err := viper.ReadInConfig(); err != nil {
		r.Log(FatalLevel, "config", err)
	}

	viper.SetConfigName("routes")
	if err := viper.MergeInConfig(); err != nil {
		r.Log(FatalLevel, "Config", err)
	}

	r.unmarshalConfig()

	//test
// 	r.Log(InfoLevel,"testconfig",r.cfg)
	fmt.Println(r.cfg)
	//监视路由配置文件routes.yml的改动，改动则做出动作sendRoutes
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		filename := filepath.Base(e.Name) //返回路径最后的文件名
		if strings.Compare(filename, "routes.yml") == 0 {
			r.unmarshalConfig()
			r.sendRoutes()
		}
	})
}

//将viper读取的配置解析到rainbow对象的cfg结构体处
func (r *Rainbow) unmarshalConfig() {
	if err := viper.Unmarshal(&r.cfg); err != nil {
		r.Log(ErrorLevel, "Config", err)
	}
}

//获取map类型配置：GetConfigValueMap("e")
//获取int类型配置：GetConfigValueInt("b")
//获取float32类型配置：GetConfigValueFloat32("a")
//获取float64类型配置：GetConfigValueFloat64("a")
//获取string类型配置：GetConfigValueString("c")
//获取bool类型配置：GetConfigValueString("d")
//获取interface类型配置：GetConfigValueInterface("a").(float64)
//获取rainbow对象中config（map类型）的值
//根据不同返回类型，设置不同的get函数
func (r *Rainbow) GetConfigValueInterface(key string) interface{} {
	config := r.cfg.Config
	if val, ok := config[strings.ToLower(key)]; ok {
		return val
	}
	return nil
}

func (r *Rainbow) GetConfigValueMap(key string) map[string]interface{} {
	config := r.cfg.Config
	if val, ok := config[strings.ToLower(key)]; ok {
		return val.(map[string]interface{})
	}
	return nil
}

func (r *Rainbow) GetConfigValueString(key string) string {
	config := r.cfg.Config
	if val, ok := config[strings.ToLower(key)]; ok {
		return val.(string)
	}
	return ""
}

func (r *Rainbow) GetConfigValueInt(key string) int {
	config := r.cfg.Config
	if val, ok := config[strings.ToLower(key)]; ok {
		return val.(int)
	}
	return 0
}

func (r *Rainbow) GetConfigValueFloat32(key string) float32 {
	config := r.cfg.Config
	if val, ok := config[strings.ToLower(key)]; ok {
		return val.(float32)
	}
	return 0
}

func (r *Rainbow) GetConfigValueFloat64(key string) float64 {
	config := r.cfg.Config
	if val, ok := config[strings.ToLower(key)]; ok {
		return val.(float64)
	}
	return 0
}

func (r *Rainbow) GetConfigValueBool(key string) bool {
	config := r.cfg.Config
	if val, ok := config[strings.ToLower(key)]; ok {
		return val.(bool)
	}
	return false
}
