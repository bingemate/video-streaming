package src

import (
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	pb "video-streaming/proto"
)

type Server struct {
	pb.UnimplementedVideoServiceServer
}

var videoMetadata = make(map[uint64]*MediaMetadata)

func getMetadata(videoId uint64) (*MediaMetadata, error) {
	if videoMetadata[videoId] != nil {
		return videoMetadata[videoId], nil
	}
	file, err := os.Open(fmt.Sprintf("./out/%d.mp4", videoId))
	defer file.Close()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "File open failed: %v", err)
	}
	metadata, err := ReadMetadata(file)
	videoMetadata[videoId] = metadata
	return ReadMetadata(file)
}

func trackLanguageMapper(mediaTracks []*MediaTrack) []*pb.TrackLanguage {
	tracks := make([]*pb.TrackLanguage, len(mediaTracks))
	for i, track := range mediaTracks {
		tracks[i] = &pb.TrackLanguage{
			Id:       track.TrackId,
			Language: track.Language,
			Name:     track.Name,
			Codec:    track.Codec,
		}
	}
	return tracks
}

func (s *Server) GetVideoMetadata(_ context.Context, req *pb.VideoMetadataRequest) (*pb.VideoMetadata, error) {
	file, err := os.Open(fmt.Sprintf("./out/%d.mp4", req.VideoId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "File open failed: %v", err)
	}
	defer file.Close()
	mediaMetadata, err := getMetadata(req.VideoId)

	buf := make([]byte, mediaMetadata.Size)
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}
	n, err := file.Read(buf)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "File read failed: %v", err)
	}

	return &pb.VideoMetadata{
		Metadata:            buf[:n],
		VideoDuration:       mediaMetadata.VideoDuration,
		TextTrackLanguages:  trackLanguageMapper(mediaMetadata.TextTracks),
		AudioTrackLanguages: trackLanguageMapper(mediaMetadata.AudioTracks),
	}, nil
}

func (s *Server) GetVideoChunk(_ context.Context, req *pb.VideoChunkRequest) (*pb.VideoChunk, error) {
	file, err := os.Open(fmt.Sprintf("./out/%d.mp4", 2))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "File open failed: %v", err)
	}
	defer file.Close()
	metadata, err := getMetadata(req.VideoId)

	chunkIndex, err := SeekChunk(metadata, req.Seek)
	moofChunk := metadata.MoofChunks[chunkIndex]
	videoChunk := moofChunk.Chunks[metadata.VideoTrack.TrackId]
	//audioChunk := moofChunk.Chunks[metadata.AudioTracks[0].TrackId]
	videoBuf := make([]byte, videoChunk.Size)
	//audioBuf := make([]byte, audioChunk.Size)
	moofBuf := make([]byte, moofChunk.Size)
	if _, err := file.Seek(moofChunk.Offset, 0); err != nil {
		return nil, err
	}
	n, err := file.Read(moofBuf)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "File read failed: %v", err)
	}

	if _, err := file.Seek(videoChunk.ByteOffset, 0); err != nil {
		return nil, err
	}
	o, err := file.Read(videoBuf)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "File read failed: %v", err)
	}

	//if _, err := file.Seek(audioChunk.ByteOffset, 0); err != nil {
	//	return nil, err
	//}
	//p, err := file.Read(audioBuf)
	//if err != nil {
	//	return nil, status.Errorf(codes.Internal, "File read failed: %v", err)
	//}

	return &pb.VideoChunk{
		Data:      append(moofBuf[:n], videoBuf[:o]...), // audioBuf[:p]...),
		StartTime: videoChunk.StartTime,
		EndTime:   videoChunk.EndTime,
	}, nil
}

func SeekChunk(metadata *MediaMetadata, timestamp float64) (int, error) {
	for index, chunk := range metadata.MoofChunks {
		if timestamp >= chunk.Chunks[metadata.VideoTrack.TrackId].StartTime && timestamp < chunk.Chunks[metadata.VideoTrack.TrackId].EndTime {
			return index, nil
		}
	}
	return 0, status.Errorf(codes.NotFound, "Segment not found for this timestamp %f", timestamp)
}
