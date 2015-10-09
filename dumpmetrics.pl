#!/usr/bin/env perl
#
# script to generate various data and statistics from the WeRelate xml dump
# this dumps out a CSV of various data
# 

use strict;
use warnings;

use XML::LibXML::Reader;
use DBM::Deep;
use Locale::Country;

$| = 1;

my $debug = 0;
my $lt = time;  #for progress indicators
my $count = 0;

# this stores information about each page
my $index = DBM::Deep->new(
    file => "pages.db",
    locking => 0,
    autoflush => 0,
    max_buckets => 48,
    pack_size => 'large',
);

# where are we reading the xml from?
my $infile = shift @ARGV;
my $in;
if (not $infile or $infile eq "-")
{
    $in = *STDIN;
}
elsif ($infile =~ /.gz$/)
{
    open($in, "gzip -dc $infile |") or die "Error: gzip $infile: $!\n";
}
else
{
    open($in, $infile) or die "Error: read $infile: $!\n";
}

processXML($in, $index);

# figure out which trees each person belongs to
my @trees = ("indexed by one, ignore");
foreach my $k (keys %$index)
{
    next unless $k =~ /^Person:/;
    print STDERR "Traversing tree ",1+$#trees,".";
    my $c = traverse($index, 1+$#trees, $k);
    push @trees, { count => $c, name => $k } if $c;
    print STDERR "\n";
}
print STDERR "\n";

while (my ($k, $v) = each %$index)
{
    print join(",",
	       $k,
	       $index->{$k}{namespace},
	       $index->{$k}{title},
	       $index->{$k}{tree}||"",
	       $index->{$k}{cn}||"",
	),"\n";
}
exit 0;
foreach my $t (1..$#trees)
{
    printf("%d,%d,%s\n",
	   $t, $trees[$t]->{count}, $trees[$t]->{name})
	if $trees[$t];
}

exit 0;

sub processXML
{
    my $in = shift;
    my $index = shift;
    my $namespaces = {};
    my $namespacepat = qr(^(\w+|\w+ talk));

    my $reader = XML::LibXML::Reader->new(IO => $in)
	or die "Error: cannot read $infile\n";
    while ($reader->read)
    {
	printf "%d %d %s %d\n", ($reader->depth,
				 $reader->nodeType,
				 $reader->name,
				 $reader->isEmptyElement)
	    if $debug;
	if ($reader->nodeType == XML_READER_TYPE_ELEMENT)
	{
	    if ($reader->name eq "page")
	    {
		print STDERR "reading page ",$count++,"\r";

		processPage($index, $reader->copyCurrentNode(1), $namespacepat);
	    }
	    elsif ($reader->name eq "namespace")
	    {
		$namespaces->{$reader->readInnerXml} = $reader->getAttribute("key");
	    }
	}
	elsif ($reader->nodeType == XML_READER_TYPE_END_ELEMENT)
	{
	    if ($reader->name eq "namespaces")
	    {
		$namespacepat = "(".join("|", keys %$namespaces).")";
		$namespacepat = qr($namespacepat)o;
	    }
	}
    }
    print STDERR "done reading $count pages\n";
}

sub processPage
{
    my $index = shift;
    my $dom = shift;
    my $namespacepat = shift;

    my $qtitle = $dom->getChildrenByTagName("title")->get_node(1)->textContent;

    my ($namespace, $title) = ($qtitle =~ /^$namespacepat:(.+)/);
    if (not $title) { $title = $namespace; $namespace = "none"; }
    $index->{$qtitle}{namespace} = $namespace;
    $index->{$qtitle}{title} = $title;
    return unless $namespace eq "Person" or $namespace eq "Family";

    # this is horrid, but xpath won't work, so I have to reprocess this
    # fragment as a separate document
    my $pgtext = $dom->getChildrenByTagName("revision")->get_node(1)
	->getChildrenByTagName("text")->get_node(1)->textContent;
    my ($facts, $text) = ($pgtext =~ m,(.+</(family|person)>)(.*),s);
    return if not $facts;

    my $textdom = eval { XML::LibXML->load_xml(string => $facts); };
    if (not $textdom)
    {
	warn "Error: parse of $qtitle failed $@\n";
	return;
    }

    # gather links to family members
    if ($namespace eq "Family")
    {
	$index->{$qtitle}{links} = [
	    map("Person:".$_,
		$textdom->find('/family/child/@title')->to_literal_list,
		$textdom->find('/family/husband/@title')->to_literal_list,
		$textdom->find('/family/wife/@title')->to_literal_list),
	    ];
    }
    elsif ($namespace eq "Person")
    {
	$index->{$qtitle}{links} = [
	    map("Family:".$_,
		$textdom->find('/person/child_of_family/@title')->to_literal_list,
		$textdom->find('/person/spouse_of_family/@title')->to_literal_list),
	    ];
	# get a list of event places
	$index->{$qtitle}{cn} =
	    country_code($textdom->find('/person/event_fact/@place')->to_literal_list)||"?";
    }
}

# convert list of places to country codes and pick the one seen most often
sub country_code
{
    my @places = @_;
    my $c = {};

    foreach my $p (@places)
    {
	$p =~ s/\s*\|.+$//;  # chop off alt text
	$p =~ s/^.+,\s*//;  # chop off before comma... should be country
	my $cn = country2code($p, LOCALE_CODE_ALPHA_3);
	next unless $cn;
	$c->{$cn}++;
    }
    (sort {$c->{$b} <=> $c->{$a}} keys %$c)[0];
}

# traverse the person/family connections and return the number of
# people traversed
sub traverse
{
    my $db = shift;      # database of people/families and connections
    my $treenum = shift; # which tree we are adding to
    my $name = shift;    # name to check

    # nonexistent
    return 0 unless $db->{$name};

    # already marked as part of a tree
    return 0 if $db->{$name}{tree};

    # this one isn't part of a tree, add it
    my $ret = ($name =~ /^Person:/ ? 1 : 0);
    $db->{$name}{tree} = $treenum;
    print STDERR ".";

    # now traverse each connection
    my @links = (ref $db->{$name}{links} ?
		 @{$db->{$name}{links}} : ());
    foreach my $p (@links)
    {
	$ret += traverse($db, $treenum, $p);
    }
    return $ret;
}
