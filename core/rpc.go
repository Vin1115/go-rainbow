package core

//finished
import (
	"context"
	"github.com/opentracing/opentracing-go"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/server"
	"github.com/smallnest/rpcx/share"
	"time"
)

//创建一个rpcx的服务端并为该rpc注册，并启动服务端监听客户端rpc请求
func (r *Rainbow) rpcListen(name, network, address string, obj interface{}, metadata string) error { //obj为定义为name的服务对象实例
	s := server.NewServer()
	if err := s.RegisterName(name, obj, metadata); err != nil {
		return err
	}
	r.Log(InfoLevel, "rpc", "listen on: "+address)
	if err := s.Serve(network, address); err != nil {
		return err
	}
	return nil
}

func rpcCall(span opentracing.Span, addr, service, method string, args, reply interface{}, timeout int) error {
	//定义了使用什么方式来实现服务发现。 在这里我们使用最简单的 Peer2PeerDiscovery（点对点）。客户端直连服务器来获取服务地址。
	d, err := client.NewPeer2PeerDiscovery("tcp@"+addr, "")
	if err != nil {
		return err
	}
	//创建了 XClient， 并且传进去了 FailMode、 SelectMode 和默认选项。
	//FailMode 告诉客户端如何处理调用失败：重试、快速返回，或者 尝试另一台服务器。
	//SelectMode 告诉客户端如何在有多台服务器提供了同一服务的情况下选择服务器。
	xClient := client.NewXClient(service, client.Failtry, client.RandomSelect, d, client.DefaultOption)
	defer xClient.Close()

	textMapString := map[string]string{}
	if span != nil {
		textMap := opentracing.TextMapCarrier{}
		opentracing.GlobalTracer().Inject(
			span.Context(),
			opentracing.TextMap,
			textMap)
		// write opentracing span to textMap
		textMap.ForeachKey(func(key, val string) error {
			textMapString[key] = val
			return nil
		})
	}

	// rpc timeout and value
	ctx := context.WithValue(context.Background(), share.ReqMetaDataKey, textMapString)
	ctx2, _ := context.WithTimeout(ctx, time.Millisecond*time.Duration(timeout))
	err = xClient.Call(ctx2, method, args, reply) //调用了远程服务并且同步获取结果

	if err != nil {
		return err
	}
	return nil
}

//todo what does it mean
// StartSpanFormRpc start and get opentracing span from rpc call
func StartSpanFormRpc(ctx context.Context, operateName string) opentracing.Span {
	reqMeta := ctx.Value(share.ReqMetaDataKey).(map[string]string)
	span := StartSpanFromTextMap(reqMeta, operateName)
	return span
}
