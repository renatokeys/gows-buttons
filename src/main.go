package main

import (
	"flag"
	"fmt"
	gowsLog "github.com/devlikeapro/gows/log"
	pb "github.com/devlikeapro/gows/proto"
	"github.com/devlikeapro/gows/server"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/debug"
)

func listenSocket(log waLog.Logger, path string) *net.Listener {
	log.Infof("Server is listening on %s", path)
	// Force remove the socket file
	_ = os.Remove(path)
	// Listen on a specified port
	listener, err := net.Listen("unix", path)
	if err != nil {
		log.Errorf("Failed to listen: %v", err)
	}
	return &listener
}

func buildGrpcServer(log waLog.Logger) *grpc.Server {
	// 128 MB
	maxMessageSize := 128 * 1024 * 1024

	// Define a custom recovery function to handle panics
	recoveryOpts := []recovery.Option{
		recovery.WithRecoveryHandler(func(p interface{}) (err error) {
			stack := debug.Stack()
			log.Errorf("Panic: %v. Stack: %s", p, stack)
			return status.Errorf(codes.Internal, "Internal server error: %v. Stack: %v", p, stack)
		}),
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recovery.UnaryServerInterceptor(recoveryOpts...),
		),
		grpc.ChainStreamInterceptor(
			recovery.StreamServerInterceptor(recoveryOpts...),
		),
		grpc.MaxRecvMsgSize(maxMessageSize),
		grpc.MaxSendMsgSize(maxMessageSize),
	)
	srv := server.NewServer()
	// Add an event handler to the client
	pb.RegisterMessageServiceServer(grpcServer, srv)
	pb.RegisterEventStreamServer(grpcServer, srv)
	return grpcServer
}

var (
	socket    string
	pprofFlag bool
	pprofPort int
	pprofHost string
)

func init() {
	flag.StringVar(&socket, "socket", "/tmp/gows.sock", "Socket path")
	flag.BoolVar(&pprofFlag, "pprof", false, "Enable pprof HTTP server")
	flag.IntVar(&pprofPort, "pprof-port", 6060, "Port for pprof HTTP server")
	flag.StringVar(&pprofHost, "pprof-host", "localhost", "Host for pprof HTTP server")
}

func startPprofServer(log waLog.Logger) {
	if !pprofFlag {
		return
	}

	addr := fmt.Sprintf("%s:%d", pprofHost, pprofPort)
	log.Infof("Starting pprof HTTP server on %s", addr)

	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Errorf("Failed to start pprof HTTP server: %v", err)
		}
	}()
}

func remove(path string) {
	_ = os.Remove(path)
}

func main() {
	flag.Parse()
	log := gowsLog.Stdout("Server", "DEBUG", false)

	// Start pprof HTTP server if enabled
	startPprofServer(log)

	// Build the server
	grpcServer := buildGrpcServer(log)
	// Open unix socket
	log.Infof("Opening socket %s", socket)
	listener := listenSocket(log, socket)
	defer remove(socket)

	// Start the server
	log.Infof("gRPC server started!")
	if err := grpcServer.Serve(*listener); err != nil {
		log.Errorf("Failed to serve: %v", err)
	}
}
