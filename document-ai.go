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
	"context"
	"fmt"
	"image"
	"net/url"
	"os"
	"strings"

	documentai "cloud.google.com/go/documentai/apiv1"
	documentaipb "cloud.google.com/go/documentai/apiv1/documentaipb"
	"github.com/adam-lavrik/go-imath/i32"
	"google.golang.org/api/option"
)

//---------------------------------------------------------------------------------------

// Call the Document AI and retrieve all words and the bounds of the words
func (info *ImageInfo) CallDocumentAI(predictionEndpoint string) error {
	ctx := context.Background()

	// Parse the Document AI Parser Prediction Endpoint into a URL structure
	endpoint, err := url.ParseRequestURI(predictionEndpoint)
	if err != nil {
		return err
	}

	client, err := documentai.NewDocumentProcessorClient(ctx, option.WithEndpoint(GetHostName(endpoint)))
	if err != nil {
		return err
	}
	defer client.Close()

	// Open and read the image file
	image, err := os.ReadFile(info.Filename)
	if err != nil {
		return err
	}

	// Construct the Process Request Payload
	request := &documentaipb.ProcessRequest{
		Name: GetRequestName(endpoint),
		Source: &documentaipb.ProcessRequest_RawDocument{
			RawDocument: &documentaipb.RawDocument{
				Content:  image,
				MimeType: info.MimeType,
			},
		},
	}

	response, err := client.ProcessDocument(ctx, request)
	if err != nil || response == nil {
		return err
	}

	info.Text = response.Document.Text
	logger.Debug().Str("Text", info.Text).Msg("... Document")

	for _, page := range response.Document.Pages {
		for _, paragraph := range page.Paragraphs {
			textBlock := TextBlock{
				BoundingBox: GetLayoutBoundingBox(paragraph.Layout.BoundingPoly),
				Confidence:  paragraph.Layout.Confidence,
				Orientation: GetLayoutOrientation(paragraph.Layout.BoundingPoly),
				Text:        GetTextFromSegments(paragraph.Layout.TextAnchor.TextSegments, &info.Text),
			}

			info.AddParagraph(textBlock)

			logger.Debug().Str("Text", textBlock.Text).Float32("Confidence", textBlock.Confidence).Msg("... Paragraph")
		}
	}

	return nil
}

//---------------------------------------------------------------------------------------

// Construct the Host Name from the Document AI Prediction Endpoint URL
func GetHostName(endpoint *url.URL) string {

	host := endpoint.Host
	if strings.Index(host, ":") >= 0 {
		return host
	}

	return fmt.Sprintf("%s:443", host)
}

//---------------------------------------------------------------------------------------

// Construct the Request Name from the Document AI Prediction Endpoint URL
func GetRequestName(endpoint *url.URL) string {

	name := endpoint.Path
	if i := strings.Index(name, "/projects/"); i >= 0 {
		name = name[i+1:]
		if i := strings.Index(name, ":"); i >= 0 {
			name = name[:i]
		}
	}
	return name
}

//---------------------------------------------------------------------------------------

// Calculate the Min/Max Bounding Box as a Rectangle
func GetLayoutBoundingBox(boundingBox *documentaipb.BoundingPoly) image.Rectangle {
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

	return result
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
func GetLayoutOrientation(boundingBox *documentaipb.BoundingPoly) int {
	var result int

	if boundingBox == nil || len(boundingBox.Vertices) < 3 {
		return 0
	}

	// Calculate orientation by the relative position of the vertex 0 to that of vertex 2
	if boundingBox.Vertices[0].X > boundingBox.Vertices[2].X {
		if boundingBox.Vertices[0].Y > boundingBox.Vertices[2].Y {
			result = 180
		} else {
			result = 270
		}
	} else {
		if boundingBox.Vertices[0].Y > boundingBox.Vertices[2].Y {
			result = 90
		} else {
			result = 0
		}
	}

	return result
}

//---------------------------------------------------------------------------------------

// Calculate the Min/Max Bounding Box as a Rectangle
func GetTextFromSegments(textSegments []*documentaipb.Document_TextAnchor_TextSegment, documentText *string) string {
	var result string

	text := *documentText
	for _, textSegment := range textSegments {
		result += text[textSegment.StartIndex : textSegment.EndIndex-1]
	}

	return result
}
