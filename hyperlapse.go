package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
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
		dataChan           = make(chan Data, 1)
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

	urlTemplate = template.New("urlTemplate")
	if urlTemplate, err = urlTemplate.Parse(UrlTemplate); err != nil {
		log.Fatal("error trying to parse url template")
	}

	log.Printf("Starting % goroutines to download images\n", nProcs)
	for idx := 0; idx < nProcs; idx++ {
		go ImgDownloader(urlTemplate, dataChan, done)
	}

	// Scan lines
	for {
		if line, err = readfile.Readln(nReader); err != nil {
			if err.Error() != "EOF" {
				log.Fatal("Done reading with err", err)
			} else {
				fmt.Printf("Parsed %v lines\n", nLines)
				log.Println("Found end of file.")
			}
			break
		}
		// Feedback on parsing
		nLines += 1
		// read data and send to goroutines
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

	close(dataChan)
	
	log.Println("Cleaning goroutines")
	for idx := 0; idx < nProcs; idx++ {
		<-done
	}

}

func ImgDownloader(urlTemplate *template.Template, dataChan chan Data, done chan struct{}) {
	// Send end signal
	defer func() {
		done <- struct{}{}
	}()

	var (
		outFileName string
		outFile     *os.File
		data        Data
		err         error
		url         bytes.Buffer
		response    *http.Response
	)

	// Write to file
	for data = range dataChan {
		outFileName = strconv.Itoa(data.Id) + ".jpg"
		err = urlTemplate.Execute(&url, data)
		if err != nil {
			log.Fatal("error trying to execute mail template")
		}

		// Download data
		if response, err = http.Get(url.String()); err != nil {
			log.Fatalf("Error while downloading %v: %v\n", url.String(), err)
		}
		defer response.Body.Close()

		// Create local file to copy to
		if outFile, err = os.Create(outFileName); err != nil {
			log.Fatalf("Error while creating %v: %v\n", outFileName, err)
		}
		defer outFile.Close()

		// Copy data to file
		if _, err = io.Copy(outFile, response.Body); err != nil {
			log.Fatalf("Error while filling %v: %v\n ", outFileName, err)
		}
	}

}

var UrlTemplate string = "https://maps.googleapis.com/maps/api/streetview?" +
	"size={{.Size}}x{{.Size}}&" +
	"location={{.Lat}},{{.Long}}&" +
	"fov={{.FOV}}&" +
	"heading={{.Head}}&" +
	"pitch={{.Pitch}}"

type Data struct {
	Id    int
	Lat   string
	Long  string
	Size  string
	FOV   string
	Head  string
	Pitch string
}
