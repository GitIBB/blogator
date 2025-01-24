package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"gator/internal/config"
	"gator/internal/database"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lib/pq"
)

// ------------ Non-command structs ----------

type state struct {
	db        *database.Queries
	configPTR *config.Config
}

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// ----------- Command handlers ---------------

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("username required")
	}
	username := cmd.args[0] // set username to first arg

	ctx := context.Background() // create background context for db operations

	// check if user exists
	user, err := s.db.GetUser(ctx, username)
	if err != nil || user.Name == "" {
		fmt.Fprintf(os.Stderr, "user does not exist: %s\n", username)
		os.Exit(1)
	}

	// sets user to username
	if err := s.configPTR.SetUser(username); err != nil {
		return err
	}

	fmt.Printf("Username set to %s\n", username)
	return nil

}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return errors.New("username required")
	}

	username := cmd.args[0] // set username to first arg

	ctx := context.Background() // create background context for db operations

	existingUser, err := s.db.GetUser(ctx, username) // check if user exists
	if err == nil && existingUser.Name != "" {
		fmt.Fprintf(os.Stderr, "user already exists: %s\n", username)
		os.Exit(1)
	}

	// define neccessary params for a new user
	params := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      username,
	}

	newUser, err := s.db.CreateUser(ctx, params) // create new user
	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}
	if err := s.configPTR.SetUser(username); err != nil { // Set user in config
		return err
	}

	fmt.Printf("Created user: %+v\n", newUser) //print success and user data
	return nil

}

// handlerReset handles the 'reset' command, which clears all users from the db
func handlerReset(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("expected 0 arguments, got %d", len(cmd.args))
	}
	ctx := context.Background()     // create new background context for db operations
	err := s.db.DeleteAllUsers(ctx) // call the DeleteAllUsers method to remove all users from db
	if err != nil {
		return fmt.Errorf("failed to reset the database: %v", err) // if deletion fails, return error with details
	}
	fmt.Println("Database reset successful") // prints success message to user
	return nil
}

func handlerUsers(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("expected 0 arguments, got %d", len(cmd.args))
	}
	ctx := context.Background()      // create new background context for db operations
	users, err := s.db.GetUsers(ctx) // get all users from db
	if err != nil {
		return fmt.Errorf("failed to get users: %v", err)
	}
	currentUser := s.configPTR.CurrentUserName

	for _, user := range users { // for loop for iterating through users and printing these users, adding (current) to the current user
		if user == currentUser {
			fmt.Printf("* %v (current)\n", user)
		} else {
			fmt.Printf("* %v\n", user)
		}
	}
	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("expected 1 argument (time_between_reqs), got %d", len(cmd.args))
	}
	ctx := context.Background() // create new background context for db operations

	timeBetweenRequests, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Collecting feeds every %v\n", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		err = scrapeFeeds(ctx, s.db)
		if err != nil {
			log.Printf("Error scraping feeds: %v", err)
			continue
		}
	}
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("expected 2 arguments (name, url), got %d", len(cmd.args))
	}

	ctx := context.Background() // background context for db operations

	params := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Url:       cmd.args[1],
		Name:      cmd.args[0],
		UserID:    user.ID,
	}

	newFeed, err := s.db.CreateFeed(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create feed")
	}

	fmt.Printf("Feed '%s' with URL '%s' and ID '%s' was created\n", newFeed.Name, newFeed.Url, newFeed.ID)

	// Add this new code here
	followParams := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    newFeed.ID,
	}

	_, err = s.db.CreateFeedFollow(ctx, followParams)
	if err != nil {
		return fmt.Errorf("failed to create feed follow: %v", err)
	}

	return nil
}

func handlerFeeds(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("expected 0 arguments, got %d", len(cmd.args))
	}
	ctx := context.Background()

	getFeeds, err := s.db.GetFeedCreator(ctx)
	if err != nil {
		return err
	}
	for _, feed := range getFeeds {
		fmt.Printf("Feed: %s\n | URL: %s\n | Creator: %s\n | ID: %s\n",
			feed.Name, feed.Url, feed.UserName, feed.ID)
	}
	return nil

}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("expected 1 argument (url), got %d", len(cmd.args))
	}
	ctx := context.Background() // background context for db operations

	feed, err := s.db.GetFeedByURL(ctx, cmd.args[0])
	if err != nil {
		return fmt.Errorf("failed to find feed by URL: %v", err)
	}

	params := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	}

	_, err = s.db.CreateFeedFollow(ctx, params)
	if err != nil {
		if pqError, ok := err.(*pq.Error); ok && pqError.Code == "23505" { // 23505 is postgres code for unique violation
			return fmt.Errorf("you are already following this feed")
		}
		return fmt.Errorf("failed to create feed follow: %v", err)
	}

	// Step 5: Provide confirmation of the operation
	feedName := feed.Name // Assuming `feed` has a `Name` field
	userName := user.Name // Assuming `currentUser` has a `Name` field
	fmt.Printf("User '%s' is now following feed '%s'\n", userName, feedName)

	return nil
}

func handlerGetFeedFollowsForUser(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("expected 0 argument (url), got %d", len(cmd.args))
	}
	ctx := context.Background() // background context for db operations

	feedFollows, err := s.db.GetFeedFollowsForUser(ctx, user.ID)
	if err != nil {
		return err
	}
	for _, feedFollow := range feedFollows {
		fmt.Println(feedFollow.FeedName)
	}
	return nil

}

func handlerUnfollowFeedForUser(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("expected 1 argument (url), got %d", len(cmd.args))
	}
	ctx := context.Background() // background context for db operations

	err := s.db.DeleteFeedFollow(ctx, database.DeleteFeedFollowParams{
		UserID: user.ID,
		Url:    cmd.args[0],
	})
	if err != nil {
		return fmt.Errorf("failed to unfollow feed by url: %v", err)
	}
	fmt.Printf("Successfully unfollowed %v\n", cmd.args[0])
	return nil
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	ctx := context.Background() // background context for db operations
	limit := 2
	// checks len of args and converts it to a string
	if len(cmd.args) > 0 {
		newLimit, err := strconv.Atoi(cmd.args[0])
		if err != nil {
			return err
		}
		limit = newLimit
	}

	params := database.GetUserPostsParams{
		UserID: user.ID,
		Limit:  int32(limit),
	}

	posts, err := s.db.GetUserPosts(ctx, params)
	if err != nil {
		return err
	}

	for _, post := range posts {
		// Title and description
		fmt.Printf("Title: %v\nDescription: %v\n", post.Title, post.Description)
		// URL and published date
		fmt.Printf("URL: %v\nPublished Date: %v\n", post.Url, post.PublishedAt)
		fmt.Println("-----------------------") // seperator between posts
	}
	return nil

}

// ------------ command struct(s) and functions ------------

type command struct {
	name string
	args []string
}

type commands struct {
	handlers map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.handlers[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	handler, exists := c.handlers[cmd.name]
	if !exists {
		return fmt.Errorf("command %q not found", cmd.name)
	}
	return handler(s, cmd) // Call the handler function for the command
}

// ----------- Other functions ------------

// fetchFeed function
func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gator")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var feed RSSFeed
	err = xml.Unmarshal(body, &feed)
	if err != nil {
		return nil, err
	}
	// unescape channel fields
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	// unescape each item's fields
	for i := range feed.Channel.Item {
		feed.Channel.Item[i].Title = html.UnescapeString(feed.Channel.Item[i].Title)
		feed.Channel.Item[i].Description = html.UnescapeString(feed.Channel.Item[i].Description)
	}

	return &feed, nil

}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	ctx := context.Background()
	return func(s *state, cmd command) error {
		currentUser, err := s.db.GetUser(ctx, s.configPTR.CurrentUserName) // get current user from DB
		if err != nil {
			return fmt.Errorf("failed to get current user: %v", err)
		}

		return handler(s, cmd, currentUser) // Call the wrapped handler on success
	}
}

func scrapeFeeds(ctx context.Context, q *database.Queries) error {
	feed, err := q.GetNextFeedToFetch(ctx) // get next feed
	if err != nil {
		return fmt.Errorf("failed to fetch feed by URL: %v", err)
	}
	rssFeed, err := fetchFeed(ctx, feed.Url) // fetch feed contents
	if err != nil {
		return fmt.Errorf("failed to fetch feed contents: %v", err)
	}
	err = q.MarkFeedFetched(ctx, feed.ID) // mark feed as fetched
	if err != nil {
		return fmt.Errorf("failed to mark feed as fetched: %v", err)
	}
	for _, item := range rssFeed.Channel.Item {
		// Generate a UUID for the post
		postID := uuid.New()

		// timeFormat slice of strings
		timeFormats := []string{
			time.RFC1123Z,
			time.RFC1123,
			time.RFC822,
			"2006-01-02T15:04:05Z", // ISO 8601
			// Add more formats as needed
		}

		// parse the published date
		publishedAt := time.Now() // default fallback
		if item.PubDate != "" {
			for _, format := range timeFormats {
				parsed, err := time.Parse(format, item.PubDate)
				if err == nil {
					publishedAt = parsed
					break // parse success, exit loop
				}
			}
		}
		// if reach this point and PublishedAt is still default, none of formats worked
		if publishedAt.Equal(time.Now()) {
			log.Printf("couldn't parse date %s", item.PubDate)
		}

		// create new post in db with params from RSS feed item
		err = q.CreatePost(ctx, database.CreatePostParams{
			ID:        postID,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			Title:     item.Title,
			Url:       item.Link,
			Description: sql.NullString{
				String: item.Description,       //post content/description
				Valid:  item.Description != "", // true if description exists
			},
			PublishedAt: publishedAt,
			FeedID: uuid.NullUUID{
				UUID:  feed.ID, // links post to parent feed
				Valid: true,    // feed ID should always be valid
			},
		})
		if err != nil {
			if strings.Contains(err.Error(), "unique constraint") {
				// silently ignore duplicate URLs
				continue
			}
			// log any other type of error but don't stop processing
			log.Printf("failed to create post: %v", err)
			continue
		}

	}
	return nil
}

// ------------ main --------------

func main() {
	cfg, err := config.Read() // read config
	if err != nil {
		fmt.Println("Error reading config:", err)
		return
	}

	db, err := sql.Open("postgres", cfg.DbUrl) // load db url to config struct and sql.Open() a connection to db
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	dbQueries := database.New(db) // create a new *database.Queries

	s := &state{ // Create state instance to hold config and store *database.Queries in state struct
		db:        dbQueries,
		configPTR: &cfg, // provides handlers access to config
	}

	cmds := &commands{ // Create command system
		handlers: make(map[string]func(*state, command) error), // Initialize map that will store command handlers
	}

	// register command handlers in commands map
	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("feeds", handlerFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerGetFeedFollowsForUser))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollowFeedForUser))
	cmds.register("browse", middlewareLoggedIn(handlerBrowse))

	if len(os.Args) < 2 { // check if sufficient args (program name + command)
		fmt.Println("Error: not enough arguments")
		os.Exit(1)
	}

	cmd := command{ // Create command struct from CLI args
		name: os.Args[1],  // ex: "login"
		args: os.Args[2:], // ex: ["alice"]
	}

	// Run the command, look up handler in map and execute it
	if err := cmds.run(s, cmd); err != nil {
		fmt.Println("Error:", err)

		// Check if the error is due to an unrecognized command.
		// If so, provide a list of available commands
		if err.Error() == fmt.Sprintf("command %q does not exist", cmd.name) {
			fmt.Println("\nAvailable commands:")
			for name := range cmds.handlers {
				fmt.Printf("  - %s\n", name)
			}
		}
		os.Exit(1)
	}
}
