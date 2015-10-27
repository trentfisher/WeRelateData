
create table pages (
       id        integer not null primary key,
       namespace integer,
       name      text,
       redirect  integer,
       start     integer,
       end       integer,
       country   text);
delete from pages;

create table namespaces (
       id integer not null primary key,
       name text);
delete from namespaces;

create table links (id1 integer, id2 integer);
delete from links;

-- select namespaces.name, pages.name from pages join namespaces on namespaces.id = pages.namespace;
-- insert into links select 1883800, id from pages where namespace = 108 and name = "Elisabeth Guilbeau (1)";
-- update table set country=? where id=?
