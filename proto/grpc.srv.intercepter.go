package proto

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/leicc520/go-orm"
	"github.com/leicc520/go-orm/log"
	"google.golang.org/grpc"
)

//普通格式GRPC数据恢复处理逻辑
func grpcUnaryRecovery(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	st := time.Now()
	defer func() {
		if r := recover(); r != nil {
			rtStack := orm.RuntimeStack(3)
			log.Write(log.ERROR, r, info, string(rtStack))
			err = errors.New(fmt.Sprintf("%v", r))
		}
		log.Write(log.INFO, info, "grpc 执行时间:", time.Since(st))
	}()
	log.Write(log.INFO, "UnaryInterceptor UnaryRecovery")
	resp, err = handler(ctx, req)
	return resp, err
}

//流格式GRPC数据恢复处理逻辑
func grpcStreamRecovery(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	st := time.Now()
	defer func() {
		if r := recover(); r != nil {
			rtStack := orm.RuntimeStack(3)
			log.Write(log.ERROR, r, info, string(rtStack))
			err = errors.New(fmt.Sprintf("%v", r))
		}
		log.Write(log.INFO, info, "grpc 执行时间:", time.Since(st))
	}()
	log.Write(log.INFO, "StreamInterceptor StreamRecovery")
	err = handler(srv, ss)
	return err
}

//创建默认的grpc服务拦截器的业务逻辑
func GrpcDefaultInterceptors() []grpc.ServerOption {
	cb := []grpc.ServerOption{grpc.UnaryInterceptor(grpcUnaryRecovery), grpc.StreamInterceptor(grpcStreamRecovery)}
	return cb
}


