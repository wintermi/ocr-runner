// Copyright 2021-2023, Matthew Winter
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	i32 "github.com/adam-lavrik/go-imath/i32"

	vision "cloud.google.com/go/vision/apiv1"
	visionpb "cloud.google.com/go/vision/v2/apiv1/visionpb"
)

type Annotation struct {
	BoundingPoly *visionpb.BoundingPoly `json:"bounding_poly"`
	BoundingBox  image.Rectangle        `json:"bounding_box"`
	Orientation  int                    `json:"orientation"`
	Confidence   float32                `json:"confidence"`
	Text         string                 `json:"text"`
}

type ImageInfo struct {
	Filename   string       `json:"filename"`
	Size       int64        `json:"size"`
	Text       string       `json:"text"`
	Paragraphs []Annotation `json:"paragraphs"`
	Words      []Annotation `json:"words"`
}

type ImageFiles struct {
	Images []ImageInfo
}

//---------------------------------------------------------------------------------------

// Walk the provided input path and populate a list of images in preparation for OCR
func (files *ImageFiles) PopulateImages(inputPath string) error {

	// Execute a GLOB to return all files matching the provided pattern
	matches, err := filepath.Glob(inputPath)
	if err != nil {
		return fmt.Errorf("Glob Failed: %w", err)
	}

	// Get lits of GLOBs to ignore
	ignoreThis := GetIgnoreList(ignoreFileName)

	// Load all matching files returned from the Glob
	for _, filename := range matches {

		if IsIgnorableFile(filename, ignoreThis) {
			continue
		}

		fileInfo, err := os.Stat(filename)
		if err != nil {
			return fmt.Errorf("Failed to get file info: %w", err)
		}

		// Skip Directories
		if fileInfo.IsDir() {
			continue
		}

		image := ImageInfo{
			Filename: filename,
			Size:     fileInfo.Size(),
		}

		files.Images = append(files.Images, image)
	}

	return nil
}

//---------------------------------------------------------------------------------------

// Iterate through the image file list and call the Vision API to detect the text
func (files *ImageFiles) DetectImageText(outputFile string, outputFull bool) error {

	// Create the output file
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("Failed to create output file: %w", err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)

	// Execute OCR using Vision API
	errorCount := 0
	for i := range files.Images {
		logger.Info().Msg("Image:")
		logger.Info().Str("Filename", files.Images[i].Filename).Msg(indent)
		logger.Info().Int64("Size", files.Images[i].Size).Msg(indent)

		err := files.Images[i].CallVisionAPI()
		if err != nil {
			logger.Error().Err(err).Msg("Vision API request failed")
			errorCount++
			continue
		}

		var jsonData []byte

		if outputFull {
			jsonData, err = files.Images[i].GetFullJSON()
			if err != nil {
				logger.Error().Err(err).Msg("Failed to marshal json data")
				errorCount++
				continue
			}
		} else {
			jsonData, err = files.Images[i].GetCompactJSON()
			if err != nil {
				logger.Error().Err(err).Msg("Failed to marshal json data")
				errorCount++
				continue
			}
		}

		// Write out the JSON
		_, err = w.Write(jsonData)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to write to output file")
			errorCount++
			continue
		}

		// Write out the newline
		_, err = w.WriteString("\n")
		if err != nil {
			logger.Error().Err(err).Msg("Failed to write to output file")
			errorCount++
			continue
		}

		w.Flush()
	}

	// Raise an Error if one of the Vision API requests failes
	if errorCount > 0 {
		return fmt.Errorf("One or more Vision API requests failed")
	}

	return nil
}

//---------------------------------------------------------------------------------------

func (info *ImageInfo) AddParagraph(paragraph Annotation) {
	info.Paragraphs = append(info.Paragraphs, paragraph)
}

//---------------------------------------------------------------------------------------

func (info *ImageInfo) AddWord(word Annotation) {
	info.Words = append(info.Words, word)
}

//---------------------------------------------------------------------------------------

// Call the Vision API and retrieve all words and the bounds of the words
func (info *ImageInfo) CallVisionAPI() error {
	ctx := context.Background()

	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	file, err := os.Open(info.Filename)
	if err != nil {
		return err
	}
	defer file.Close()

	image, err := vision.NewImageFromReader(file)
	if err != nil {
		return err
	}

	imageContext := &visionpb.ImageContext{
		LanguageHints: []string{"en-t-i0-handwrit"},
	}

	annotation, err := client.DetectDocumentText(ctx, image, imageContext)
	if err != nil || annotation == nil {
		return err
	}

	info.Text = annotation.Text
	logger.Debug().Str("Text", info.Text).Msg("... Annotation")
	//logger.Debug().Any("Annotation", annotation).Msg(indent)

	for _, page := range annotation.Pages {
		for _, block := range page.Blocks {
			for _, paragraph := range block.Paragraphs {
				words := make([]string, len(paragraph.Words))

				for w, word := range paragraph.Words {
					symbols := make([]string, len(word.Symbols))

					for s, symbol := range word.Symbols {
						symbols[s] = symbol.Text
					}

					newWord := Annotation{
						BoundingPoly: word.BoundingBox,
						Confidence:   word.Confidence,
						Text:         strings.Join(symbols, ""),
					}
					newWord.SetRectangle(word.BoundingBox)
					newWord.SetOrientation(word.BoundingBox)

					info.AddWord(newWord)

					words[w] = newWord.Text
				}

				newParagraph := Annotation{
					BoundingPoly: paragraph.BoundingBox,
					Confidence:   paragraph.Confidence,
					Text:         strings.Join(words, " "),
				}
				newParagraph.SetRectangle(paragraph.BoundingBox)
				newParagraph.SetOrientation(paragraph.BoundingBox)

				info.AddParagraph(newParagraph)

				logger.Debug().Str("Text", newParagraph.Text).Float32("Confidence", newParagraph.Confidence).Msg("... Paragraph")
			}
		}
	}

	return nil
}

//---------------------------------------------------------------------------------------

// Calculate the Min/Max Bounding Box as a Rectangle
func (box *Annotation) SetRectangle(boundingBox *visionpb.BoundingPoly) {
	var result image.Rectangle

	if boundingBox != nil {
		minX := i32.Maximal
		minY := i32.Maximal
		maxX := i32.Minimal
		maxY := i32.Minimal

		for _, vertex := range boundingBox.Vertices {
			minX = i32.Min(minX, vertex.X)
			minY = i32.Min(minY, vertex.Y)
			maxX = i32.Max(maxX, vertex.X)
			maxY = i32.Max(maxY, vertex.Y)
		}

		result.Min.X = int(minX)
		result.Min.Y = int(minY)
		result.Max.X = int(maxX)
		result.Max.Y = int(maxY)
	}

	box.BoundingBox = result
}

//---------------------------------------------------------------------------------------

// Calculate the Orientation of the Bounding Box
//
//	When the text is horizontal it might look like:
//
//	    0----1
//	    |    |
//	    3----2
//
//	When it's rotated 180 degrees around the top-left corner it becomes:
//
//	    2----3
//	    |    |
//	    1----0
//
//	Example BoundingBox: vertices:{x:2496  y:1632}  vertices:{x:2220  y:1659}  vertices:{x:2215  y:1601}  vertices:{x:2490  y:1574}
func (box *Annotation) SetOrientation(boundingBox *visionpb.BoundingPoly) {
	if boundingBox == nil || len(boundingBox.Vertices) < 3 {
		box.Orientation = 0
		return
	}

	// Calculate orientation by the relative position of the vertex 0 to that of vertex 2
	if boundingBox.Vertices[0].X > boundingBox.Vertices[2].X {
		if boundingBox.Vertices[0].Y > boundingBox.Vertices[2].Y {
			box.Orientation = 180
		} else {
			box.Orientation = 270
		}
	} else {
		if boundingBox.Vertices[0].Y > boundingBox.Vertices[2].Y {
			box.Orientation = 90
		} else {
			box.Orientation = 0
		}
	}
}

//---------------------------------------------------------------------------------------

func (info *ImageInfo) GetCompactJSON() ([]byte, error) {

	compact := make(map[string]interface{})
	compact["filename"] = info.Filename
	compact["size"] = info.Size
	compact["text"] = info.Text

	paragraphs := make([]map[string]interface{}, len(info.Paragraphs))
	compact["paragraphs"] = paragraphs

	for i := range info.Paragraphs {
		paragraphs[i] = make(map[string]interface{})
		paragraphs[i]["confidence"] = info.Paragraphs[i].Confidence
		paragraphs[i]["text"] = info.Paragraphs[i].Text
	}

	return json.Marshal(compact)
}

//---------------------------------------------------------------------------------------

func (info *ImageInfo) GetFullJSON() ([]byte, error) {
	return json.Marshal(info)
}

// Simply opens a file so handleIgnoreFile can do the work and make testing easier.
func GetIgnoreList(ignoreFileName string) []string {
	fileReader, err := os.Open(path.Join("./", ignoreFileName))
	if err != nil {
		// ignore if error is about the ignore file not existing
		if os.IsNotExist(err) {
			return []string{}
		}

		logger.Error().Err(err).Msg("couldn't Open ignore file.")
		return []string{}
	}

	defer fileReader.Close()

	return handleIgnoreFile(fileReader)
}

func handleIgnoreFile(file io.Reader) []string {
	fileContents, err := io.ReadAll(file)
	if err != nil {
		logger.Error().Err(err).Msg("couldn't Read ignore file.")
		return []string{}
	}
	lineBreakRegExp := regexp.MustCompile(`\r?\n`)
	globs := lineBreakRegExp.Split(string(fileContents), -1)

	cleanGlobs := make([]string, 0)

	for _, g := range globs {
		s := strings.TrimSpace(g)
		if len(s) > 0 {
			cleanGlobs = append(cleanGlobs, s)
		}
	}

	return cleanGlobs
}

func IsIgnorableFile(fileName string, ignoreList []string) bool {

	for _, glob := range ignoreList {

		matches, err := filepath.Match(glob, fileName)
		if err != nil {
			errMsg := fmt.Sprintf("Ecountered a malformed GLOB while checking if a file should be ignored. Offending glob: %s", glob)
			logger.Error().Err(err).Msg(errMsg)
			continue
		}

		if matches {
			return true
		}
	}

	return false
}
