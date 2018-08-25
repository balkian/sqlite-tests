#!/bin/sh
DBFILE="$1.cli.db"
sqlite3 $DBFILE 'create table if not exists followers (user integer, follower integer)'
sqlite3 $DBFILE '.mode tabs' ".import $1 followers"
sqlite3 $DBFILE 'create index if not exists followers_user ON followers(user);' 'create index if not exists followers_follower on followers(follower);' 'create unique index if not exists followers_unique on followers(user,follower);'