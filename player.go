package main

import (
	// "bytes"

	"io"
	"log"
	"unsafe"

	"os"
	"runtime"
	"time"

	"github.com/bogem/id3v2/v2"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto/v2"
)

func main() {
	file := os.Args[1]
	tag, errid3 := id3v2.Open(file, id3v2.Options{Parse: true})
	if errid3 != nil {
		log.Fatalln("File not found or error in metadata extraction")
	}

	gtk.Init(nil)

	//новое окно
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatal("Unable to create window:", err)
	}
	win.SetTitle("MP3 Player")
	win.Connect("destroy", func() {
		//при событии закрытия окна отключаемся от gtk
		gtk.MainQuit()
	})

	ui(win, tag)

	win.SetDefaultSize(400, 300)
	win.ShowAll()
	//отображение окна и запуск цикла событий
	log.Println("Start")
	go initOto(file)
	gtk.Main()
}

func ui(win *gtk.Window, tag *id3v2.Tag) {
	layout, _ := gtk.GridNew()
	title, _ := gtk.LabelNew(tag.Title())
	layout.Attach(title, 0, 0, 1, 1) //верхний ряд

	button, _ := gtk.ButtonNew()
	button.Connect("clicked", func() {
		if loaded {
			if playing {
				log.Println("Pause")
				player.Pause()
			} else {
				log.Println("Play")
				player.Play()
			}
		}
	})
	button_label, _ = gtk.LabelNew("Play")
	button.Add(button_label)
	//кнопка под меткой
	layout.AttachNextTo(button, title, gtk.POS_BOTTOM, 1, 1)

	eventbox, _ := gtk.EventBoxNew()
	eventbox.Connect("button-press-event", func(widget *gtk.EventBox, event *gdk.Event) {
		allocation := widget.GetAllocation()
		width := allocation.GetWidth()
		//получаем координату X из структуры gdk.ButtonEvent
		x := *(*float64)(unsafe.Pointer(event.Native() + 24))
		//получаем долю общей продолжительности
		fraction := float64(x) / float64(width)
		//определяем смещение (с округлением до 4)
		offset := int64(float64(decodedStream.Length()) * fraction)
		offset = offset / 4 * 4 //align to sample
		//перемещаем воспроизведение на смещение
		player.(io.Seeker).Seek(offset, io.SeekStart)
		log.Println("Seek to fraction ", fraction)
	})
	progress, _ = gtk.ProgressBarNew()
	eventbox.Add(progress)
	layout.AttachNextTo(eventbox, button, gtk.POS_BOTTOM, 1, 1)
	win.Add(layout)
}

func initOto(file string) {
	log.Println("Loading mp3 from " + file)
	//загружаем файл целиком (чтобы узнать длину)
	data, e1 := os.Open(file)
	if e1 != nil {
		log.Fatalln(e1.Error())
	}
	// bts := bytes.NewReader(data)
	decodedStream, _ = mp3.NewDecoder(data)
	otoCtx, readyChan, e3 := oto.NewContext(44100, 2, 2)
	if e3 != nil {
		log.Fatalln(e3.Error())
	}
	//ждем завершения инициализации
	<-readyChan
	player = otoCtx.NewPlayer(decodedStream)
	log.Println("Loaded")
	loaded = true
	lengthInBytes := decodedStream.Length()
	// lengthInMs := decodedStream.Length() / (int64)(decodedStream.SampleRate()*4/1000)
	// log.Println(lengthInMs)

	//сохраним объект для использования, пока окно открыто
	players := make([]oto.Player, 1, 1)
	players[0] = player
	runtime.KeepAlive(players)
	go func() {
		playing = player.IsPlaying()
		for {
			if playing {
				pos, _ := decodedStream.Seek(0, io.SeekCurrent)
				var fraction float64
				fraction = float64(pos) / float64(lengthInBytes)
				progress.SetFraction(fraction)
			}
			if player.IsPlaying() != playing {
				playing = player.IsPlaying()
				if playing {
					button_label.SetLabel("Pause")
				} else {
					button_label.SetLabel("Play")
				}
			}
			time.Sleep(16 * time.Millisecond) //60fps
		}
	}()
}

var button_label *gtk.Label
var player oto.Player
var playing bool
var loaded bool
var progress *gtk.ProgressBar
var length *int
var decodedStream *mp3.Decoder
