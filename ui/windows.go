// Copyright (c) 2016, 2017 Evgeny Badin

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// +build windows

package ui

import (
	"io"
	"log"
	"time"

	"github.com/korandiz/mpa"

	"github.com/koron/go-waveout"

	"github.com/budkin/jam/music"
)

func (app *App) player() {
	stop := make(chan bool)
	pause := make(chan bool)
	playing := false
	paused := false
	next := false
	prev := false
	pauseDur := time.Duration(0)

	stream, err := waveout.New(2, 44100, 16)
	if err != nil {
		log.Printf("device open failure: %v", err)
		return
	}
	defer stream.Close()

	//var d mpa.Decoder
	var r *mpa.Reader
	data := make([]byte, 1024*8)

	//var buff [2][]float32
	for {
		switch <-app.Status.State {
		case 0:
			if paused {
				pause <- true
			}
			if playing {
				stop <- true
			}

			album := app.Status.NumAlbum[true]
			ntrack := app.Status.NumTrack
			queueTemp := make([][]*music.BTrack, len(app.Status.Queue))
			copy(queueTemp, app.Status.Queue)

			track := queueTemp[album][ntrack]
			song, err := app.GMusic.GetStream(track.ID)
			if err != nil {
				log.Fatalf("Can't play stream: %s", err)
			}
			defDur = time.Duration(0)
			defTrack = &music.BTrack{}
			app.updateUI()

			//d = mpa.Decoder{Input: song.Body}
			r = &mpa.Reader{Decoder: &mpa.Decoder{Input: song.Body}}
			defer song.Body.Close()
			timer := time.Now()
			go func() {
				for {
					select {
					case <-pause:
						pauseDur = defDur
						paused = true
					loop:
						for {
							select {
							case <-stop:
								pauseDur = time.Duration(0)
								paused = false
								return
							case <-pause:
								timer = time.Now()
								paused = false
								break loop
							}
						}
					case <-stop:
						playing = false
						pauseDur = time.Duration(0)
						return
					default:
						defer func() {
							playing = false
							defDur = time.Duration(0)
							defTrack = &music.BTrack{}
							app.updateUI()
						}()
						playing = true

						defDur = time.Since(timer) + pauseDur
						defTrack = track
						app.printBar(defDur, defTrack)

						//buf := new(bytes.Buffer)

						i, err := r.Read(data)
						if err == io.EOF || i == 0 || next || prev {
							if next {
								next = false
							}

							switch err.(type) {
							case mpa.MalformedStream:
								continue
							}
							if !prev {
								if ntrack < len(queueTemp[album])-1 {
									ntrack++
								} else if album < len(queueTemp)-1 {
									album++
									ntrack = 0
								} else {
									return
								}
							} else {
								if ntrack > 0 {
									ntrack--
								} else if album > 0 && len(queueTemp[album-1]) > 0 {
									album--
									ntrack = len(queueTemp[album]) - 1
								} else {
									ntrack = 0
								}
								prev = false
							}

							track = queueTemp[album][ntrack]
							song, err = app.GMusic.GetStream(track.ID)
							if err != nil {
								log.Fatalf("Can't get stream: %s", err)
							}
							//d = mpa.Decoder{Input: song.Body}
							r = &mpa.Reader{Decoder: &mpa.Decoder{Input: song.Body}}
							pauseDur = time.Duration(0)
							defDur = time.Duration(0)
							defTrack = &music.BTrack{}
							app.updateUI()

							timer = time.Now()
							continue
						}

						i, err = stream.Write(data)
						if err != nil {
							log.Fatalf("Can't write stream: %s", err)
						}

					}
				}
			}()
		case 1:
			if playing {
				stop <- true
			}
		case 2:
			if playing {
				pause <- true
			}
		case 3:
			if playing {
				next = true
			}
			if paused {
				pause <- true
				next = true
			}
		case 4:
			if playing {
				prev = true
			}
			if paused {
				pause <- true
				prev = true
			}
		}
	}
}
