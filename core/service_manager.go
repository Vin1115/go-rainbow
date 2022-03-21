package core

//finished
import (
	"context"
	"errors"
	"fmt"
	clientV3 "go.etcd.io/etcd/client/v3"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"
)

type service struct {
	Nodes []node
}

type node struct { //每种服务里的不同节点
	Addr    string
	Waiting int64 //todo 该等待值的含义和用处
	Finish  int64
}

//操作类型：如addNode、delNode。要对特定服务的特定节点所要执行的操作
type serviceOperate struct {
	operate     string
	serviceName string
	serviceAddr string
	nodeIndex   int
}

//GetServices get all cluster service map by etcdKey
func (r *Rainbow) GetServices() map[string]*service {
	return r.services
}

//GetServiceIp 获取该服务的ip地址
func (r *Rainbow) GetServiceIp() string {
	return r.cfg.Service.ServiceIp
}

//GetServiceId 获取service union id：etcdkey_servicename_serviceIp_httpport_rpcport
func (r *Rainbow) GetServiceId() string {
	return r.cfg.Service.EtcdKey + "_" + r.cfg.Service.ServiceName + "_" + r.cfg.Service.ServiceIp + ":" + r.cfg.Service.HttpPort + ":" + r.cfg.Service.RpcPort
}

//初始化引导
func (r *Rainbow) bootService() {
	var err error
	r.services = map[string]*service{}

	if r.cfg.Service.ServiceIp == "" {
		r.cfg.Service.ServiceIp, err = getOutboundIP()
		if err != nil {
			r.Log(FatalLevel, "bootService", err)
		}
	}
	r.serviceManager = make(chan serviceOperate, 0)
	go r.RebootFunc("serviceManageWatchReboot", func() {
		r.serviceManageWatch(r.serviceManager)
	})

	if err = r.serviceRegister(); err != nil {
		r.Log(FatalLevel, "bootService", err)
	}
}

//监视服务操作，选择删除还是添加节点
func (r *Rainbow) serviceManageWatch(ch chan serviceOperate) {
	for {
		select {
		case sm := <-ch:
			switch sm.operate {
			case "addNode":
				r.createServiceIndex(sm.serviceName)
				r.services[sm.serviceName].Nodes = append(r.services[sm.serviceName].Nodes, node{Addr: sm.serviceAddr})
				break

			case "delNode":
				if r.existsService(sm.serviceName) {
					for i := 0; i < len(r.services[sm.serviceName].Nodes); i++ {
						if r.services[sm.serviceName].Nodes[i].Addr == sm.serviceAddr {
							r.services[sm.serviceName].Nodes = append(r.services[sm.serviceName].Nodes[:i], r.services[sm.serviceName].Nodes[i+1:]...)
							i--
						}
					}
				}
				break
			}
		}
	}
}

//判断该服务是否存在
func (r *Rainbow) existsService(name string) bool {
	_, ok := r.services[name]
	return ok
}

//如果该服务不存在则创建一个该服务
func (r *Rainbow) createServiceIndex(name string) {
	if !r.existsService(name) {
		r.services[name] = &service{
			Nodes: []node{},
		}
	}
}

//服务注册
func (r *Rainbow) serviceRegister() error {
	client := r.GetEtcd()
	// New lease
	resp, err := client.Grant(context.TODO(), 2)
	if err != nil {
		return err
	}
	// The lease was granted
	if err != nil {
		return err
	}
	//放入强一致性kv数据库etcd中
	_, err = client.Put(context.TODO(), r.GetServiceId(), "0", clientV3.WithLease(resp.ID))
	if err != nil {
		return err
	}
	// keep alive
	ch, err := client.KeepAlive(context.TODO(), resp.ID)
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-ch:

			}
		}
	}()

	services := r.getAllServices()
	for _, service := range services {
		arr := strings.Split(service, "_")
		serviceName := arr[0]
		serviceHttpAddr := arr[1]

		r.addServiceNode(serviceName, serviceHttpAddr)
	}

	go r.RebootFunc("serviceWatcherReboot", r.serviceWatcher)
	go func() {
		for {
			time.Sleep(15 * time.Second)
			r.getAllServices()
		}
	}()

	return nil
}

//获取同一个etcdkey的所有服务部分id：servicename_serviceIp_httpport_rpcport
func (r *Rainbow) getAllServices() []string {
	client := r.GetEtcd()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := client.Get(ctx, r.cfg.Service.EtcdKey+"_", clientV3.WithPrefix()) //取所有具有同一个etcdkey的服务
	cancel()
	if err != nil {
		r.Log(ErrorLevel, "GetAllServices", err)
		return []string{}
	}
	var services []string
	for _, ev := range resp.Kvs {
		arr := strings.Split(string(ev.Key), r.cfg.Service.EtcdKey+"_")
		service := arr[1]
		services = append(services, service)
	}
	return services
}

//添加服务节点（通过添加addNode操作到服务操作struct中实现）
func (r *Rainbow) addServiceNode(name, addr string) {
	sm := serviceOperate{
		operate:     "addNode",
		serviceName: name,
		serviceAddr: addr,
	}
	r.serviceManager <- sm
}

//删除服务节点（通过添加delNode操作到服务操作struct中实现）
func (r *Rainbow) delServiceNode(name, addr string) {
	sm := serviceOperate{
		operate:     "delNode",
		serviceName: name,
		serviceAddr: addr,
	}
	r.serviceManager <- sm
}

//etcd 前缀watch监视服务变动，当监视对象发生变化的时候被会被记录
func (r *Rainbow) serviceWatcher() {
	client := r.GetEtcd()
	rch := client.Watch(context.Background(), r.cfg.Service.EtcdKey+"_", clientV3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			arr := strings.Split(string(ev.Kv.Key), "_")
			serviceName := arr[1]
			httpAddr := arr[2]
			serviceAddr := httpAddr
			switch ev.Type {
			case 0: //put
				r.addServiceNode(serviceName, serviceAddr)
				r.Log(InfoLevel, "Service", fmt.Sprintf("[%s] node [%s] join", serviceName, serviceAddr))
			case 1: //delete
				r.delServiceNode(serviceName, serviceAddr)
				r.Log(InfoLevel, "Service", fmt.Sprintf("[%s] node [%s] leave", serviceName, serviceAddr))
			}
		}
	}
}

//获取服务的rpc地址
func (r *Rainbow) getServiceRpcAddr(name string, index int) (string, error) {
	if index > len(r.services[name].Nodes)-1 {
		return "", errors.New("service node not found")
	}
	arr := strings.Split(strings.Split(r.services[name].Nodes[index].Addr, "_")[0], ":")
	return arr[0] + ":" + arr[2], nil
}

func (r *Rainbow) selectService(name string) (string, int, error) {
	if _, ok := r.services[name]; !ok {
		return "", 0, errors.New("service not found")
	}

	var waitingMin int64 = 0
	nodeIndex := 0
	nodeLen := len(r.services[name].Nodes) //同一服务中结点的数目
	if nodeLen < 1 {
		return "", 0, errors.New("service node not found")
	} else if nodeLen > 1 {
		// 获取最小waiting的服务结点
		for k, v := range r.services[name].Nodes {
			if k == 0 {
				waitingMin = atomic.LoadInt64(&v.Waiting) //初始化waitingMin为第一个节点的waiting值
				continue
			}
			if t := atomic.LoadInt64(&v.Waiting); t < waitingMin { //找出最小waiting的节点的索引
				nodeIndex = k
				waitingMin = t
			}
		}
		// 如果所有结点的waiting值都为0，则随机选择一个结点
		if waitingMin == 0 {
			nodeIndex = rand.Intn(nodeLen)
		} /* else { //test
			fmt.Println("not rand")
		}*/
	}

	return r.services[name].Nodes[nodeIndex].Addr, nodeIndex, nil
}

func (r *Rainbow) getServiceHttpAddr(name string, index int) (string, error) {
	if index > len(r.services[name].Nodes)-1 {
		return "", errors.New("service node not found")
	}
	arr := strings.Split(strings.Split(r.services[name].Nodes[index].Addr, "_")[0], ":")
	return arr[0] + ":" + arr[1], nil
}

func (r *Rainbow) getServicesByName(serviceName string) []string {
	client := r.GetEtcd()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := client.Get(ctx, r.cfg.Service.EtcdKey+"_"+serviceName, clientV3.WithPrefix())
	cancel()
	if err != nil {
		r.Log(ErrorLevel, "GetServicesByName", err)
		return []string{}
	}
	var services []string
	for _, ev := range resp.Kvs {
		arr := strings.Split(string(ev.Key), r.cfg.Service.EtcdKey+"_"+serviceName+"_")
		serviceAddr := arr[1]
		services = append(services, serviceAddr)
	}
	return services
}
