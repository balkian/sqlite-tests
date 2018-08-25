import sys
import os
import os.path
import sqlite3


def addusers(c, ufrom, uto):
    # Insert a row of data
    c.execute("INSERT OR IGNORE INTO followers VALUES (?, ?)", [ufrom, uto])


def update_progress(i, current, total):
    sys.stdout.write("\033[F") #back to previous line
    sys.stdout.write("\033[K") #clear line
    print('Done: {} lines. {} / {} bytes ({} %)'.format(i, current, total, 100.0*current/float(total)))


def main(infile):
    conn = sqlite3.connect('%s.py.db' % os.path.basename(infile))
    # Create table
    conn.execute('''CREATE TABLE IF NOT EXISTS followers
                (user int, follower int)''')
    conn.execute("CREATE UNIQUE INDEX IF NOT EXISTS followersindex ON followers(user, follower) ")
    conn.execute("CREATE INDEX IF NOT EXISTS followersindex_follower ON followers(follower) ")
    conn.execute("CREATE INDEX IF NOT EXISTS followersindex_user ON followers(user) ")

    with open(infile) as f:
        total = os.fstat(f.fileno()).st_size
        for i, line in enumerate(f):
            tokens = line.strip().split('\t')
            if len(tokens) != 2:
                print('Wrong line: ', i, tokens)
                continue
            addusers(conn, tokens[0], tokens[1])

            if i % 10000 == 0:
                conn.commit()
                update_progress(i, f.tell(), total)
            # Save (commit) the changes
    conn.commit()

    # We can also close the connection if we are done with it.
    # Just be sure any changes have been committed or they will be lost.
    conn.close()
    update_progress(i, total, total)

if __name__ == '__main__':
    main(sys.argv[1])