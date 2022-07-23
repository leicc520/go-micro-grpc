package proto

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"runtime"
	"time"

	"github.com/leicc520/go-orm/log"
	"google.golang.org/grpc"
)

//获取服务器panic 指定情况的获取写日志
func RuntimeStack(skip int) []byte {
	buf := new(bytes.Buffer)
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		fmt.Fprintf(buf, "\t%s: %s\n", func(pcPtr uintptr) []byte {
			fn := runtime.FuncForPC(pcPtr)
			if fn == nil {
				return []byte("no")
			}
			name := []byte(fn.Name())
			return name
		}(pc) , func(lines [][]byte, n int) []byte {
			n--
			if n < 0 || n >= len(lines) {
				return []byte("no")
			}
			return bytes.TrimSpace(lines[n])
		}(lines, line))
	}
	return buf.Bytes()
}

//普通格式GRPC数据恢复处理逻辑
func grpcUnaryRecovery(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	st := time.Now()
	defer func() {
		if r := recover(); r != nil {
			rtStack := RuntimeStack(3)
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
			rtStack := RuntimeStack(3)
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


