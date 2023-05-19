import {Component, OnInit} from '@angular/core';
import {BitrateOptions} from "@videogular/ngx-videogular/core";

@Component({
  selector: 'app-video-player',
  templateUrl: './video-player.component.html',
  styleUrls: ['./video-player.component.less']
})
export class VideoPlayerComponent implements OnInit {
  bitrates: BitrateOptions[] | undefined;

  audioList: string[] = [
    "http://localhost:5000/file.mkv/audio_1.m3u8",
    "http://localhost:5000/file.mkv/audio_2.m3u8",
  ];

  currentAudio = "";

  ngOnInit(): void {
    this.bitrates = [
      {
        qualityIndex: 0,
        width: 0,
        height: 0,
        bitrate: 0,
        mediaType: "audio",
        label: "FR",
      },
      {
        qualityIndex: 1,
        width: 0,
        height: 0,
        bitrate: 0,
        mediaType: "audio",
        label: "EN",
      },
    ];
    this.currentAudio = this.audioList[0];
  }

  onGetBitrate(event: any) {
    console.log(event);
    this.bitrates = event;
  }

  onSelectedAudio(event: BitrateOptions) {
    console.log(event);
    this.currentAudio= this.audioList[event.qualityIndex];
  }

}
