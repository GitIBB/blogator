# Blogator is a blog aggregator microservice application built in the Go language, using PostgreSQL for database functionality.

## Prerequisites

To run the application, the following must be installed on your machine:
###    Golang 1.23.2

#### Option 1: use the webi installer. Run the following command in your terminal:
```bash
curl -sS https://webi.sh/golang | sh
```

Then, read the output of the command and follow any instructions.

#### Option 2: Use the official installation instructions, which can be found at https://go.dev/doc/install




### PostgreSQL 16.6
#### macOS
```bash
brew install postgresql@16
```

#### Linux / WSL (Debian)

```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
```

verify that your PostgreSQL installation succeeded with the following command:
```bash
psql --version
```

#### Database Setup

After installing PostgreSQL, set up your password:
```bash
# Log into PostgreSQL as the postgres user
sudo -u postgres psql

# Once in the PostgreSQL prompt, set your password
ALTER USER postgres PASSWORD 'your-password';

# Exit PostgreSQL prompt
```

then, create the database:

```bash
createdb gator
```

then start up the PostgreSQL server:

##### On macOS
```bash
brew services start postgresql
```

##### On Linux/WSL
```bash
sudo service postgresql start
```

## Gator Installation Instructions:

run the following command in a terminal:

```bash
go install github.com/GitIBB/gator@latest
```

## Gator usage

Gator uses a configuration file located at the user's home directory `~/.gatorconfig.json`. Create this file with the following content:

```json
{
    "db_url": "postgresql://postgres:your-password@localhost:5432/gator"
}
```


### Commands

- `gator login` - Authenticates an existing user and returns an authentication token
```bash
gator login username
```

- `gator register` - Creates a new user account in the system
```bash
gator register examplename
```

- `gator reset` - Resets the database to its initial state (use with caution)
```bash
gator reset
```

- `gator users` - Lists all registered users in the system
```bash
gator users
```

- `gator agg` - Aggregates and updates all RSS feeds in the database. Takes an optional time interval for continuous updates.
```bash
# Run once
gator agg

# Run every 30 seconds
gator agg 30s

# Run every minute
gator agg 1m

# Run every 5 minutes
gator agg 5m
```

- `gator addfeed [Name] [URL]` - Adds a new RSS feed to track (requires authentication)
```bash
gator addfeed example_feedname "https://blog.exampleurl.com/feed.xml"
```
- `gator feeds` - Lists all RSS feeds currently tracked in the system
```bash
gator feeds
```

- `gator follow [FEED_URL]` - Subscribes the authenticated user to a specific feed
```bash
gator follow https://blog.exampleurl.com/feed.xml
```

- `gator following` - Shows all feeds the authenticated user is following
```bash
gator following
```

- `gator unfollow [FEED_URL]` - Removes the authenticated user's subscription to a feed
```bash
gator unfollow "https://blog.exampleurl.com/feed.xml"
```

- `gator browse` - Shows recent posts from feeds the authenticated user follows
```bash
gator browse
```
