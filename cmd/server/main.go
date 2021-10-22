package main

import (
	"cacheservice/internal/server"
	"cacheservice/pkg/model"
	"cacheservice/proto"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const port = 5555

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if err := startCacheServer(); err != nil {
		log.Fatal(err)
	}
}

// startCacheServer will start listen to a port (default 5555), create a gRPC server,
// register Get() & Set() CmpAndSet() and Watch() implementatation and range over a map created here that
// will execute a func passed from those implementations when the Serve receives
// the query.
func startCacheServer() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("startCacheServer: %w", err)
	}
	log.Println("listening on port:", port)
	defer listener.Close()

	fmt.Println("Initiating cache")
	grpcServer := grpc.NewServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigsCh := make(chan os.Signal, 1)
		signal.Notify(sigsCh, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigsCh)

		select {
		case <-ctx.Done():
			return
		case <-sigsCh:
			log.Println("shutting down...")
		}

		go grpcServer.GracefulStop()
		select {
		case <-ctx.Done():
			return
		case <-sigsCh:
			log.Println("shutdown gracefully...")
		}
		grpcServer.Stop()
	}()

	// use my interface instead of the unimplemented one in grpc
	ch := model.NewCache(ctx, 0)
	myServerImpl := &server.Server{Cache: ch}
	proto.RegisterCacheServiceServer(grpcServer, myServerImpl)

	return grpcServer.Serve(listener)
}
