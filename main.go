package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/abema/go-mp4"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/sunfish-shogi/bufseekio"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	StartTime  float64
	EndTime    float64
	ByteOffset int64
	Size       int64
}

type MediaMetadata struct {
	MetadataSize  uint64
	VideoDuration float64
	AudioTracks   []*pb.TrackLanguage
	TextTracks    []*pb.TrackLanguage
}

func ReadMetadata(file *os.File) (*MediaMetadata, error) {
	moov, err := mp4.ExtractBox(file, nil, mp4.BoxPath{mp4.BoxTypeMoov()})
	if err != nil {
		return nil, err
	}
	mediaMetadata := MediaMetadata{
		MetadataSize: moov[0].Size,
		AudioTracks:  make([]*pb.TrackLanguage, 0),
		TextTracks:   make([]*pb.TrackLanguage, 0),
	}
	hdlrBoxes, err := mp4.ExtractBoxWithPayload(file, nil, mp4.BoxPath{mp4.BoxTypeMoov(), mp4.BoxTypeTrak(), mp4.BoxTypeMdia(), mp4.BoxTypeHdlr()})
	mdhdBoxes, err := mp4.ExtractBoxWithPayload(file, nil, mp4.BoxPath{mp4.BoxTypeMoov(), mp4.BoxTypeTrak(), mp4.BoxTypeMdia(), mp4.BoxTypeMdhd()})
	if err != nil {
		return nil, err
	}
	for i := range hdlrBoxes {
		hldr := hdlrBoxes[i].Payload.(*mp4.Hdlr)
		mdhd := mdhdBoxes[i].Payload.(*mp4.Mdhd)
		if hldr.HandlerType == [4]byte{'s', 'o', 'u', 'n'} {
			mediaMetadata.AudioTracks = append(mediaMetadata.AudioTracks, &pb.TrackLanguage{Name: hldr.Name, Language: bytesToAsciiStr(mdhd.Language)})
		} else if hldr.HandlerType == [4]byte{'s', 'b', 't', 'l'} {
			mediaMetadata.TextTracks = append(mediaMetadata.TextTracks, &pb.TrackLanguage{Name: hldr.Name, Language: bytesToAsciiStr(mdhd.Language)})
		}
	}
	return &mediaMetadata, nil
}

func findVideoTrack(file *os.File) (*mp4.Track, error) {
	hdlrBoxes, err := mp4.ExtractBoxWithPayload(file, nil, mp4.BoxPath{mp4.BoxTypeMoov(), mp4.BoxTypeTrak(), mp4.BoxTypeMdia(), mp4.BoxTypeHdlr()})
	mdhdBoxes, err := mp4.ExtractBoxWithPayload(file, nil, mp4.BoxPath{mp4.BoxTypeMoov(), mp4.BoxTypeTrak(), mp4.BoxTypeMdia(), mp4.BoxTypeMdhd()})
	tkhdBoxes, err := mp4.ExtractBoxWithPayload(file, nil, mp4.BoxPath{mp4.BoxTypeMoov(), mp4.BoxTypeTrak(), mp4.BoxTypeTkhd()})
	if err != nil {
		return nil, err
	}
	for i := range hdlrBoxes {
		hldr := hdlrBoxes[i].Payload.(*mp4.Hdlr)
		tkhd := tkhdBoxes[i].Payload.(*mp4.Tkhd)
		mdhd := mdhdBoxes[i].Payload.(*mp4.Mdhd)
		if hldr.HandlerType == [4]byte{'v', 'i', 'd', 'e'} {
			return &mp4.Track{TrackID: tkhd.TrackID, Timescale: mdhd.Timescale}, nil
		}
	}
	return nil, errors.New("not found")
}

func bytesToAsciiStr(bytes [3]byte) string {
	for i, value := range bytes {
		bytes[i] = value + 96
	}
	return string(bytes[:])
}

func ReadMediaDescription(file *os.File) ([]*MediaChunk, error) {
	info, err := mp4.Probe(bufseekio.NewReadSeeker(file, 1024, 4))
	if err != nil {
		return nil, err
	}
	chunks := make([]*MediaChunk, 0)

	videoTrack, err := findVideoTrack(file)
	if err != nil {
		return nil, err
	}
	var timer float64
	for i := 0; i < len(info.Segments); i++ {
		segment := info.Segments[i]
		if videoTrack.TrackID != segment.TrackID {
			continue
		}
		fmt.Printf("%d\n", segment.TrackID)
		duration := float64(segment.Duration) / float64(videoTrack.Timescale)
		endTime := timer + duration
		chunk := MediaChunk{
			ByteOffset: int64(segment.MoofOffset),
			StartTime:  timer,
			EndTime:    endTime,
		}
		timer = endTime
		chunks = append(chunks, &chunk)
		if len(chunks) > 0 {
			chunks[len(chunks)-1].Size = int64(segment.MoofOffset) - chunks[len(chunks)-1].ByteOffset
		}
	}
	fi, _ := file.Stat()
	chunks[len(chunks)-1].Size = fi.Size()
	fmt.Printf("%v\n", chunks[len(chunks)-1])
	return chunks, nil
}

func SeekChunk(chunks []*MediaChunk, timeCode float64) (int, error) {
	for index, chunk := range chunks {
		if timeCode >= chunk.StartTime && timeCode < chunk.EndTime {
			return index, nil
		}
	}
	return 0, status.Errorf(codes.NotFound, "Impossible de trouver un segment pour le time code %f", timeCode)
}

func (s *server) GetVideoMetadata(_ context.Context, req *pb.VideoMetadataRequest) (*pb.VideoMetadata, error) {
	file, err := os.Open(fmt.Sprintf("./out/%d.mp4", req.VideoId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Erreur de lecture du media: %v", err)
	}
	defer file.Close()
	mediaMetadata, err := ReadMetadata(file)

	buf := make([]byte, mediaMetadata.MetadataSize)
	if _, err := file.Seek(0, int(mediaMetadata.MetadataSize)); err != nil {
		return nil, err
	}
	n, err := file.Read(buf)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Erreur de lecture du fichier de sortie: %v", err)
	}

	return &pb.VideoMetadata{
		Metadata:            buf[:n],
		VideoDuration:       mediaMetadata.VideoDuration,
		TextTrackLanguages:  mediaMetadata.TextTracks,
		AudioTrackLanguages: mediaMetadata.AudioTracks,
	}, nil
}

func (s *server) GetVideoChunk(_ context.Context, req *pb.VideoChunkRequest) (*pb.VideoChunk, error) {
	file, err := os.Open(fmt.Sprintf("./out/%d.mp4", req.VideoId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Erreur de lecture du media: %v", err)
	}
	defer file.Close()
	chunks, err := ReadMediaDescription(file)

	chunkIndex, err := SeekChunk(chunks, req.Seek)
	chunk := chunks[chunkIndex]
	buf := make([]byte, chunk.Size)
	if _, err := file.Seek(chunk.ByteOffset, 0); err != nil {
		return nil, err
	}
	n, err := file.Read(buf)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Erreur de lecture du fichier de sortie: %v", err)
	}

	return &pb.VideoChunk{Data: buf[:n], StartTime: chunk.StartTime, EndTime: chunk.EndTime}, nil
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
