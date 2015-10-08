
serve:
	python -m SimpleHTTPServer
datamaps.world.min.js:
	wget http://datamaps.github.io/scripts/datamaps.world.min.js

pages.csv:
	perl dumpmetrics.pl > pages.csv
trees.json: pages.csv
	echo TBD
