package main

import (
	"github.com/gin-gonic/gin"
	"github.com/notedit/gstreamer-go"
	"log"
	"net/http"
)

var player = newPlayer()

func main() {
	r := setupGin()
	r.POST("/player/start", func(c *gin.Context) {
		file := c.Query("file")

		if err := player.Start(file); err != nil {
			log.Fatalf("failed to start player: %v", err)
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "playing",
		})
	})

	r.DELETE("/player/stop", func(c *gin.Context) {
		player.Stop()

		c.JSON(http.StatusOK, gin.H{
			"status": "stopped",
		})
	})

	r.GET("/stream", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "application/x-mpegURL")
		c.File("/tmp/hls/live.m3u8")
	})

	r.Run(":8080")
}

type Player struct {
	pipeline *gstreamer.Pipeline
}

func newPlayer() *Player {
	return &Player{}
}

func setupGin() *gin.Engine {
	r := gin.Default()
	return r
}

func (p *Player) startGstPipeline(file string) error {
	pipeline, err := gstreamer.New(
		"filesrc location=" + file + " " +
			"! decodebin name=decoder " +
			"! queue " +
			"! videoconvert " +
			//"! queue " +
			//"! x264enc preset=fast " +
			//"! mpegtsmux " +
			//"! queue " +
			//"! hlssink target-duration=5 max-files=5 location=tmp/index.m3u8 playlist-location=tmp/playlist.m3u8",
			"! autovideosink sync=0 name=video video. " +
			"! queue " +
			"! audioconvert " +
			"! autoaudiosink sync=0 name=audio",
	)
	if err != nil {
		return err
	}
	p.pipeline = pipeline
	return nil
}

func (p *Player) Start(file string) error {
	err := p.startGstPipeline(file)
	if err != nil {
		log.Println("Error setting up pipeline")
		return err
	}

	p.pipeline.Start()
	return nil
}

func (p *Player) Stop() {
	p.pipeline.SendEOS()
	p.pipeline.Stop()
}

func (p *Player) onMessage(msg *gst.Message) {
	log.Printf("Got message: %v", msg.GetName())
	switch msg.GetType() {
	case gst.MessageStateChanged:
		stateOld, stateNew, pending := msg.ParseStateChanged()
		log.Printf("State changed from %v to %v, pending: %v", stateOld, stateNew, pending)
	case gst.MessageError:
		log.Printf("Error: %v", msg)
	case gst.MessageEos:
		log.Printf("End of stream")
	}
}
