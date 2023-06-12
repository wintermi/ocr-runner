# OCR Runner
[![Go Workflow Status](https://github.com/wintermi/ocr-runner/workflows/Go/badge.svg)](https://github.com/wintermi/ocr-runner/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/wintermi/ocr-runner)](https://goreportcard.com/report/github.com/wintermi/ocr-runner)
[![license](https://img.shields.io/github/license/wintermi/ocr-runner)](https://github.com/wintermi/ocr-runner/blob/main/LICENSE)
[![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/wintermi/ocr-runner?include_prereleases)](https://github.com/wintermi/ocr-runner/releases)


## Description
A command line application designed to recursively walk through the input path submitting all image files for optical character recognition (OCR) via the Google Cloud Vision API, Outputting the OCR response to a single newline delimited JSON File.

```
USAGE:
    ocr-runner -i PATH -o FILE

ARGS:
  -full
        Output full details to JSON
  -i string
        Input Path  (Required)
  -o string
        Output File  (Required)
  -verbose
        Display verbose or debug detail
```


## License
**ocr-runner** is released under the [Apache License 2.0](https://github.com/wintermi/ocr-runner/blob/main/LICENSE) unless explicitly mentioned in the file header.
