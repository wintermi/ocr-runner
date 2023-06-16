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
	"image"
	"os"
	"strings"

	vision "cloud.google.com/go/vision/apiv1"
	visionpb "cloud.google.com/go/vision/v2/apiv1/visionpb"
	"github.com/adam-lavrik/go-imath/i32"
)

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

	for _, page := range annotation.Pages {
		for _, block := range page.Blocks {
			for _, paragraph := range block.Paragraphs {
				words := make([]string, len(paragraph.Words))

				for w, word := range paragraph.Words {
					symbols := make([]string, len(word.Symbols))

					for s, symbol := range word.Symbols {
						symbols[s] = symbol.Text
					}

					words[w] = strings.Join(symbols, "")
				}

				textBlock := TextBlock{
					BoundingBox: GetBoundingBox(paragraph.BoundingBox),
					Confidence:  paragraph.Confidence,
					Orientation: GetOrientation(paragraph.BoundingBox),
					Text:        strings.Join(words, " "),
				}

				info.AddParagraph(textBlock)

				logger.Debug().Str("Text", textBlock.Text).Float32("Confidence", textBlock.Confidence).Msg("... Paragraph")
			}
		}
	}

	return nil
}

//---------------------------------------------------------------------------------------

// Calculate the Min/Max Bounding Box as a Rectangle
func GetBoundingBox(boundingBox *visionpb.BoundingPoly) image.Rectangle {
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
func GetOrientation(boundingBox *visionpb.BoundingPoly) int {
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
