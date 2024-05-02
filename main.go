package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

const (
	baseURL        = "https://tmate.cc/"
	downloadURL    = "https://tmate.cc/download"
	emptyOptionMsg = `Usage: tiktok-dl [OPTIONS]
tiktok-dl: error: You must provide at txt file for links video url & select option video type.

Options:
 -file
 -with-watermark
 -without-watermark-hd
`
)

var optWithWatermark = false
var optWithoutWatermarkHD = false
var fileLinks = ""

var (
	headers = map[string]string{
		"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:107.0) Gecko/20100101 Firefox/107.4",
		"Content-Type":              "application/x-www-form-urlencoded",
		"Origin":                    "https://tmate.cc",
		"Connection":                "keep-alive",
		"Referer":                   "https://tmate.cc/",
		"Cache-Control":             "max-age=0",
		"Upgrade-Insecure-Requests": "1",
	}
)

func main() {
	flag.BoolVar(&optWithWatermark, "with-watermark", false, "video with watermark")
	flag.BoolVar(&optWithoutWatermarkHD, "without-watermark-hd", false, "video without watermark HD")
	flag.StringVar(&fileLinks, "file", "", "file .txt list of video link (required)")
	flag.Parse()

	if len(os.Args) == 1 {
		fmt.Print(emptyOptionMsg)
		return
	}
	if fileLinks == "" {
		fmt.Print(emptyOptionMsg)
		return
	}

	cookieJar, _ := cookiejar.New(nil)
	client := &http.Client{
		Timeout: 5 * time.Minute,
		Jar:     cookieJar,
	}

	sources := readLinksFile(fileLinks)

	//get token dulu
	token, err := getToken(client)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	p := mpb.New(mpb.WithWidth(60), mpb.WithWaitGroup(&wg))
	_, _ = 100, len(sources)
	wg.Add(len(sources))
	for _, s := range sources {
		// Create dir
		outputPath := createDir(s)
		source := s
		name := strings.Replace(s, "https://www.tiktok.com/", "", 1)

		//bar for invalid link
		if !isValidLink(s) {
			bar := p.AddBar(100, //max 100%,
				mpb.PrependDecorators(decor.Name(s, decor.WC{C: decor.DSyncWidth | decor.DindentRight})),
				mpb.AppendDecorators(decor.OnComplete(decor.Percentage(), "invalid link")),
			)
			bar.SetCurrent(100)
			wg.Done()
			continue
		}

		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			bar := p.AddBar(100, //max 100%,
				mpb.PrependDecorators(
					decor.Name(name, decor.WC{C: decor.DSyncWidth | decor.DindentRight | decor.DextraSpace}),
					decor.OnComplete(decor.AverageETA(decor.ET_STYLE_GO), "0s"),
				),
				mpb.AppendDecorators(decor.OnComplete(decor.Percentage(), "done")),
			)
			go func() {
				defer wg.Done()
				exec(client, *token, source, outputPath, bar)
			}()
		} else {
			bar := p.AddBar(100, //max 100%,
				mpb.PrependDecorators(decor.Name(name, decor.WC{C: decor.DSyncWidth | decor.DindentRight})),
				mpb.AppendDecorators(decor.OnComplete(decor.Percentage(), "already exists")),
			)
			bar.SetCurrent(100)
			wg.Done()
		}
	}

	wg.Wait()
	p.Wait()
	log.Println("All video downloaded")
}

func exec(client *http.Client, token, source string, outputPath string, bar *mpb.Bar) {
	links, err := findDownloadLink(client, token, source)
	if err != nil {
		log.Fatal(err)
		return
	}

	idx := getIdxByOption()
	if len(links) > 0 {
		download(client, outputPath, links[idx], bar)
	} else {
		bar.SetCurrent(100)
		bar.Abort(false)
	}
}

func readLinksFile(fileLocation string) (result []string) {
	file, err := os.Open(fileLocation)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue // Skip empty lines
		}

		result = append(result, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error:", err)
		return
	}

	return
}

func getIdxByOption() int {
	if optWithWatermark {
		return 3
	} else {
		return 2
	}
}

func getToken(client *http.Client) (result *string, err error) {
	//-----Get token from main page
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Unexpected status code:", resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	strHtmlResBody := string(body)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(strHtmlResBody))
	if err != nil {
		log.Fatal("Error loading HTML:", err)
	}

	// Find the input element with the name="token" attribute
	input := doc.Find("input[name='token']")
	if input.Length() == 0 {
		fmt.Println("Input element not found")
		return
	}
	tokenValue, exists := input.Attr("value")
	if !exists {
		fmt.Println("Value attribute not found")
		return
	}

	return &tokenValue, nil
}

func findDownloadLink(client *http.Client, token, source string) (result []string, err error) {
	formData := url.Values{}
	formData.Set("url", source)
	formData.Set("token", token)
	payload := formData.Encode()
	req, err := http.NewRequest("POST", downloadURL, bytes.NewBufferString(payload))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Unexpected status code:", resp.StatusCode)
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	strHtmlResBody := string(body)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(strHtmlResBody))
	if err != nil {
		log.Fatal("Error loading HTML:", err)
	}

	links := doc.Find(".downtmate-right.is-desktop-only.right .abuttons a")
	links.Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			result = append(result, href)
		}
	})

	return
}

func createDir(originSource string) (outputPath string) {
	username := "tiktok-dl"
	re := regexp.MustCompile(`@([^/]+)`)
	match := re.FindStringSubmatch(originSource)
	if len(match) > 1 {
		username = match[1]
	}

	videoFileName := strings.Split(originSource, "/")

	if _, err := os.Stat(username); os.IsNotExist(err) {
		err := os.MkdirAll(username, 0755)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			return
		}
	}

	fileName := fmt.Sprintf("%s-%s.mp4", username, videoFileName[len(videoFileName)-1])
	outputPath = fmt.Sprintf("%s/%s", username, fileName)

	return
}

type progressWriter struct {
	total      int64
	downloaded int64
	name       string
	bar        *mpb.Bar
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.downloaded += int64(n)
	percentage := float64(pw.downloaded) / float64(pw.total) * 100
	pw.bar.SetCurrent(int64(percentage))
	return n, nil
}

func download(client *http.Client, outputPath, selectedlink string, bar *mpb.Bar) {
	resp, err := client.Get(selectedlink)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	outputFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Println(" > Error creating output file:", err)
		return
	}
	defer outputFile.Close()

	fileSize := resp.ContentLength
	_, err = io.Copy(io.MultiWriter(outputFile, &progressWriter{total: fileSize, name: outputPath, bar: bar}), resp.Body)
	if err != nil {
		fmt.Println(" > Error downloading file:", err)
		return
	}
}

func isValidLink(link string) bool {
	pattern := `^https://(?:www\.)?tiktok\.com/@[a-zA-Z0-9_]+/video/[0-9]+$`
	re := regexp.MustCompile(pattern)
	return re.MatchString(link)
}
