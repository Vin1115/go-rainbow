package core

//finished
import (
	"bytes"
	"context"
	"strings"
)

type Rpc int

type SyncRoutesArgs struct {
	Yml []byte
}

type SyncRoutesReply struct {
	Result bool
}

//被通过rpcx远程调用
//同步routes.yml方法,args为要写入filepath里的filedata
func (rpc *Rpc) SyncRoutes(ctx context.Context, args *SyncRoutesArgs, reply *SyncRoutesReply) error {
	reply.Result = true
	if err := writeFile("./configs/routes.yml", args.Yml); err != nil {
		reply.Result = false
	}
	return nil
}

//若routes.yml被监视到发生变化，则会调用sendRoutes（）同步配置
func (r *Rainbow) sendRoutes() {
	fileData, err := readFile("configs/routes.yml")
	if err != nil {
		r.Log(ErrorLevel, "SyncRoutes", err)
		return
	}

	if len(fileData) == 0 {
		r.Log(WarnLevel, "SyncRoutes route.yml's length is 0", nil)
		return
	}

	if bytes.Compare(r.syncCache, fileData) == 0 {
		return //与syncCache同步缓存中相等则返回
	}

	r.syncCache = fileData

	args := SyncRoutesArgs{
		Yml: fileData,
	}
	reply := SyncRoutesReply{}
	for k1, v1 := range r.services {
		for k2, v2 := range v1.Nodes {
			if strings.Compare(v2.Addr, r.GetServiceId()) == 0 {
				continue
			}

			addr, err := r.getServiceRpcAddr(k1, k2)
			if err != nil {
				r.Log(ErrorLevel, "getServiceRpcAddr", err)
				continue
			}
			//rpccall rpcx远程调用SyncRoutes，根据参数args，返回结果reply
			if err := rpcCall(nil, addr, k1, "SyncRoutes", &args, &reply, 10000); err != nil {
				r.Log(ErrorLevel, "SyncRoutes", err)
				return
			}
			if !reply.Result {
				r.Log(ErrorLevel, "SyncRoutes", "fail")
			}
			r.Log(InfoLevel, "SyncRoutes", "success")

		}
	}

}
