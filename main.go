package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/notedit/gst" // Wrapper plus récent de GStreamer
)

type VideoPlayer struct {
	pipeline       *gst.Pipeline
	audioSelector  *gst.Element
	subtitleSelect *gst.Element
	position       int64
}

// LoadFile Fonction pour charger un nouveau fichier video.
func (vp *VideoPlayer) LoadFile(c *gin.Context) {
	filePath := c.Query("filePath")

	// Crée le pipeline GStreamer et ajoute les éléments nécessaires.
	pipeline, err := gst.PipelineNew("video-player")
	if err != nil {
		fmt.Println(err)
		return
	}
	vp.pipeline = pipeline

	source, err := gst.ElementFactoryMake("filesrc", "source")
	if err != nil {
		fmt.Println(err)
		return
	}
	// Définit le chemin du fichier à lire.
	source.SetObject("location", filePath)

	demuxer, err := gst.ElementFactoryMake("qtdemux", "demuxer")
	if err != nil {
		fmt.Println(err)
		return
	}

	videoDec, err := gst.ElementFactoryMake("h264parse", "video-decoder")
	if err != nil {
		fmt.Println(err)
		return
	}
	videoDec2, err := gst.ElementFactoryMake("avdec_h264", "decoder")
	if err != nil {
		fmt.Println(err)
		return
	}

	audioQueue, err := gst.ElementFactoryMake("queue", "audio-queue")
	if err != nil {
		fmt.Println(err)
		return
	}

	audioQueue.SetObject("max-size-buffers", 0)
	audioQueue.SetObject("max-size-bytes", 0)
	audioQueue.SetObject("max-size-time", 0)
	audioDec, err := gst.ElementFactoryMake("aacparse", "audio-decoder")
	if err != nil {
		fmt.Println(err)
		return
	}
	audioConv, err := gst.ElementFactoryMake("audioconvert", "audio-convert")
	if err != nil {
		fmt.Println(err)
		return
	}
	audioSink, err := gst.ElementFactoryMake("autoaudiosink", "audio-sink")
	if err != nil {
		fmt.Println(err)
		return
	}
	subtitleQueue, err := gst.ElementFactoryMake("queue", "subtitle-queue")
	if err != nil {
		fmt.Println(err)
		return
	}
	subtitleQueue.SetObject("max-size-buffers", 0)
	subtitleQueue.SetObject("max-size-bytes", 0)
	subtitleQueue.SetObject("max-size-time", 0)
	subtitleDec, err := gst.ElementFactoryMake("h264parse", "subtitle-decoder")
	if err != nil {
		fmt.Println(err)
		return
	}
	subtitleConv, err := gst.ElementFactoryMake("textoverlay", "subtitle-overlay")
	if err != nil {
		fmt.Println(err)
		return
	}

	vp.audioSelector = audioQueue
	vp.subtitleSelect = subtitleQueue

	// Crée les pads pour les pistes audio et les sous-titres.
	audioPad := demuxer.GetStaticPad("audio_0")
	audioPad.Link(audioQueue.GetStaticPad("sink"))
	audioQueue.GetStaticPad("src").Link(audioDec.GetStaticPad("sink"))
	audioDec.GetStaticPad("src").Link(audioConv.GetStaticPad("sink"))
	audioConv.GetStaticPad("src").Link(audioSink.GetStaticPad("sink"))

	videoPad := demuxer.GetStaticPad("video_0")
	videoPad.Link(videoDec.GetStaticPad("sink"))
	videoDec.GetStaticPad("src").Link(videoDec2.GetStaticPad("sink"))
	videoDec2.GetStaticPad("src").Link(vp.pipeline.GetStaticPad("sink"))

	subtitlePad := demuxer.GetStaticPad("subtitle_0")
	subtitlePad.Link(subtitleQueue.GetStaticPad("sink"))
	subtitleQueue.GetStaticPad("src").Link(subtitleDec.GetStaticPad("sink"))
	subtitleDec.GetStaticPad("src").Link(subtitleConv.GetStaticPad("sink"))
	subtitleConv.GetStaticPad("src").Link(vp.pipeline.GetStaticPad("video"))

	// Ajoute les éléments au pipeline GStreamer.
	vp.pipeline.Add(source)
	vp.pipeline.Add(demuxer)
	vp.pipeline.Add(videoDec)
	vp.pipeline.Add(videoDec2)
	vp.pipeline.Add(audioQueue)
	vp.pipeline.Add(audioDec)
	vp.pipeline.Add(audioConv)
	vp.pipeline.Add(audioSink)
	vp.pipeline.Add(subtitleQueue)
	vp.pipeline.Add(subtitleDec)
	vp.pipeline.Add(subtitleConv)

	// Lancement du pipeline GStreamer en pause.
	vp.pipeline.SetState(gst.StatePaused)
}

// Play Fonction pour lire la vidéo.
func (vp *VideoPlayer) Play(c *gin.Context) {
	vp.pipeline.SetState(gst.StatePlaying)
}

// Pause Fonction pour mettre en pause la vidéo.
func (vp *VideoPlayer) Pause(c *gin.Context) {
	vp.pipeline.SetState(gst.StatePaused)
}

// Stop Fonction pour arrêter la lecture et fermer le pipeline.
func (vp *VideoPlayer) Stop(c *gin.Context) {
	vp.pipeline.SetState(gst.StateNull)
}

// SetAudioTrack Fonction pour changer la piste audio.
func (vp *VideoPlayer) SetAudioTrack(c *gin.Context) {
	track := c.Query("track")
	pad := fmt.Sprintf("audio_%s", track)
	demuxer := vp.pipeline.GetByName("demuxer")
	vp.audioSelector.Unlink(vp.pipeline.GetByName("audio-output"))
	demuxer.LinkPads(pad, "audio-output")
	vp.audioSelector = vp.pipeline.GetByPName("audio-output").(*gst.Element)
}

// Fonction pour changer la piste de sous-titres.
func (vp *VideoPlayer) SetSubtitleTrack(c *gin.Context) {
	track := c.Query("track")
	pad := fmt.Sprintf("subtitle_%s", track)
	demuxer := vp.pipeline.GetByName("demuxer")
	vp.subtitleSelect.Unlink(vp.pipeline.GetByName("video"))
	demuxer.LinkPads(pad, "video")
	vp.subtitleSelect = vp.pipeline.GetByPName("video").(*gst.Element)
}

// Fonction pour obtenir la position actuelle de la lecture dans la vidéo.
func (vp *VideoPlayer) GetCurrentPosition(c *gin.Context) {
	bus := vp.pipeline.GetBus()
	msg := bus.Pull(gst.MessageProgress)
	if msg != nil {
		_, currPos, _ := msg.ParsePosition()
		vp.position = currPos
	}
	c.JSON(200, gin.H{
		"position": vp.position,
	})
}

func main() {
	r := gin.Default()
	v := VideoPlayer{}
	r.GET("/load", v.LoadFile)
	r.GET("/play", v.Play)
	r.GET("/pause", v.Pause)
	r.GET("/stop", v.Stop)
	r.GET("/audio", v.SetAudioTrack)
	r.GET("/subtitle", v.SetSubtitleTrack)
	r.GET("/position", v.GetCurrentPosition)
	r.Run(":8080")
}
