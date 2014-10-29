package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"regexp"
	"text/template"
	"time"

	"github.com/brunetto/goutils/debug"
	"github.com/brunetto/goutils/readfile"
)

func main() {
	defer debug.TimeMe(time.Now())

	var (
		urlTemplate *template.Template
		err         error
		inFileName  string
		inFile      *os.File
		nReader     *bufio.Reader
		nLines      int = 0
		line        string
		nProcs      int    = 4
		dataChan           = make(chan Data, nProcs)
		frameChan          = make(chan Frame, nProcs)
		done               = make(chan struct{})
		regString   string = `^(-*\d+\.*\d*\w*[-\+]{0,1}\d*),\s*` + // 1: Lat
			`(-*\d+\.*\d*\w*[-\+]{0,1}\d*),\s*` + // 2: Long
			`(-*\d+\.*\d*\w*[-\+]{0,1}\d*),\s*` + // 3: Size
			`(-*\d+\.*\d*\w*[-\+]{0,1}\d*),\s*` + // 4: FOV
			`(-*\d+\.*\d*\w*[-\+]{0,1}\d*),\s*` + // 5: Head
			`(-*\d+\.*\d*\w*[-\+]{0,1}\d*)\s*` // 6: Pitch
		regExp *regexp.Regexp = regexp.MustCompile(regString)
		regRes []string
	)

	if len(os.Args) < 2 {
		log.Fatal(`Provide a file with a list of 
		lat, long, size, FOV, heading, pitch
		
like:
		
		40.721184,-69.988354, 400, 90, 90, 0`)
	}

	inFileName = os.Args[1]

	// Open infile for reading
	if inFile, err = os.Open(inFileName); err != nil {
		log.Fatal(err)
	}
	defer inFile.Close()
	nReader = bufio.NewReader(inFile)

	// Create the url template to download the images
	urlTemplate = template.New("urlTemplate")
	if urlTemplate, err = urlTemplate.Parse(UrlTemplate); err != nil {
		log.Fatal("Error trying to parse url template")
	}

	log.Printf("Start %v goroutines to download images and the collector\n", nProcs)
	for idx := 0; idx < nProcs; idx++ {
		go ImgDownloader(urlTemplate, dataChan, frameChan, done)
	}
	go ImgCollector(frameChan, done)

	// Scan lines
	for nLines = 0; ; nLines++ {
		if line, err = readfile.Readln(nReader); err != nil {
			if err.Error() != "EOF" {
				log.Fatal("Done reading with err", err)
			} else {
				fmt.Printf("Parsed %v lines\n", nLines)
				log.Println("Found end of file.")
			}
			break
		}

		// Read data and send to goroutines
		if regRes = regExp.FindStringSubmatch(line); regRes == nil {
			log.Fatal("Can't regexp ", line)
		}

		dataChan <- Data{Id: nLines,
			Lat:   regRes[1],
			Long:  regRes[2],
			Size:  regRes[3],
			FOV:   regRes[4],
			Head:  regRes[5],
			Pitch: regRes[6],
		}
	}

	log.Println("Close downloaders channel and clean downloaders goroutines")
	close(dataChan)
	for idx := 0; idx < nProcs; idx++ {
		<-done
	}

	log.Println("Close collector channel and clean collector goroutine")
	close(frameChan)
	<-done
}

func ImgDownloader(urlTemplate *template.Template, dataChan chan Data, frameChan chan Frame, done chan struct{}) {
	// Send end signal
	defer func() {
		done <- struct{}{}
	}()

	var (
		// 		outFileName string
		// 		outFile     *os.File
		data     Data
		err      error
		url      bytes.Buffer
		response *http.Response
		img      image.Image
		tmpImg   image.Image
	)

	// Download images and send data to the collector
	for data = range dataChan {
		// 		outFileName = strconv.Itoa(data.Id) + ".jpg"
		err = urlTemplate.Execute(&url, data)
		if err != nil {
			log.Fatal("Error trying to execute mail template")
		}

		// Download data
		if response, err = http.Get(url.String()); err != nil {
			log.Fatalf("Error while downloading %v: %v\n", url.String(), err)
		}
		defer response.Body.Close()

		// Create local file to copy to
		// 		if outFile, err = os.Create(outFileName); err != nil {
		// 			log.Fatalf("Error while creating %v: %v\n", outFileName, err)
		// 		}
		// 		defer outFile.Close()

		// GIF creation
		// Inspired by https://github.com/srinathh/goanigiffy/blob/master/goanigiffy.go
		// Decode jpeg, encode in gif, decode gif to be able to stack frames
		if img, err = jpeg.Decode(response.Body); err != nil {
			log.Fatal("Can't decode jpg image: ", err)
		}

		buf := bytes.Buffer{}
		if err := gif.Encode(&buf, img, nil); err != nil {
			log.Fatal("Can't gif-encode image: ", err)
		}

		if tmpImg, err = gif.Decode(&buf); err != nil {
			log.Fatal("Can't decode gif img: ", err)
		}

		frameChan <- Frame{Id: data.Id,
			Img: tmpImg.(*image.Paletted),
		}

		// Copy data to file
		// 		if _, err = io.Copy(outFile, response.Body); err != nil {
		// 			log.Fatalf("Error while filling %v: %v\n ", outFileName, err)
		// 		}
	}
}

func ImgCollector(frameChan chan Frame, done chan struct{}) {
	// Send end signal
	defer func() {
		done <- struct{}{}
	}()
	var (
		outFileName string = "final-hyperlapse.gif"
		outFile     *os.File
		frames      = map[int]*image.Paletted{}
		framesList  = []*image.Paletted{}
		frame       Frame
		delays      []int
		delay       int = 3
		err         error
	)

	// Collect frames
	for frame = range frameChan {
		frames[frame.Id] = frame.Img
	}

	// Create sorted frame list
	for idx := 0; idx < len(frames); idx++ {
		framesList = append(framesList, frames[idx])
	}

	// Create delays list
	delays = make([]int, len(frames))
	for idx, _ := range delays {
		delays[idx] = delay
	}

	log.Println("Create outfile ", outFileName)
	if outFile, err = os.Create(outFileName); err != nil {
		log.Fatalf("Error creating the destination file %s: %s\n", outFile, err)
	}
	defer outFile.Close()

	log.Println("Encode all frames in the final gif image")
	if err = gif.EncodeAll(outFile, &gif.GIF{framesList, delays, 0}); err != nil {
		log.Fatalf("Error encoding output into animated gif :%s\n", err)
	}
}

var UrlTemplate string = "https://maps.googleapis.com/maps/api/streetview?" +
	"size={{.Size}}x{{.Size}}&" +
	"location={{.Lat}},{{.Long}}&" +
	"fov={{.FOV}}&" +
	"heading={{.Head}}&" +
	"pitch={{.Pitch}}"

// Images data (for the downloaders)
type Data struct {
	Id    int
	Lat   string
	Long  string
	Size  string
	FOV   string
	Head  string
	Pitch string
}

type Frame struct {
	Id  int
	Img *image.Paletted
}
