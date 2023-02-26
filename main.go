package main

import (
	"fmt"
	"github.com/abema/go-mp4"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/sunfish-shogi/bufseekio"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	pb "video-streaming/proto"
)

const (
	port = ":50051"
)

type server struct {
	pb.UnimplementedVideoServiceServer
}

type MediaChunk struct {
	StartTime  float32
	EndTime    float32
	ByteOffset int64
	Size       uint32
}

type MediaDescription struct {
	MetadataSize uint64
	Chunks       []MediaChunk
}

func ReadMediaDescription(file *os.File) (*MediaDescription, error) {
	info, err := mp4.Probe(bufseekio.NewReadSeeker(file, 1024, 4))
	if err != nil {
		return nil, err
	}
	mediaDescription := MediaDescription{
		MetadataSize: info.Segments[0].MoofOffset - 1,
		Chunks:       make([]MediaChunk, len(info.Segments)),
	}
	var timer float32
	for index, segment := range info.Segments {
		duration := float32(segment.Duration) / float32(info.Tracks[segment.TrackID-1].Timescale)
		endTime := timer + duration
		chunk := MediaChunk{
			ByteOffset: int64(segment.MoofOffset),
			Size:       segment.Size,
			StartTime:  timer,
			EndTime:    endTime,
		}
		timer = endTime
		mediaDescription.Chunks[index] = chunk
	}
	return &mediaDescription, nil
}

func (m *MediaDescription) SeekChunk(timeCode float32) (int, error) {
	for index, chunk := range m.Chunks {
		if timeCode >= chunk.StartTime && timeCode <= chunk.EndTime {
			return index, nil
		}
	}
	return 0, status.Errorf(codes.NotFound, "Impossible de trouver un segment pour le time code %f", timeCode)
}

func (s *server) GetVideoStream(req *pb.VideoRequest, stream pb.VideoService_GetVideoStreamServer) error {
	fmt.Printf("test")
	file, err := os.Open(fmt.Sprintf("./out/%d.mp4", req.VideoId))
	if err != nil {
		return status.Errorf(codes.Internal, "Erreur de lecture du media: %v", err)
	}
	fmt.Printf("test")
	defer file.Close()
	mediaDescription, err := ReadMediaDescription(file)
	// Send metadata
	buf := make([]byte, mediaDescription.MetadataSize)
	n, err := file.Read(buf)
	fmt.Printf("test")
	if err != nil {
		return status.Errorf(codes.Internal, "Erreur de lecture du fichier de sortie: %v", err)
	}
	if err := stream.Send(&pb.VideoResponse{MetaData: buf[:n]}); err != nil {
		return status.Errorf(codes.Internal, "Erreur d'envoi des données de la vidéo: %v", err)
	}
	fmt.Printf("test")
	i, err := mediaDescription.SeekChunk(req.Seek)
	if err != nil {
		return err
	}
	// Send chunks
	fmt.Printf("test")
	for ; i < len(mediaDescription.Chunks); i++ {
		chunk := mediaDescription.Chunks[i]
		buf := make([]byte, chunk.Size)
		if _, err := file.Seek(chunk.ByteOffset, 0); err != nil {
			return err
		}
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return status.Errorf(codes.Internal, "Erreur de lecture du fichier de sortie: %v", err)
		}

		if err := stream.Send(&pb.VideoResponse{Data: buf[:n]}); err != nil {
			return status.Errorf(codes.Internal, "Erreur d'envoi des données de la vidéo: %v", err)
		}
	}
	fmt.Printf("fin")

	return nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Échec de l'écoute sur le port %v: %v", port, err)
	}
	s := grpc.NewServer()
	wrappedServer := grpcweb.WrapServer(s)
	pb.RegisterVideoServiceServer(s, &server{})
	handler := func(resp http.ResponseWriter, req *http.Request) {
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
