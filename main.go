package main

import (
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
	pb "video-streaming/proto"
	"video-streaming/src"
)

const (
	port = ":50051"
)

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Échec de l'écoute sur le port %v: %v", port, err)
	}
	grpcServer := grpc.NewServer()
	wrappedServer := grpcweb.WrapServer(grpcServer)
	pb.RegisterVideoServiceServer(grpcServer, &src.Server{})
	handler := func(resp http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodOptions {
			resp.Header().Set("Access-Control-Allow-Origin", "*")
			resp.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			resp.Header().Set("Access-Control-Allow-Headers", "Content-Type, x-grpc-web, x-user-agent")
			resp.Header().Set("Access-Control-Max-Age", "600")
			resp.WriteHeader(http.StatusOK)
			return
		}
		resp.Header().Set("Access-Control-Allow-Origin", "*")
		resp.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		resp.Header().Set("Access-Control-Allow-Headers", "Content-Type, x-grpc-web, x-user-agent")
		resp.Header().Set("Access-Control-Allow-Credentials", "true")
		if wrappedServer.IsGrpcWebRequest(req) {
			wrappedServer.ServeHTTP(resp, req)
		} else {
			http.NotFound(resp, req)
		}
	}
	if err := http.Serve(lis, http.HandlerFunc(handler)); err != nil {
		log.Fatalf("Échec du lancement du serveur gRPC: %v", err)
	}
}
