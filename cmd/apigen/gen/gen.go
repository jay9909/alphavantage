package gen

import (
	"alphavantage/cmd/apigen/api"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"time"
)

const generatedFileName = "api_generated.go"

func GetPreviousChecksum() ([32]byte, error) {
	// Find the last line in the generated file, which should look like this:
	// Checksum: CIktsNg2arwunITY/h7J5dfhhT+AqdwzqmWQEAtZLeI=\n

	var previousChecksum [32]byte

	fileInfo, err := os.Stat(generatedFileName)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist.  Return the zeroed out checksum, which won't match whatever it's
			// being compared to and force re-generation of doc page.
			return previousChecksum, nil
		} else {
			return previousChecksum, fmt.Errorf("could not access %v: %w", generatedFileName, err)
		}
	}

	// The Checksum line that we are looking for is ~58 bytes including the trailing newline
	const checksumLength = 58
	fileSize := fileInfo.Size()
	if fileSize < checksumLength {
		// If the file is less than this, it can't possibly contain the needed checksum line, let alone the
		// generated code.  Bail out.
		return previousChecksum, nil
	}

	file, err := os.Open(generatedFileName)
	if err != nil {
		return previousChecksum, fmt.Errorf("could not open the generated api file: %w", err)
	}

	fileEnd := make([]byte, checksumLength, checksumLength)
	_, err = file.ReadAt(fileEnd, fileSize-checksumLength)
	if err != nil && err != io.EOF {
		return previousChecksum, fmt.Errorf("error reading the end of the generated api file: %w", err)
	}

	checksumTag := "// Checksum: "
	checksumLine := strings.TrimSpace(string(fileEnd))

	strings.Index(checksumLine, "// Checksum: ")
	checksumStr := checksumLine[len(checksumTag):]

	checksumBytes, err := base64.StdEncoding.DecodeString(checksumStr)
	if err != nil {
		return previousChecksum, fmt.Errorf("error decoding previous checksum: %w", err)
	}

	previousChecksum = [32]byte(checksumBytes)
	return previousChecksum, nil
}

func GenerateApi(endpoints api.Endpoints, accessRecord api.AccessRecord) error {
	//templateBytes, err := os.ReadFile("cmd/apigen/template.go")
	//if err != nil {
	//	panic(fmt.Errorf("error opening the template: %W", err))
	//}
	//
	//// Extract templates from the file
	//templates := string(templateBytes)

	f, err := os.Create(generatedFileName)
	if err != nil {
		panic(err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}()

	return nil
}

func GenerateHead(f *os.File, templates string) error {
	// Get the generated file header template
	templateStart := "/* Start File Template */"
	templateEnd := "/* End File Template */"

	headerTemplateStart := strings.Index(templates, templateStart) + len(templateStart) + 1
	headerTemplateEnd := strings.Index(templates, templateEnd)
	headerTemplate := templates[headerTemplateStart:headerTemplateEnd]

	templateImpl, err := template.New("header_template").Parse(headerTemplate)
	if err != nil {
		return err
	}

	err = templateImpl.Execute(f, struct {
		DateTime time.Time
	}{
		DateTime: time.Now().UTC(),
	})
	return err
}
