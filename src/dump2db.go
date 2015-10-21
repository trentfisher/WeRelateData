package main

import (
    "database/sql"
    "fmt"
    _ "github.com/mattn/go-sqlite3"
    "log"
    "os"
    "werelate"
    "strconv"
    "encoding/xml"
    "regexp"
    "country"
)

// representations of each page in the dump file
type Redirect struct {
	Title string `xml:"title,attr"`
}

type Page struct {
	Title string `xml:"title"`
	Redir Redirect `xml:"redirect"`
	Text string `xml:"revision>text"`
}
type NameSpace struct {
    Key int `xml:"key,attr"`
    Name string `xml:",chardata"`
}
type NameSpaces struct {
    Names []NameSpace `xml:"namespace"`
}
// map namespace names to id numbers
var namespace2id map[string]int

// representation of the person and family data inside each page
type Event struct {
    Type    string `xml:"type,attr"`
    Date    string `xml:"date,attr"`
    Place   string `xml:"place,attr"`
    Sources string `xml:"sources,attr"`
    Text    string `xml:",chardata"`
}
type Source struct {
    Id    string `xml:"id,attr"`
    Title string `xml:"title,attr"`
    Text  string `xml:",chardata"`
}
type Name struct {
    Given   string `xml:"given,attr"`
    Surname string `xml:"surname,attr"`
}
type Link struct {
    Title string  `xml:"title,attr"`
}
type Person struct {
    Name             Name     `xml:"name"`
    Gender           string   `xml:"gender"`
    Event_fact       []Event  `xml:"event_fact"`
    Source_citation  []Source `xml:"source_citation"`
    Child_of_family  []Link   `xml:"child_of_family"`
    Spouse_of_family []Link   `xml:"spouse_of_family"`
    Text             string
    namespace        string
    title            string
    country          string
}
 type Family struct {
    Source_citation  []Source `xml:"source_citation"`
    Child            []Link   `xml:"child"`
    Husband          []Link     `xml:"husband"`
    Wife             []Link     `xml:"wife"`
    Text             string
    namespace        string
    title            string
}

func setupdb(dbfile string) *sql.DB {
    db, err := sql.Open("sqlite3", dbfile)
    if err != nil {
        log.Fatal("Error: sqlite db open", err)
    }

    return db
}

// ------------------------------------------------------------------------
// do the first pass through the xml file,
// populating the database with basic info
func loadindex(db *sql.DB, pagefile string) {
    // start a transaction
    tx, err := db.Begin()
    if err != nil {
        log.Fatal(err)
    }

    // set up the page insert statement
    stmt, err := tx.Prepare("insert into pages(id, namespace, name, start, end) values(?,?,?,?,?)")
    if err != nil {
        log.Fatal("Error: prepare insert page failed:",err)
    }
    defer stmt.Close()

    // and another for inserting namespace info

	xmlFile, err := os.Open(pagefile)
	if err != nil {
		fmt.Println("Error: opening file:", err)
		return
	}
	defer xmlFile.Close()

	decoder := xml.NewDecoder(xmlFile)
	total := 0
    var lastoffset int64
	var inElement string
	for {
        lastoffset = decoder.InputOffset()
		// Read tokens from the XML document in a stream.
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		// Inspect the type of the token just read.
		switch se := t.(type) {
		case xml.StartElement:
			inElement = se.Name.Local
			if inElement == "page" {
				var p Page
				// decode a whole chunk of following XML
                decoder.DecodeElement(&p, &se)
                namespace, title := werelate.SplitTitle(p.Title)
                n := namespace2id[namespace]
                //fmt.Fprintf(os.Stderr, "namespace %s => %d\n", namespace, n)
                _, err = stmt.Exec(total, 
                    n, title,
                    strconv.FormatInt(lastoffset, 10),
                    strconv.FormatInt(decoder.InputOffset(), 10))
                if err != nil {
                    log.Fatal(err)
                }
                total++
                fmt.Fprintf(os.Stderr, "Indexing page %d\r", total)
			} else if (inElement == "namespaces") {
                var n NameSpaces
                decoder.DecodeElement(&n, &se)
                //fmt.Fprintf(os.Stderr, "namespaces = %q\n", n);
                stmt, err := tx.Prepare("insert into namespaces(id, name) values(?, ?)")
                if err != nil {
                    log.Fatal(err)
                }
                defer stmt.Close()
                namespace2id = make(map[string]int)
                for i := range n.Names {
                    //fmt.Fprintf(os.Stderr, "namespace %d => %s\n", n.Names[i].Key, n.Names[i].Name)
                    _, err = stmt.Exec(n.Names[i].Key, n.Names[i].Name)
                    if err != nil {
                        log.Fatal(err)
                    }
                    namespace2id[n.Names[i].Name] = n.Names[i].Key
                }
                fmt.Printf("NAMESPACES %q\n", namespace2id)

            }
		default:
		}
	}
    tx.Commit()
}

// ------------------------------------------------------------------------
func persondata(str string) (factsxml Person) {
    facts, text := splitfacts(str)
    err := xml.Unmarshal([]byte(facts), &factsxml)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: xml person facts parse %v %s\n", err, str)
		return
	}
    factsxml.Text = text
    return factsxml
}

// split the wiki page contents into the xml facts section and the page
// text which doesn't seem to be xml
func splitfacts(str string) (facts, text string) {
    m := regexp.MustCompile("(?is)^(.*<(/family|/person|family/|person/)>)(.*)").
        FindStringSubmatch(str)
    if (len(m) != 4) {
        fmt.Fprintf(os.Stderr, "Error: split facts failed for %s\n", str)
        facts = ""
        text = str
    } else {
        facts = m[1]
        text = m[3]
    }
    // trim out some gunk and white space from the text
    text = regexp.MustCompile("<show_sources_images_notes/>").
        ReplaceAllString(text, "")
    text = regexp.MustCompile("(?s)[ \t\r\n]+$").ReplaceAllString(text, "")
    text = regexp.MustCompile("(?s)^[ \t\r\n]+").ReplaceAllString(text, "")
    return facts, text
}
func familydata(str string) (factsxml Family) {
    facts, text := splitfacts(str)
    err := xml.Unmarshal([]byte(facts), &factsxml)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: xml facts parse: %v: %s\n", err, str)
		return
	}
    factsxml.Text = text
    return factsxml
}

// what country is this person from?
func getcountry(personxml Person) string {
    cncnt := map[string]int {} // count how many times we see each country
    birthcn := ""
    for i := range personxml.Event_fact {
        p := personxml.Event_fact[i].Place
        p = regexp.MustCompile("[|].+").ReplaceAllString(p, "")
        p = regexp.MustCompile("^.+, *").ReplaceAllString(p, "")
        p = regexp.MustCompile(" +$").ReplaceAllString(p, "")
        c := country.Country2code(p)
        fmt.Fprintf(os.Stderr, "Event %d %s => %s => %s\n", i, personxml.Event_fact[i].Place, p, c)
        cncnt[c]++
        if (personxml.Event_fact[i].Type == "Birth" ||
            personxml.Event_fact[i].Type == "Christening") {
            birthcn = c
        }
    }

    // we prefer birth data
    if (len(birthcn) > 0) { return birthcn }

    cn := ""
    for i := range cncnt {
        if (len(cn) == 0 || cncnt[i] > cncnt[cn]) { cn = i }
    }
    return cn
}

// get a chunk of xml
func fetchxmlfrag(pagefile *os.File, start int64, end int64) (p Page) {
    buf := make([]byte, end-start)
    pagefile.ReadAt(buf, start)
    err := xml.Unmarshal(buf, &p)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: parsing xml for: %v", err)
    }
    return p
}

func redirectpage(p Page) bool {
    m := regexp.MustCompile("(?is)[\n\r]#REDIRECT[ ]*(.+?)[\n\r]").
        FindStringSubmatch(p.Text)
    if (len(m) == 1) {
        fmt.Println("REDIRECT to",m[0])
        return true
    }
    return false
}

// ------------------------------------------------------------------------
// now load the database with the detailed information from each page
func loadpageinfo(db *sql.DB, pagefile *os.File) {
    // start a transaction
    tx, err := db.Begin()
    if err != nil {
        log.Fatal(err)
    }

    // prepare some queries for later
    addcountry, err := tx.Prepare("update pages set country=? where id=?")
    if err != nil {
        log.Fatal("Error: prepare add country failed:",err)
    }
    defer addcountry.Close()
    addlink, err := tx.Prepare("insert into links select ?, id from pages where namespace = ? and name = ?");
    if err != nil {
        log.Fatal("Error: prepare insert link failed:",err)
    }
    defer addlink.Close()

    // go through every page entry
    rows, err := db.Query(
        "select pages.id, namespaces.name, pages.name,start, end from pages join namespaces on namespaces.id = pages.namespace")
    if err != nil {
        log.Fatal(err)
    }
    for rows.Next() {
        var id int
        var start, end int64
        var namespace, title string
        rows.Scan(&id, &namespace, &title, &start, &end)
        fmt.Println(">>>", id, namespace, title, start, end)

        p := fetchxmlfrag(pagefile, start, end)

        // is this a redirect page?
        if (redirectpage(p)) {
            continue
        }

        switch namespace {
        case "Person":
            //fmt.Println("PAGE",p.Title,"NAMESPACE",namespace,"TITLE",title)
            personxml := persondata(p.Text)

            // which country is this person "from"?
            _, err = addcountry.Exec(getcountry(personxml), id)
            if err != nil {
                log.Fatal(err)
            }

            // add links to families
            for i := range personxml.Child_of_family {
                fmt.Println("LINK",id,"to","Family",personxml.Child_of_family[i].Title,namespace2id["Family"]);
                _,err = addlink.Exec(id, namespace2id["Family"], personxml.Child_of_family[i].Title)
                if err != nil {
                    log.Fatal(err)
                }
            }
            for i := range personxml.Spouse_of_family {
                fmt.Println("LINK",id,"to","Family",personxml.Spouse_of_family[i].Title,namespace2id["Family"]);
                _,err = addlink.Exec(id, namespace2id["Family"], personxml.Spouse_of_family[i].Title)
                if err != nil {
                    log.Fatal(err)
                }
            }
            
            //fmt.Printf("PERSON %q\n", personxml)
        case "Family":
            //fmt.Println("PAGE",p.Title,"NAMESPACE",namespace,"TITLE",title)
            familyxml := familydata(p.Text)

            // links to people
            for i := range familyxml.Child {
                fmt.Println("LINK",id,"to","Person",familyxml.Child[i].Title,namespace2id["Person"]);
                _,err = addlink.Exec(id, namespace2id["Person"], familyxml.Child[i].Title)
                if err != nil {
                    log.Fatal(err)
                }
            }
            for i := range familyxml.Wife {
                fmt.Println("LINK",id,"to","Person",familyxml.Wife[i].Title,namespace2id["Person"]);
                _,err = addlink.Exec(id, namespace2id["Person"], familyxml.Wife[i].Title)
                if err != nil {
                    log.Fatal(err)
                }
            }
            for i := range familyxml.Husband {
                fmt.Println("LINK",id,"to","Person",familyxml.Husband[i].Title,namespace2id["Person"]);
                _,err = addlink.Exec(id, namespace2id["Person"], familyxml.Husband[i].Title)
                if err != nil {
                    log.Fatal(err)
                }
            }
            fmt.Printf("FAMILY %q\n", familyxml)
        }
    }

    tx.Commit()
}

func main() {
    db := setupdb(os.Args[1])

	pagefile, err := os.Open(os.Args[2])
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer pagefile.Close()

    // first pass, gather all page names and put them into the db
    loadindex(db, os.Args[2])

    // second pass, fill in details from each page
    loadpageinfo(db, pagefile)

    // TBD traverse family connections

    db.Close()
}
