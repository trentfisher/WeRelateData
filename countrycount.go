package main

// generate a summary of countries

import (
    "encoding/csv"
    "os"
    "fmt"
    "io"
    "werelate"
    "country"
    "strconv"
)

func main() {
    indexfile, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening file:", err)
		return
	}
	defer indexfile.Close()

    cn := map[string]int {}
    reader := csv.NewReader(indexfile)
    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        } else if err != nil {
            fmt.Fprintln(os.Stderr, "Error: bad record ",err)
            continue
        }

        // only count people
        namespace, _ := werelate.SplitTitle(record[1])
        if (namespace != "Person") { continue }
        //if (len(record[2]) == 0) { record[2] = "???" }
        cn[record[2]]++
    }

    csvout := csv.NewWriter(os.Stdout)
    csvout.Write([]string{"country","name","population"})
    for i := range cn {
        if (i == "") { continue }
        csvout.Write([]string{
            i, country.Code2country(i), strconv.Itoa(cn[i])})
    }
    csvout.Flush()
}
