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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

var logger zerolog.Logger
var applicationText = "%s 0.2.0%s"
var copyrightText = "Copyright 2022-2023, Matthew Winter\n"
var indent = "..."

var helpText = `
A command line application designed to recursively walk through the input path
submitting all image files for optical character recognition (OCR) via either
the Google Cloud Vision API or a Google Cloud Document AI processor if a
prediction endpoint is provided.  The application will then output the image
information and annotations to a single newline delimited JSON File.

Use --help for more details.


USAGE:
    ocr-runner -i PATH -o FILE

ARGS:
`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, applicationText, filepath.Base(os.Args[0]), "\n")
		fmt.Fprint(os.Stderr, copyrightText)
		fmt.Fprint(os.Stderr, helpText)
		flag.PrintDefaults()
	}

	// Define the Long CLI flag names
	var inputPath = flag.String("i", "", "Input Path  (Required)")
	var outputFile = flag.String("o", "", "Output File  (Required)")
	var outputFull = flag.Bool("full", false, "Output full details to JSON")
	var predictionEndpoint = flag.String("endpoint", "", "Document AI Prediction Endpoint  (Optional)")
	var verbose = flag.Bool("verbose", false, "Display verbose or debug detail")

	// Parse the flags
	flag.Parse()

	// Validate the Required Flags
	if *inputPath == "" || *outputFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Setup Zero Log for Consolo Output
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	logger = zerolog.New(output).With().Timestamp().Logger()
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05.000"
	zerolog.DurationFieldUnit = time.Millisecond
	zerolog.DurationFieldInteger = true
	if *verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Output Header
	logger.Info().Msgf(applicationText, filepath.Base(os.Args[0]), "")
	logger.Info().Msg("Arguments")
	logger.Info().Str("Input Path", *inputPath).Msg(indent)
	logger.Info().Str("Output File", *outputFile).Msg(indent)
	logger.Info().Bool("Output Full Details", *outputFull).Msg(indent)
	logger.Info().Str("Document AI Prediction Endpoint", *predictionEndpoint).Msg(indent)
	logger.Info().Msg("Begin")

	// Walk the provided input path and populate a list of images in preparation for OCR
	var imageFiles ImageFiles
	err := imageFiles.PopulateImages(*inputPath)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to populate images list from provided input path")
		os.Exit(1)
	}

	// Check that we did find images to process
	if len(imageFiles.Images) == 0 {
		logger.Error().Msg("No image files found, check the input path provided")
		os.Exit(1)
	}
	logger.Info().Int("Image Count", len(imageFiles.Images)).Msg("Populating image file list complete")

	// Iterate through the image file list and call the Vision API to detect the text
	// Writing out the image information and annotations in JSON format to a file
	err = imageFiles.DetectImageText(*outputFile, *outputFull, *predictionEndpoint)
	if err != nil {
		logger.Error().Err(err).Msg("Image text detection failed")
		os.Exit(1)
	}
	logger.Info().Msg("End")
}
