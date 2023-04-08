package src

import (
	"github.com/abema/go-mp4"
	"os"
)

type MediaChunk struct {
	TrackId    uint32
	StartTime  float64
	EndTime    float64
	ByteOffset int64
	Size       int64
}

type MediaTrack struct {
	TrackId   uint32
	Language  string
	Name      string
	Codec     string
	TimeScale uint32
}

type MoofChunks struct {
	Offset int64
	Size   int64
	Chunks map[uint32]*MediaChunk
}

type MediaMetadata struct {
	Size          uint64
	VideoDuration float64
	MoofChunks    []*MoofChunks
	VideoTrack    *MediaTrack
	AudioTracks   []*MediaTrack
	TextTracks    []*MediaTrack
}

func ReadMetadata(file *os.File) (*MediaMetadata, error) {
	moov, err := mp4.ExtractBox(file, nil, mp4.BoxPath{mp4.BoxTypeMoov()})
	if err != nil {
		return nil, err
	}
	mediaMetadata := MediaMetadata{
		Size:        moov[0].Offset + moov[0].Size,
		AudioTracks: make([]*MediaTrack, 0),
		TextTracks:  make([]*MediaTrack, 0),
	}
	tkhdBoxes, err := mp4.ExtractBoxWithPayload(file, nil, mp4.BoxPath{mp4.BoxTypeMoov(), mp4.BoxTypeTrak(), mp4.BoxTypeTkhd()})
	if err != nil {
		return nil, err
	}
	hdlrBoxes, err := mp4.ExtractBoxWithPayload(file, nil, mp4.BoxPath{mp4.BoxTypeMoov(), mp4.BoxTypeTrak(), mp4.BoxTypeMdia(), mp4.BoxTypeHdlr()})
	if err != nil {
		return nil, err
	}
	mdhdBoxes, err := mp4.ExtractBoxWithPayload(file, nil, mp4.BoxPath{mp4.BoxTypeMoov(), mp4.BoxTypeTrak(), mp4.BoxTypeMdia(), mp4.BoxTypeMdhd()})
	if err != nil {
		return nil, err
	}

	for i := range hdlrBoxes {
		hldr := hdlrBoxes[i].Payload.(*mp4.Hdlr)
		mdhd := mdhdBoxes[i].Payload.(*mp4.Mdhd)
		tkhd := tkhdBoxes[i].Payload.(*mp4.Tkhd)

		mediaTrack := MediaTrack{TrackId: tkhd.TrackID, Name: hldr.Name, Language: BytesToAsciiStr(mdhd.Language), TimeScale: mdhd.Timescale}
		if hldr.HandlerType == [4]byte{'s', 'o', 'u', 'n'} {
			mediaMetadata.AudioTracks = append(mediaMetadata.AudioTracks, &mediaTrack)
		} else if hldr.HandlerType == [4]byte{'s', 'b', 't', 'l'} {
			mediaMetadata.TextTracks = append(mediaMetadata.AudioTracks, &mediaTrack)
		} else if hldr.HandlerType == [4]byte{'v', 'i', 'd', 'e'} {
			mediaMetadata.VideoTrack = &mediaTrack
		}
	}
	mediaMetadata.MoofChunks = readMoofChunks(&mediaMetadata, file)
	return &mediaMetadata, nil
}

func readMoofChunks(metadata *MediaMetadata, file *os.File) []*MoofChunks {
	moofChunks := make([]*MoofChunks, 0)
	var videoDuration float64
	moofBoxes, _ := mp4.ExtractBoxWithPayload(file, nil, mp4.BoxPath{mp4.BoxTypeMoof()})
	mdatBoxes, _ := mp4.ExtractBoxWithPayload(file, nil, mp4.BoxPath{mp4.BoxTypeMdat()})
	duration := 0.0
	for index := range moofBoxes {
		moofBox := moofBoxes[index]
		mdatBox := mdatBoxes[index]
		//trafBoxes, _ := mp4.ExtractBoxWithPayload(file, &moofBox.Info, mp4.BoxPath{mp4.BoxTypeTraf()})
		moofChunk := MoofChunks{Chunks: make(map[uint32]*MediaChunk, 0), Offset: int64(moofBox.Info.Offset), Size: int64(moofBox.Info.Size)}
		moofChunks = append(moofChunks, &moofChunk)
		mediaChunk := MediaChunk{
			TrackId:    1,
			ByteOffset: int64(mdatBox.Info.Offset),
			Size:       int64(mdatBox.Info.Size),
			StartTime:  duration,
			EndTime:    duration + 1,
		}
		duration += 1
		moofChunk.Chunks[1] = &mediaChunk
		//for _, trafBox := range trafBoxes {
		//	chunkBoxes, _ := mp4.ExtractBoxesWithPayload(file, &trafBox.Info, []mp4.BoxPath{
		//		{mp4.BoxTypeTfhd()},
		//		{mp4.BoxTypeTrun()},
		//	})
		//	var tfhd *mp4.Tfhd
		//	var trun *mp4.Trun
		//	for _, bip := range chunkBoxes {
		//		switch bip.Info.Type {
		//		case mp4.BoxTypeTfhd():
		//			tfhd = bip.Payload.(*mp4.Tfhd)
		//		case mp4.BoxTypeTrun():
		//			trun = bip.Payload.(*mp4.Trun)
		//		}
		//	}
		//	var size uint32
		//	for _, entry := range trun.Entries {
		//		size += entry.SampleSize
		//	}
		//	mediaChunk := MediaChunk{
		//		TrackId:    tfhd.TrackID,
		//		ByteOffset: int64(moofBox.Info.Offset + uint64(trun.DataOffset)),
		//		Size:       int64(size),
		//	}
		//	if tfhd.TrackID == metadata.VideoTrack.TrackId {
		//		endTime := videoDuration + float64((tfhd.DefaultSampleDuration*trun.SampleCount)/metadata.VideoTrack.TimeScale)
		//		mediaChunk.StartTime = videoDuration
		//		mediaChunk.EndTime = endTime
		//		videoDuration = endTime
		//	}
		//	moofChunk.Chunks[tfhd.TrackID] = &mediaChunk
		//}
	}
	metadata.VideoDuration = videoDuration
	return moofChunks
}
