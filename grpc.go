package micro

import (
	"github.com/leicc520/go-orm"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/leicc520/go-core/proto"
	"github.com/leicc520/go-orm/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const _GRPCPROTO_ = "grpc"

type RegisterGrpcServiceHandler func(s *grpc.Server)
type GRPCServiceSrv struct {
	name string				//GRPC服务名称
	port int64				//GRPC服务端口
	srv  string     		//GRPC服务地址
	version string  		//GRPC服务版本
	msrv *MicRegSrv 		//GRPC发现服务
	sysStop chan os.Signal 	//系统优雅重启
	srvHandler []RegisterGrpcServiceHandler
}

type GrpcConfigSt struct {
	Name string 	`yaml:"name"`    //定义GRPC服务名称
	Version string 	`yaml:"version"` //定义GRPC服务版本号
}

//绑定心跳的出来逻辑
func (g *GRPCServiceSrv) Health(ctx context.Context, in *proto.GrpcHealthRequest) (*proto.GrpcHealthResponse, error) {
	data := &proto.GrpcHealthResponse{Code: 0, Msg: "OK"}
	log.Write(log.INFO, g.srv + " health check ok")
	return data, nil
}

//绑定获取全局的id记录信息
func (g *GRPCServiceSrv) GetId(ctx context.Context, in *proto.GrpcGetIdRequest) (*proto.GrpcGetIdResponse, error) {
	ids    := GetId(in.DataCenterId, 1)
	autoId := int64(-1)
	if ids != nil && len(ids) == 1 {
		autoId = ids[0]
	}
	data := &proto.GrpcGetIdResponse{Id: autoId}
	log.Write(log.INFO, g.srv + " GRPC GetId ok ", autoId)
	return data, nil
}

//绑定获取全局的id记录信息
func (g *GRPCServiceSrv) GetIds(ctx context.Context, in *proto.GrpcGetIdsRequest) (*proto.GrpcGetIdsResponse, error) {
	ids  := GetId(in.DataCenterId, int(in.Nums))
	data := &proto.GrpcGetIdsResponse{Id: ids}
	log.Write(log.INFO, g.srv + " getids ok", data.Id)
	return data, nil
}

//初始化GRPC服务对象结构
func NewGRPCServiceSrvSt(grpc *GrpcConfigSt, msrv *MicRegSrv) *GRPCServiceSrv {
	srv := &GRPCServiceSrv{name:grpc.Name, port: 0, msrv: msrv, sysStop: make(chan os.Signal),
		version: grpc.Version, srvHandler: make([]RegisterGrpcServiceHandler, 0)}
	return srv
}

//注册开启服务的信息回调
func (g *GRPCServiceSrv) Register(handle RegisterGrpcServiceHandler)  {
	g.srvHandler = append(g.srvHandler, handle)
}

//服务启动失败的情况注销服务
func (g *GRPCServiceSrv) UnRegister()  {
	if g.msrv != nil && strings.HasPrefix(g.msrv.RegSrv, "http") {
		g.msrv.UnRegister(_GRPCPROTO_, g.name, g.srv)
	}
}

//启动Grpc 服务
func (g *GRPCServiceSrv) Start(port int64) error  {
	addStr := "0.0.0.0:0" //默认随机端口
	if port > 0 {
		addStr = ":"+strconv.FormatInt(port, 10)
	}
	orm.WritePidFile(g.name) //写入进程pid数据资料信息
	lis, err := net.Listen("tcp", addStr)
	if err != nil {//tcp服务开启异常的情况
		log.Write(log.ERROR, "启动grpc服务异常"+err.Error())
		return err
	}
	g.port = int64(lis.Addr().(*net.TCPAddr).Port)
	grpcSrv:= grpc.NewServer(proto.GrpcDefaultInterceptors()...) //创建gRPC服务
	proto.RegisterGrpcCoreServiceServer(grpcSrv, g)
	for _, handle := range g.srvHandler {
		handle(grpcSrv) //执行GRPC服务的函数
	}
	reflection.Register(grpcSrv) //注册微服务处理逻辑
	if g.msrv != nil && strings.HasPrefix(g.msrv.RegSrv, "http") {
		time.AfterFunc(time.Second, func() {//服务注册延迟处理 延迟注册上报服务
			g.srv = strconv.FormatInt(g.port, 10)
			g.srv = g.msrv.Register(g.name, g.srv, _GRPCPROTO_,  g.version)
		})
	}
	//监听服务器kill 命令 重启服务
	signal.Notify(g.sysStop, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
	errChan := make(chan error)
	go func() {//监听系统促发的事件模型 优雅的终止GRPC服务
		select {
			case <-errChan:
			case <-g.sysStop:
				log.Write(-1, "优雅的关闭了GRPC服务...")
				grpcSrv.GracefulStop()
		}
	}()
	log.Write(-1, "启动grpc服务成功,Port:"+strconv.FormatInt(g.port, 10))
	if err = grpcSrv.Serve(lis); err != nil {
		log.Write(log.FATAL, "绑定监听GRPC服务失败"+err.Error())
		errChan <- err
	}
	return nil
}