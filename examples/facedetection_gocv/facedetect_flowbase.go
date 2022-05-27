package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/flowbase/flowbase"
	"gocv.io/x/gocv"
)

const (
	BUFSIZE = 16
)

func main() {
	runtime.GOMAXPROCS(3)

	if len(os.Args) < 3 {
		fmt.Println("How to run:\n\tfacedetect [camera ID] [classifier XML file]")
		return
	}
	// parse args
	deviceID, _ := strconv.Atoi(os.Args[1])
	xmlFile := os.Args[2]

	// Initiate network
	net := flowbase.NewNetwork()

	// Initiate components
	webcamReader := NewWebcamReader(deviceID)
	net.AddProcess(webcamReader)

	faceDetector := NewFaceDetector(xmlFile)
	net.AddProcess(faceDetector)

	fpsPrinter := NewFPSPrinter()
	net.AddProcess(fpsPrinter)

	windowDisplayer := NewWindowDisplayer()
	net.AddProcess(windowDisplayer)

	// Connect network
	webcamReader.OutImage = faceDetector.InImage
	faceDetector.OutImage = fpsPrinter.InImage
	fpsPrinter.OutImage = windowDisplayer.InImage

	// Run network
	net.Run()
}

// --------------------------------------------------------------------------------
// Webcam reader
// --------------------------------------------------------------------------------

type WebcamReader struct {
	OutImage chan *gocv.Mat
	deviceId int
}

func NewWebcamReader(deviceId int) *WebcamReader {
	return &WebcamReader{make(chan *gocv.Mat, BUFSIZE), deviceId}
}

func (p *WebcamReader) Run() {
	defer close(p.OutImage)
	webcam, err := gocv.VideoCaptureDevice(int(p.deviceId))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer webcam.Close()

	for {
		img := gocv.NewMat()
		if ok := webcam.Read(&img); !ok {
			fmt.Printf("Cannot read device %d\n", p.deviceId)
			return
		}
		if img.Empty() {
			continue
		}
		p.OutImage <- &img
	}
}

// --------------------------------------------------------------------------------
// Face detector
// --------------------------------------------------------------------------------

type FaceDetector struct {
	InImage  chan *gocv.Mat
	OutImage chan *gocv.Mat
	xmlFile  string
}

func NewFaceDetector(xmlFile string) *FaceDetector {
	return &FaceDetector{
		make(chan *gocv.Mat, BUFSIZE),
		make(chan *gocv.Mat, BUFSIZE),
		xmlFile,
	}
}

func (p *FaceDetector) Run() {
	defer close(p.OutImage)

	// color for the rect when faces detected
	blue := color.RGBA{0, 0, 255, 0}

	// load classifier to recognize faces
	classifier := gocv.NewCascadeClassifier()
	defer classifier.Close()

	if !classifier.Load(p.xmlFile) {
		fmt.Printf("Error reading cascade file: %v\n", p.xmlFile)
		return
	}

	for img := range p.InImage {
		// detect faces
		rects := classifier.DetectMultiScale(*img)
		fmt.Printf("found %d faces\n", len(rects))

		// draw a rectangle around each face on the original image,
		// along with text identifying as "Human"
		for _, r := range rects {
			gocv.Rectangle(img, r, blue, 3)
			size := gocv.GetTextSize("Human", gocv.FontHersheyPlain, 1.2, 2)
			pt := image.Pt(r.Min.X+(r.Min.X/2)-(size.X/2), r.Min.Y-2)
			gocv.PutText(img, "Human", pt, gocv.FontHersheyPlain, 1.2, blue, 2)
		}

		p.OutImage <- img
	}
}

// --------------------------------------------------------------------------------
// FPS printer
// --------------------------------------------------------------------------------

type FPSPrinter struct {
	InImage  chan *gocv.Mat
	OutImage chan *gocv.Mat
}

func NewFPSPrinter() *FPSPrinter {
	return &FPSPrinter{
		make(chan *gocv.Mat, BUFSIZE),
		make(chan *gocv.Mat, BUFSIZE),
	}
}

func (p *FPSPrinter) Run() {
	defer close(p.OutImage)

	red := color.RGBA{255, 0, 0, 0}
	origo := image.Pt(40, 60)
	start := time.Now()
	frames := 0

	for img := range p.InImage {
		// Calculate and print FPS in  image
		elapsed := time.Since(start)
		fps := float64(frames) / elapsed.Seconds()
		fpsText := fmt.Sprintf("%3.1f FPS", fps)
		gocv.PutText(img, fpsText, origo, gocv.FontHersheyPlain, 4, red, 2)
		p.OutImage <- img
		frames++
	}
}

// --------------------------------------------------------------------------------
// Window displayer
// --------------------------------------------------------------------------------

type WindowDisplayer struct {
	InImage chan *gocv.Mat
}

func NewWindowDisplayer() *WindowDisplayer {
	return &WindowDisplayer{make(chan *gocv.Mat, BUFSIZE)}
}

func (p *WindowDisplayer) Run() {
	window := gocv.NewWindow("Image output")
	defer window.Close()

	for img := range p.InImage {
		window.IMShow(*img)
		if window.WaitKey(1) >= 0 {
			break
		}
		img.Close()
	}
}
