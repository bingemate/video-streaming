package main

import (
	"fmt"
	"github.com/abema/go-mp4"
	"github.com/sunfish-shogi/bufseekio"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"net"
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
	StartTime  float64
	EndTime    float64
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
	var timer float64
	for index, segment := range info.Segments {
		duration := float64(segment.Duration) / float64(info.Tracks[segment.TrackID-1].Timescale)
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

func (s *server) GetVideoStream(req *pb.VideoRequest, stream pb.VideoService_GetVideoStreamServer) error {
	file, err := os.Open(fmt.Sprintf("./out/%d.mp4", req.VideoId))
	if err != nil {
		return status.Errorf(codes.Internal, "Erreur de lecture du media: %v", err)
	}
	defer file.Close()
	mediaDescription, err := ReadMediaDescription(file)
	// Send metadata
	buf := make([]byte, mediaDescription.MetadataSize)
	n, err := file.Read(buf)
	if err != nil {
		return status.Errorf(codes.Internal, "Erreur de lecture du fichier de sortie: %v", err)
	}
	if err := stream.Send(&pb.VideoResponse{MetaData: buf[:n]}); err != nil {
		return status.Errorf(codes.Internal, "Erreur d'envoi des données de la vidéo: %v", err)
	}
	// Send chunks
	for _, chunk := range mediaDescription.Chunks {
		buf := make([]byte, chunk.Size)
		_, err := file.Seek(chunk.ByteOffset, 0)
		if err != nil {
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
	s := grpc.NewServer()
	pb.RegisterVideoServiceServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Échec du lancement du serveur gRPC: %v", err)
	}
}
