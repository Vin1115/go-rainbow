package core

//finished
import (
	"io/ioutil"
	"net"
	"os"
)

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

//从文件路径读取文件，返回字节流数组
func readFile(filepath string) ([]byte, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func createDir(path string) error {
	exists, err := pathExists(path)
	if err != nil {
		return err
	}
	if !exists {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeFile(filepath string, data []byte) error {
	if err := ioutil.WriteFile(filepath, data, 0777); err != nil {
		return nil
	}
	return nil
}

//通过dns服务器8.8.8.8:80获取外围本地IP
func getOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}

//使首字母大写
func capitalize(str string) string {
	var upperStr string
	vv := []rune(str)
	for i := 0; i < len(vv); i++ {
		if i == 0 {
			if vv[i] >= 97 && vv[i] <= 122 {
				vv[i] -= 32
				upperStr += string(vv[i])
			} else {
				return str
			}
		} else {
			upperStr += string(vv[i])
		}
	}
	return upperStr
}
