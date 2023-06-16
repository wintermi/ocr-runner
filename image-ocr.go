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
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
)

type TextBlock struct {
	BoundingBox image.Rectangle `json:"bounding_box"`
	Orientation int             `json:"orientation"`
	Confidence  float32         `json:"confidence"`
	Text        string          `json:"text"`
}

type ImageInfo struct {
	Filename   string      `json:"filename"`
	Size       int64       `json:"size"`
	MimeType   string      `json:"mime_type"`
	Text       string      `json:"text"`
	Paragraphs []TextBlock `json:"paragraphs"`
}

type ImageFiles struct {
	Images []ImageInfo
}

var mimeTypes = map[string]string{
	".bmp":  "image/bmp",
	".gif":  "image/gif",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".pdf":  "application/pdf",
	".png":  "image/png",
	".tif":  "image/tiff",
	".tiff": "image/tiff",
	".webp": "image/webp",
}

//---------------------------------------------------------------------------------------

// Walk the provided input path and populate a list of images in preparation for OCR
func (files *ImageFiles) PopulateImages(inputPath string) error {

	// Execute a GLOB to return all files matching the provided pattern
	matches, err := filepath.Glob(inputPath)
	if err != nil {
		return fmt.Errorf("Glob Failed: %w", err)
	}

	// Load all matching files returned from the Glob
	for _, filename := range matches {
		fileInfo, err := os.Stat(filename)
		if err != nil {
			return fmt.Errorf("Failed to get file info: %w", err)
		}
		mimeType := mimeTypes[filepath.Ext(filename)]

		// Skip Directories and invalid File Extensions
		if fileInfo.IsDir() || len(mimeType) == 0 {
			continue
		}

		image := ImageInfo{
			Filename: filename,
			Size:     fileInfo.Size(),
			MimeType: mimeType,
		}

		files.Images = append(files.Images, image)
	}

	return nil
}

//---------------------------------------------------------------------------------------

// Iterate through the image file list and call the Vision API to detect the text
func (files *ImageFiles) DetectImageText(outputFile string, outputFull bool, predictionEndpoint string) error {

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
		logger.Info().Str("MimeType", files.Images[i].MimeType).Msg(indent)

		// Call the Vision API if no Document AI Parser Prediction Endpoint is provided
		if len(predictionEndpoint) == 0 {
			err := files.Images[i].CallVisionAPI()
			if err != nil {
				logger.Error().Err(err).Msg("Vision API request failed")
				errorCount++
				continue
			}
		} else {
			err := files.Images[i].CallDocumentAI(predictionEndpoint)
			if err != nil {
				logger.Error().Err(err).Msg("Document AI Parser request failed")
				errorCount++
				continue
			}
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

	// Raise an Error if one of the OCR requests failes
	if errorCount > 0 {
		return fmt.Errorf("One or more OCR request failed")
	}

	return nil
}

//---------------------------------------------------------------------------------------

func (info *ImageInfo) AddParagraph(paragraph TextBlock) {
	info.Paragraphs = append(info.Paragraphs, paragraph)
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
