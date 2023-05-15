package parse

import (
	"alphavantage/cmd/apigen/api"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"html"
	"io"
	"net/http"
	"strings"
	"time"
)

const documentationUrl = "https://www.alphavantage.co/documentation/"

var NoChangeError = errors.New("no change since previous checksum")

func FindEndpoints(previousChecksum [32]byte) (api.Endpoints, api.AccessRecord, error) {
	var accessRecord api.AccessRecord

	fmt.Println("Fetching documentation page")
	resp, err := http.Get(documentationUrl)
	if err != nil {
		return nil, accessRecord, err
	}
	defer func() {
		closeErr := resp.Body.Close()
		if err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	// Get the response body into a byte array.  Checksum it. If it's the same as before, bail out.
	// If new, keep going.
	documentationPage, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Errorf("could not read documentation from response: %w", err))
	}

	// Cloudflare adds some single-use values to every page reload, so we need to scrape those out to check for changes
	// to the actual documentation.
	documentationPage = removeCloudflareStuff(documentationPage)

	currentChecksum := sha256.Sum256(documentationPage)
	if previousChecksum == currentChecksum {
		fmt.Printf("Checksums match!\n")
		return nil, accessRecord, NoChangeError
	} else {
		previousCheckStr := base64.StdEncoding.EncodeToString(previousChecksum[:])
		newCheckStr := base64.StdEncoding.EncodeToString(currentChecksum[:])

		fmt.Printf("Checksums do not match\nPrevious Checksum: %v\nNew Checksum: %v\n",
			previousCheckStr, newCheckStr)
	}

	fmt.Println("Parsing documentation page")
	root, err := goquery.NewDocumentFromReader(bytes.NewReader(documentationPage))
	if err != nil {
		return nil, accessRecord, err
	}

	endpoints := api.Endpoints{}

	fmt.Println("Building endpoint list")
	toc := root.Find("#table-of-contents")

	categoryLi := toc.Next()
	for categoryLi.Length() != 0 {
		// categoryLi looks like this:
		// 	<li><a href="#time-series-data">Core Stock APIs</a>

		a := categoryLi.Children().First() // get the <a>
		if a.Length() == 0 {
			return nil, accessRecord, fmt.Errorf("no child node under category LI: %#v", categoryLi)
		}
		if !a.Is("a") {
			return nil, accessRecord, fmt.Errorf("unexpected node type: %v", a)
		}
		linkName, isPresent := a.Attr("href")

		if !isPresent {
			return nil, accessRecord, fmt.Errorf("table of content link did not have href attribute: %v", a)
		}

		newCategory := api.Category{}
		newCategory.LinkName = linkName
		newCategory.ReadableName, newCategory.Desc, err = findCategoryDetails(linkName, root)
		if err != nil {
			return nil, accessRecord, fmt.Errorf("error extracting category details for %v: %w", linkName, err)
		}

		categoryEndpoints, err := findCategoryEndpoints(root, categoryLi)
		if err != nil {
			return nil, accessRecord, fmt.Errorf("error extracting endpoints for category %v: %w", linkName, err)
		}
		endpoints[newCategory] = categoryEndpoints

		fmt.Printf("Category done: %v\n", newCategory.ReadableName)
		categoryLi = categoryLi.Next()
	}

	accessRecord.Checksum = currentChecksum
	accessRecord.Date = time.Now()

	return endpoints, accessRecord, nil
}

func findCategoryDetails(linkName string, root *goquery.Document) (readableName, desc string, err error) {
	selector := "h2" + linkName
	categoryHead := root.Find(selector).First()

	if categoryHead.Length() != 1 {
		return "", "",
			fmt.Errorf("found unexpected DOM structure in categoryHead: %v", selector)
	}

	readableName = categoryHead.Text()

	// Move forward to the category description <p>
	categoryDescP := categoryHead.Next()
	desc = strings.TrimSpace(categoryDescP.Text())

	return readableName, desc, nil
}

func findCategoryEndpoints(root *goquery.Document, categoryLi *goquery.Selection) ([]api.Endpoint, error) {
	endpoints := make([]api.Endpoint, 0)

	endpointLinks := categoryLi.Children().Find("ul > li > a")
	endpointLinks.Each(func(i int, link *goquery.Selection) {
		endpoint := api.Endpoint{}
		// A link should look like this:
		// <a href="#fx-intraday">Intraday <span class="premium-label">Premium</span></a>

		////////////////////////////////////////////////////////////////////////////////
		// Get the contents of the link and determine if it's a Premium link or not
		// innerHtml, err := link.Html()
		linkName, isPresent := link.Attr("href")
		if !isPresent || linkName == "" {
			panic(fmt.Errorf("did not find an href in the %dth table of contents endpoint link in %v",
				i, categoryLi.Text()))
		}
		endpoint.LinkName = linkName

		////////////////////////////////////////////////////////////////////////////////
		// Now find it in the main page body and extract the good stuff
		endpointHead := root.Find("h4" + linkName)
		if endpointHead.Length() != 1 {
			linkHtml, _ := link.Html()
			panic(fmt.Errorf("found %d endpoint heads for link name %v: %v",
				endpointHead.Length(), linkName, linkHtml))
		}

		// An endpoint section looks like this:
		// <h4 id="company-overview">Company Overview</h4>
		// <p>This API returns the company information, financial ratios, and other key metrics for the equity specified. Data is generally refreshed on the same day a company reports its latest earnings and financials. </p>
		// <br>
		// <h6><b>API Parameters</b></h6>
		// <p><b>❚ Required: <code>function</code></b></p>
		// <p>The function of your choice. In this case, <code>function=OVERVIEW</code> </p>
		// <p><b>❚ Required: <code>symbol</code></b></p>
		// <p>The symbol of the token of your choice. For example: <code>symbol=IBM</code>.
		// </p>
		// <p><b>❚ Required: <code>apikey</code></b></p>
		// <p>Your API key. Claim your free API key <a href="https://www.alphavantage.co/support/#api-key" target="_blank">here</a>. </p>
		// <br>

		// Is it premium?
		endpoint.Premium = endpointHead.Find(".premium-label").Length() > 0

		// Extract endpoint name
		readableName, err := endpointHead.Html()
		if err != nil {
			panic(fmt.Errorf("could not extract html from %v header", endpointHead.Text()))
		}

		spanStart := strings.Index(readableName, "<span")
		if spanStart == -1 {
			endpoint.ReadableName = html.UnescapeString(readableName)
		} else {
			readableName = html.UnescapeString(readableName[0:spanStart])
			endpoint.ReadableName = readableName
		}

		// Extract endpoint description
		descP := endpointHead

		// Some endpoints have a <br> after the H4, but some go right to the <p>  Make sure you find the <p>
		for goquery.NodeName(descP) != "p" {
			descP = descP.Next()
		}

		var descBuffer bytes.Buffer
		for goquery.NodeName(descP) != "h6" {
			if goquery.NodeName(descP) == "p" {
				endpointDesc, err := descP.Html()
				if err != nil {
					panic(fmt.Errorf("could not extract html from %v endpoint description tag", linkName))
				}
				_, err = fmt.Fprintf(&descBuffer, "%v\n", endpointDesc)
				if err != nil {
					panic(fmt.Errorf("could not write to endpoint description buffer: %w", err))
				}
				endpoint.Desc = strings.TrimSpace(descBuffer.String())
			}
			descP = descP.Next()
		}

		// Extract parameters

		// Get to `<p><b>❚ Required: <code>function</code></b></p>`
		functionParamP := descP.Next()

		functionParam, err := readParameter(functionParamP, linkName)
		if err != nil {
			panic(fmt.Errorf("could not extract function parameter for %v: %w", linkName, err))
		}
		if functionParam.Name != "function" || functionParam.Required == false {
			panic(fmt.Errorf("incorrectly extracted function parameter for %v", linkName))
		}

		// Pull the actual function key out of the description, which looks like this:
		// The API function of your choice. In this case, <code>function=SYMBOL_SEARCH</code>
		functionDesc := functionParam.Desc
		funcStart := strings.Index(functionDesc, "function=") + len("function=")
		funcEnd := strings.Index(functionDesc, "</code>")

		var functionName string
		if funcStart < 0 || funcEnd < 0 || linkName == "latestPrice" {
			// Special case: the documentation page has one endpoint that forgets to include the function.
			// It's endpoint #latestprice, and the function name is `GLOBAL_QUOTE`
			functionName = "GLOBAL_QUOTE"
		} else {
			functionName = functionDesc[funcStart:funcEnd]
		}
		endpoint.Function = functionName

		// Scrape the rest of the parameters
		param := functionParam
		paramP := functionParamP.Next()
		for param.Name != "apikey" { // apikey is always the last parameter
			// Find the next parameter, starting with the Required/Optional label.  It can be an arbitrary number of
			// elements down from here.

			paramText := paramP.Text()
			for strings.Contains(paramText, "Required") == false &&
				strings.Contains(paramText, "Optional") == false {
				paramP = paramP.Next()
				paramText = paramP.Text()
			}

			param, err = readParameter(paramP, linkName)
			if err != nil {
				panic(fmt.Errorf("could not scrape next parameter for %v: %w", linkName, err))
			}

			paramP = paramP.Next()
			endpoint.Params = append(endpoint.Params, *param)
		}

		endpoints = append(endpoints, endpoint)
	})

	return endpoints, nil
}

func readParameter(paramNode *goquery.Selection, linkName string) (*api.Parameter, error) {
	// paramNode should have this structure:
	// <p><b>❚ Required: <code>from_symbol</code></b></p>
	// <p>DESCRIPTION HERE</p>
	//
	// OR
	//
	// <p>❚ Optional: <code>NATURAL_GAS</code></p>    <-- Notice lack of <b> tags around Optional
	// <p>DESCRIPTION HERE</p>

	param := api.Parameter{}

	// Figure out if this parameter is required or not

	paramNameText := paramNode.Text()
	paramNameHtml, err := paramNode.Html()
	if err != nil {
		return nil, fmt.Errorf("Could not get param name HTML for %v: %w\n\"\"\"%v\"\"\"",
			linkName, err, paramNameText)
	}

	if strings.Contains(paramNameHtml, "Required") {
		param.Required = true
	} else if strings.Contains(paramNameHtml, "Optional") {
		param.Required = false
	} else {
		return nil, fmt.Errorf("could not determine required/optional for %v\n\"\"\"\n%v\n\"\"\"",
			linkName, paramNameHtml)
	}

	// Get the name
	nameNode := paramNode.Find("code").First()
	if nameNode.Length() != 1 {
		return nil, fmt.Errorf("could not determine parameter name for %v", linkName)
	}
	param.Name = nameNode.Text()

	// Get the description, which can be one or more
	descP := paramNode.Next()
	descText := descP.Text()
	var descBuffer bytes.Buffer
	for strings.Contains(descText, "Required") == false &&
		strings.Contains(descText, "Optional") == false &&
		goquery.NodeName(descP) != "br" {

		descHtml, err := descP.Html()
		if err != nil {
			panic(fmt.Errorf("could not extract the parameter description for %v param %v: %w",
				linkName, param.Name, err))
		}

		_, err = fmt.Fprintf(&descBuffer, "%v\n", descHtml)
		descP = descP.Next()
		descText = descP.Text()
	}
	param.Desc = strings.TrimSpace(descBuffer.String())

	return &param, nil
}

// Cloudflare includes several one-time values throughout the document for various protections.  They do not affect
// the content we need, but mean that every page download is different, and therefore we can't simply checksum the
// entire page.  removeCloudflareStuff deletes these one-time values, leaving the rest of the document intact.
func removeCloudflareStuff(documentBytes []byte) []byte {
	// The three known issues are:
	// * a `data-cfemail="<base64 number>"` tag in the "Market News & Sentiment" endpoint `limit` parameter
	// * the ContactUs e-mail href in the page footer
	// * the cloudflare script data tag at the bottom of the page.

	// Deal with the data-cfemail tag.

	var newDocument []byte
	newDocumentBuffer := bytes.NewBuffer(newDocument)

	cfemailTag := []byte("data-cfemail=")
	cfemailIndex := bytes.Index(documentBytes, cfemailTag)

	// Dump the entire document up to this point in a new document buffer and clear it from the remainder
	newDocumentBuffer.Write(documentBytes[:cfemailIndex-1])
	documentBytes = documentBytes[cfemailIndex:]

	// The start of the remaining document looks like this:
	// data-cfemail="b3c0c6c3c3dcc1c7f3d2dfc3dbd2c5d2ddc7d2d4d69dd0dc">[email&#160;protected]</a>

	// Ignore everything up to the second ", but leave the ">[email..." in tact to close the tag.
	closingBracketIndex := bytes.Index(documentBytes, []byte{'>'})
	documentBytes = documentBytes[closingBracketIndex:] // SKIP

	// Next, find and cleanse the Contact Us link from the footer:
	// <a href="/cdn-cgi/l/email-protection#295a5c5959465b5d6948455941485f48475d484e4c074a46">Contact us</a>

	contactUsTag := []byte("<a href=\"/cdn-cgi/l/email-protection#")
	contactUsIndex := bytes.Index(documentBytes, contactUsTag)

	// Write everything from the end of the last problem to the start of this problem
	newDocumentBuffer.Write(documentBytes[:contactUsIndex])
	documentBytes = documentBytes[contactUsIndex:]

	contactUsText := []byte("Contact us")
	newDocumentBuffer.Write(contactUsText)

	linkEndIndex := bytes.Index(documentBytes, []byte("</li>"))
	documentBytes = documentBytes[linkEndIndex:]

	// The last thing in the document is a line with two script tag which we can completely ignore.
	scriptStart := []byte("<script ")
	scriptStartIndex := bytes.Index(documentBytes, scriptStart)
	newDocumentBuffer.Write(documentBytes[:scriptStartIndex-1])
	documentBytes = documentBytes[scriptStartIndex:]

	scriptEnd := []byte("\n")
	scriptEndIndex := bytes.Index(documentBytes, scriptEnd) + len(scriptEnd)
	documentBytes = documentBytes[scriptEndIndex:]

	// Whew....
	newDocumentBuffer.Write(documentBytes)
	return newDocumentBuffer.Bytes()
}
