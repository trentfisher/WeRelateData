GOPATH=$(shell pwd)
export GOPATH
DUMPFILE=pages-30-Sep-2015.xml
DBFILE=pages.db
.PRECIOUS: $(DBFILE) $(DUMPFILE)

all: $(DBFILE) serve

serve:
	python -m SimpleHTTPServer
datamaps.world.min.js:
	wget http://datamaps.github.io/scripts/datamaps.world.min.js

$(DBFILE): src/dump2db.go $(DUMPFILE)
	sqlite3 $@ < werelate.sql
	go run src/dump2db.go $@ $(DUMPFILE)

# first build an index of all pages
pages.csv: $(DUMPFILE) mkindex
	./mkindex $(DUMPFILE) > $@
# now fill in more details using that index
pageinfo.csv: $(DUMPFILE) dumpmetrics pages.csv
	./dumpmetrics pages.csv $(DUMPFILE) > $@
# now generate summaries from that data
# ... people by country
countries.csv: countrycount.go pageinfo.csv
	go run countrycount.go pageinfo.csv > $@
# ... page counts by namespace
pagecount.csv: pagecount.go pageinfo.csv
	go run pagecount.go pageinfo.csv > $@

mkindex: mkindex.go
	go build $<
dumpmetrics: dumpmetrics.go
	go build $<
trees.json: pages.csv
	echo TBD
