all: pageinfo.csv serve

serve:
	python -m SimpleHTTPServer
datamaps.world.min.js:
	wget http://datamaps.github.io/scripts/datamaps.world.min.js

DUMPFILE=pages-30-Sep-2015.xml
pages.csv: $(DUMPFILE) mkindex
	./mkindex $(DUMPFILE) > $@
pageinfo.csv: $(DUMPFILE) dumpmetrics pages.csv
	./dumpmetrics pages.csv $(DUMPFILE) > $@

mkindex: mkindex.go
	go build $<
dumpmetrics: dumpmetrics.go
	go build $<
trees.json: pages.csv
	echo TBD
