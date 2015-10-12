package main

// go through the WeRelate dump file and generate a csv of all pages and their offsets within the file

import (
//	"bufio"
	"fmt"
	"os"
	"flag"
	"encoding/xml"
    "encoding/csv"
//	"strings"
	"regexp"
    "strconv"
)

var inputFile = flag.String("infile", "pages-30-Sep-2015.xml", "Input file path")

var filter, _ = regexp.Compile("^file:.*|^talk:.*|^special:.*|^wikipedia:.*|^wiktionary:.*|^user:.*|^user_talk:.*")

// Here is an example article from the Wikipedia XML dump
//
// <page>
// 	<title>Apollo 11</title>
//      <redirect title="Foo bar" />
// 	...
// 	<revision>
// 	...
// 	  <text xml:space="preserve">
// 	  {{Infobox Space mission
// 	  |mission_name=&lt;!--See above--&gt;
// 	  |insignia=Apollo_11_insignia.png
// 	...
// 	  </text>
// 	</revision>
// </page>
//
// Note how the tags on the fields of Page and Redirect below
// describe the XML schema structure.

type Redirect struct {
	Title string `xml:"title,attr"`
}

type Page struct {
	Title string `xml:"title"`
	Redir Redirect `xml:"redirect"`
//	Text string `xml:"revision>text"`
}

func main() {
//	flag.Parse()

	xmlFile, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer xmlFile.Close()

    csvout := csv.NewWriter(os.Stdout)

	decoder := xml.NewDecoder(xmlFile)
	total := 0
    var lastoffset int64
	var inElement string
	for {
        lastoffset = decoder.InputOffset()
        //fmt.Printf("offset %d %d\n", lastoffset, decoder.InputOffset())
		// Read tokens from the XML document in a stream.
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		// Inspect the type of the token just read.
		switch se := t.(type) {
		case xml.StartElement:
			// If we just read a StartElement token
			inElement = se.Name.Local
			// ...and its name is "page"
			if inElement == "page" {
//                offset := decoder.InputOffset();
				var p Page
				// decode a whole chunk of following XML
				decoder.DecodeElement(&p, &se)
//                fmt.Printf("%s,%d,%d\n", p.Title,
//                    lastoffset, decoder.InputOffset())
                csvout.Write([]string{
                    p.Title,
                    strconv.FormatInt(lastoffset, 10),
                    strconv.FormatInt(decoder.InputOffset(), 10)})
                total++
                fmt.Fprintf(os.Stderr, "reading page %d\r", total)
			}
		default:
		}
		
	}
    csvout.Flush()
//	fmt.Printf("Total articles: %d \n", total)
}
