package core

//finished
import "flag"

// new一个rainbow实例,并初始化引导程序启动bootstrap（）
func New() *Rainbow {
	service := Rainbow{}

	//解析命令行参数并设置默认值，eg：go run main.go -config=configs -runtime=runtime
	var configPath string
	var runtimePath string
	flag.StringVar(&configPath, "configs", "configs", "config yml files path")
	flag.StringVar(&runtimePath, "runtime", "runtime", "runtime log files path")
	flag.Parse()

	service.bootstrap(configPath, runtimePath)
	return &service
}
