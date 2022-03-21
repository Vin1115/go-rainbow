package core

//finished
import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

//格式化metric map结构成字符串
func metricFormat(data MapData) string {
	body := ""
	for k, v := range data {
		body += fmt.Sprintf("%s %v\n", k, v)
	}
	return body
}

//SetMetric
func (r *Rainbow) SetMetric(key string, val interface{}) {
	r.metrics.Store(key, val)
}

//PushGateway 上传metric到pushGateway
func (r *Rainbow) PushGateway(job string, data MapData) (string, error) {
	client := &http.Client{
		Timeout: time.Millisecond * time.Duration(5000),
	}

	url := fmt.Sprintf("http://%s/metrics/job/%s/instance/%s", r.cfg.Service.PushGatewayAddress, job, job)
	req, err := http.NewRequest("POST", url, strings.NewReader(metricFormat(data)))
	if err != nil {
		return "", err
	}

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body2, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body2), nil
}
