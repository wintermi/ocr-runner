# OCR Runner

[![Workflows](https://github.com/wintermi/ocr-runner/workflows/Go/badge.svg)](https://github.com/wintermi/ocr-runner/actions/workflows/go.yml)
[![Go Report](https://goreportcard.com/badge/github.com/wintermi/ocr-runner)](https://goreportcard.com/report/github.com/wintermi/ocr-runner)
[![License](https://img.shields.io/github/license/wintermi/ocr-runner)](https://github.com/wintermi/ocr-runner/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/wintermi/ocr-runner?include_prereleases)](https://github.com/wintermi/ocr-runner/releases)


## Description

A command line application designed to recursively walk through the input path submitting all image files for optical character recognition (OCR) via either the Google Cloud Vision API or a Google Cloud Document AI processor if a prediction endpoint is provided.  The application will then output the image information and annotations to a single newline delimited JSON File.

```
USAGE:
    ocr-runner -i PATH -o FILE

ARGS:
  -endpoint string
        Document AI Prediction Endpoint  (Optional)
  -full
        Output full details to JSON
  -i string
        Input Path  (Required)
  -o string
        Output File  (Required)
  -verbose
        Display verbose or debug detail
```

## Valid File Extensions

The application will automatically filter out all files that do not have one of the following extensions:

- `.bmp`
- `.gif`
- `.jpg`
- `.jpeg`
- `.pdf`
- `.png`
- `.tif`
- `.tiff`
- `.webp`

## License

**ocr-runner** is released under the [Apache License 2.0](https://github.com/wintermi/ocr-runner/blob/main/LICENSE) unless explicitly mentioned in the file header.
