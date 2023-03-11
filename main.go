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
	Size       int64
}

type MediaDescription struct {
	MetadataSize uint64
	Chunks       []MediaChunk
}

func mapTracks(tracks mp4.Tracks) map[uint32]*mp4.Track {
	m := make(map[uint32]*mp4.Track)
	for _, track := range tracks {
		m[track.TrackID] = track
	}
	return m
}

func ReadMediaDescription(file *os.File) (*MediaDescription, error) {
	info, err := mp4.Probe(bufseekio.NewReadSeeker(file, 1024, 4))
	if err != nil {
		return nil, err
	}
	mediaDescription := MediaDescription{
		MetadataSize: info.Segments[0].MoofOffset,
		Chunks:       make([]MediaChunk, len(info.Segments)),
	}
	tracks := mapTracks(info.Tracks)
	var timer float32
	for i := 0; i < len(info.Segments); i++ {
		segment := info.Segments[i]
		duration := float32(segment.Duration) / float32(tracks[segment.TrackID].Timescale)
		endTime := timer + duration
		chunk := MediaChunk{
			ByteOffset: int64(segment.MoofOffset),
			StartTime:  timer,
			EndTime:    endTime,
		}
		timer = endTime
		mediaDescription.Chunks[i] = chunk
		if i > 0 {
			mediaDescription.Chunks[i-1].Size = int64(segment.MoofOffset) - mediaDescription.Chunks[i-1].ByteOffset
		}
	}
	fi, _ := file.Stat()
	mediaDescription.Chunks[len(info.Segments)-1].Size = fi.Size()
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
	file, err := os.Open(fmt.Sprintf("./out/%d.mp4", req.VideoId))
	if err != nil {
		return status.Errorf(codes.Internal, "Erreur de lecture du media: %v", err)
	}
	defer file.Close()
	mediaDescription, err := ReadMediaDescription(file)

	if _, err = file.Seek(0, 0); err != nil {
		return err
	}
	// Send metadata
	buf := make([]byte, mediaDescription.MetadataSize)
	n, err := file.Read(buf)
	if err != nil {
		return status.Errorf(codes.Internal, "Erreur de lecture du fichier de sortie: %v", err)
	}
	if err := stream.Send(&pb.VideoResponse{Data: buf[:n]}); err != nil {
		return status.Errorf(codes.Internal, "Erreur d'envoi des données de la vidéo: %v", err)
	}
	i, err := mediaDescription.SeekChunk(req.Seek)
	if err != nil {
		return err
	}
	// Send chunks
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

	return nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Échec de l'écoute sur le port %v: %v", port, err)
	}
	grpcServer := grpc.NewServer()
	wrappedServer := grpcweb.WrapServer(grpcServer)
	pb.RegisterVideoServiceServer(grpcServer, &server{})
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
