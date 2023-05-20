package gen

import (
	"encoding/base64"
	"fmt"
	"github.com/jay9909/alphavantage/cmd/apigen/api"
	"go/format"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"io"
	"os"
	"strings"
	"time"
)

const generatedFileName = "api_generated.go"
const documentationPage = "https://www.alphavantage.co/documentation/"

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

	// The Checksum line that we are looking for should be 58 bytes including the trailing newline
	const checksumLength = 60
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

	checksumPos := strings.Index(checksumLine, "// Checksum: ")
	checksumStr := checksumLine[checksumPos+len(checksumTag):]

	checksumBytes, err := base64.StdEncoding.DecodeString(checksumStr)
	if err != nil {
		return previousChecksum, fmt.Errorf("error decoding previous checksum: %w", err)
	}

	previousChecksum = [32]byte(checksumBytes)
	return previousChecksum, nil
}

func GenerateApi(endpoints api.Endpoints, accessRecord api.AccessRecord) error {
	f, err := os.Create(generatedFileName)
	if err != nil {
		panic(err)
	}

	err = writeHeader(f, accessRecord)
	if err != nil {
		panic(fmt.Errorf("could not write file header to file: %w", err))
	}

	categories := maps.Keys(endpoints)
	slices.SortFunc(categories, func(cat1, cat2 api.Category) bool {
		return cat1.LinkName < cat2.LinkName
	})
	for _, category := range categories {
		err = writeCategory(f, category)
		if err != nil {
			panic(fmt.Errorf("could not write category header for %v: %w", category.ReadableName, err))
		}

		endpointList := endpoints[category]
		for _, endpoint := range endpointList {
			err = writeEndpoint(f, endpoint)
			if err != nil {
				panic(fmt.Errorf("could not write endpoint %v function to file: %w",
					endpoint.Function, err))
			}
		}
	}

	err = writeChecksum(f, accessRecord)
	if err != nil {
		panic(fmt.Errorf("could not write file header to file: %w", err))
	}

	err = f.Close()
	if err != nil {
		fmt.Printf("Error closing generated file: %v", err)
	}

	// Gofmt the file
	generatedContents, err := os.ReadFile(generatedFileName)
	if err != nil {
		panic(fmt.Errorf("error reading the generated file prior to formatting: %w", err))
	}

	formattedGeneratedContents, err := format.Source(generatedContents)
	if err != nil {
		panic(fmt.Errorf("error formatting the generated file: %w", err))
	}
	err = os.WriteFile(generatedFileName, formattedGeneratedContents, 666)
	if err != nil {
		panic(fmt.Errorf("error writing the go-fmt'ed generated code to disk: %w", err))
	}

	return nil
}

func writeHeader(f *os.File, accessRecord api.AccessRecord) error {
	headerParams := map[string]string{
		"Date": accessRecord.Date.Format(time.DateTime),
	}

	return fileHeaderTemplate.Execute(f, headerParams)
}

func writeCategory(f *os.File, category api.Category) error {
	categoryParams := map[string]string{
		"LinkName":     documentationPage + category.LinkName,
		"ReadableName": category.ReadableName,
		"Desc":         wrapInComments(category.Desc),
	}

	return categoryTemplate.Execute(f, categoryParams)
}

func writeEndpoint(f *os.File, endpoint api.Endpoint) error {
	// The function CURRENCY_EXCHANGE_RATE is in the documentation twice, once under Foreign Exchange and
	// once under Digital & Crypto Currencies.  The "function" and other parameters are identical.  Keep
	// the one under Foreign Exchange and skip the one under Digital & Crypto Currencies
	if endpoint.LinkName == "#crypto-exchange" {
		return nil
	}

	function := endpoint.Function
	funcLower := strings.ToLower(function)
	splitFunc := strings.Split(funcLower, "_")
	for i, word := range splitFunc {
		splitFunc[i] = strings.ToTitle(string(word[0])) + word[1:]
	}
	funcName := strings.Join(splitFunc, "")

	var docCommentBuilder strings.Builder

	if endpoint.Premium == true {
		docCommentBuilder.WriteString("// [PREMIUM] ")
	} else {
		docCommentBuilder.WriteString("// ")
	}

	// Add the name, endpoint description, and link to the doc comment.
	docCommentBuilder.WriteString(fmt.Sprintf("%v\n// %v\n// %v\n//\n",
		endpoint.ReadableName, wrapInComments(endpoint.Desc), documentationPage+endpoint.LinkName))

	// Now collect the parameters.  Add them to the doc comment and set up the function body params.
	docCommentBuilder.WriteString("// Parameters:")

	var argList []string // Argument list for the function signature
	var params []string  // Function body shuttling from arguments to parameter map

	for _, param := range endpoint.Params {
		if param.Name != "function" && param.Name != "apikey" {
			paramName := param.Name
			if param.Required != true {
				paramName = "opt_" + paramName
			}

			docCommentBuilder.WriteString(fmt.Sprintf("\n// -\t%v: %v",
				paramName, wrapInComments(param.Desc)))

			argList = append(argList, strings.ToLower(paramName))
			params = append(params, fmt.Sprintf("\t\t\"%v\": %v,",
				param.Name, strings.ToLower(paramName)))
		}
	}

	arguments := strings.Join(argList, ", ") + " string"

	endpointParams := map[string]string{
		"FuncName":         funcName,
		"EndpointFunction": function,
		"DocComment":       docCommentBuilder.String(),
		"ArgList":          arguments,
		"QueryParams":      strings.Join(params, "\n"),
	}

	return endpointTemplate.Execute(f, endpointParams)
}

func writeChecksum(f *os.File, accessRecord api.AccessRecord) error {
	checksumBytes := accessRecord.Checksum
	checksum := base64.StdEncoding.EncodeToString(checksumBytes[:])

	footerParams := map[string]string{
		"Checksum": checksum,
	}

	return checksumTemplate.Execute(f, footerParams)
}

func wrapInComments(description string) string {
	return strings.Join(strings.Split(description, "\n"), "\n// ")
}
