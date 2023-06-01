import {NgModule} from '@angular/core';
import {BrowserModule} from '@angular/platform-browser';

import {AppComponent} from './app.component';
import {VideoPlayerComponent} from './video-player/video-player.component';
import {VgCoreModule} from "@videogular/ngx-videogular/core";
import {VgControlsModule} from "@videogular/ngx-videogular/controls";
import {VgOverlayPlayModule} from "@videogular/ngx-videogular/overlay-play";
import {VgBufferingModule} from "@videogular/ngx-videogular/buffering";
import {VgStreamingModule} from "@videogular/ngx-videogular/streaming";

@NgModule({
  declarations: [AppComponent, VideoPlayerComponent],
  imports: [
    BrowserModule,
    VgCoreModule,
    VgControlsModule,
    VgOverlayPlayModule,
    VgBufferingModule,
    VgStreamingModule,
  ],
  providers: [],
  exports: [VideoPlayerComponent],
  bootstrap: [AppComponent],
})
export class AppModule {}
