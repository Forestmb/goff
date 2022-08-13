// Package goff provides a basic Yahoo Fantasy Sports API client.
//
// This package is designed to facilitate communication with the Yahoo Fantasy
// Sports API. It is recommended, but not required, to use the
// golang.org/x/oauth2 package to generate a HTTP client to make authenticated
// API request. The steps required to get a new client up and running with this
// package are as follows:
//
//    1. Obtain an API key for your application.
//         See https://developer.apps.yahoo.com/dashboard/createKey.html
//    2. Call goff.GetOAuth2Config(clientId, clientSecret, redirectURL) using
//       your client's information.
//    3. Use oath2.Config to obtain an oauth2.Token.
//         See https://godoc.org/golang.org/x/oauth2#example-Config
//    4. Call oauth2Config.Client(ctx, token) with the config and access token.
//    5. Pass the returned http.Client into goff.NewClient.
//    6. Use the returned goff.Client to make direct API requests with
//       GetFantasyContent(url) or through one of the convenience methods.
//         See http://developer.yahoo.com/fantasysports/guide/ for the type
//         requests that can be made.
//
// To use OAuth 1.0 for authentication, use:
//
//    1. Obtain an API key for your application.
//         See https://developer.apps.yahoo.com/dashboard/createKey.html
//    2. Call goff.GetConsumer(clientID, clientSecret) using your client's
//       information.
//    3. Use oauth.Consumer to obtain an oauth.AccessToken.
//         See https://godoc.org/github.com/mrjones/oauth
//    4. Call oauthConsumer.MakeHttpClient(accessToken) with the consumer and
//       access token.
//    5. Pass the returned http.Client into goff.NewClient.
//    6. Use the returned goff.Client to make direct API requests with
//       GetFantasyContent(url) or through one of the convenience methods.
//         See http://developer.yahoo.com/fantasysports/guide/ for the type
//         requests that can be made.
//
// The goff client is currently in early stage development and the API is
// subject to change at any moment.
package goff

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mrjones/oauth"
	"golang.org/x/oauth2"
	lru "vitess.io/vitess/go/cache"
)

//
// API Access Definitions
//

const (
	// NflGameKey represents the current year's Yahoo fantasy football game
	NflGameKey = "nfl"

	// YahooBaseURL is the base URL for all calls to Yahoo's fantasy sports API
	YahooBaseURL = "https://fantasysports.yahooapis.com/fantasy/v2"

	// YahooRequestTokenURL is used to create OAuth request tokens
	YahooRequestTokenURL = "https://api.login.yahoo.com/oauth/v2/get_request_token"

	// YahooAuthTokenURL is used to create OAuth authorization tokens
	YahooAuthTokenURL = "https://api.login.yahoo.com/oauth/v2/request_auth"

	// YahooGetTokenURL is used to get the OAuth access token used when making
	// calls to the fantasy sports API.
	YahooGetTokenURL = "https://api.login.yahoo.com/oauth/v2/get_token"

	// YahooOauth2AuthURL is uesd to start the OAuth 2 login process.
	YahooOauth2AuthURL = "https://api.login.yahoo.com/oauth2/request_auth"

	// YahooOauth2TokenURL is used to create OAuth 2 access tokens used when
	// making calls to the fantasy sports API.
	YahooOauth2TokenURL = "https://api.login.yahoo.com/oauth2/get_token"
)

// ErrAccessDenied is returned when the user does not have permision to
// access the requested resource.
var ErrAccessDenied = errors.New(
	"user does not have permission to access the requested resource")

// YearKeys is map of a string year to the string Yahoo uses to identify the
// fantasy football game for that year.
var YearKeys = map[string]string{
	"nfl":  NflGameKey,
	"2022": "414",
	"2021": "406",
	"2020": "399",
	"2019": "390",
	"2018": "380",
	"2017": "371",
	"2016": "359",
	"2015": "348",
	"2014": "331",
	"2013": "314",
	"2012": "273",
	"2011": "257",
	"2010": "242",
	"2009": "222",
	"2008": "199",
	"2007": "175",
	"2006": "153",
	"2005": "124",
	"2004": "101",
	"2003": "79",
	"2002": "49",
	"2001": "57",
}

//
// Client
//

// Client is an application authorized to use the Yahoo fantasy sports API.
type Client struct {
	// Provides fantasy content for this application.
	Provider ContentProvider
}

// ContentProvider returns the data from an API request.
type ContentProvider interface {
	Get(url string) (content *FantasyContent, err error)
	// The amount of requests made to the Yahoo API on behalf of the application
	// represented by this Client.
	RequestCount() int
}

// Cache sets and retrieves fantasy content for request URLs based on the time
// for which the content was valid
type Cache interface {
	// Sets the content retrieved for the URL at the given time
	Set(url string, time time.Time, content *FantasyContent)

	// Gets the content for the URL given a time for which the content should
	// be valid
	Get(url string, time time.Time) (content *FantasyContent, ok bool)
}

// LRUCache implements Cache utilizing a LRU cache and unique keys to cache
// content for up to a maximum duration.
type LRUCache struct {
	ClientID        string
	Duration        time.Duration
	DurationSeconds int64
	Cache           *lru.LRUCache
}

// LRUCacheValue implements lru.Value to be able to store fantasy content in
// a LRUCache
type LRUCacheValue struct {
	content *FantasyContent
}

// cachedContentProvider implements ContentProvider and caches data from
// another ContentProvider for a period of time up to a maximum duration.
type cachedContentProvider struct {
	delegate ContentProvider
	cache    Cache
}

// xmlContentProvider implements ContentProvider and translates XML responses
// from an httpAPIClient into the appropriate data.
type xmlContentProvider struct {
	// Makes HTTP requests to the API
	client httpAPIClient
}

// httpAPIClient defines methods needed to communicate with the Yahoo fantasy
// sports API over HTTP
type httpAPIClient interface {
	// Makes HTTP request to the API
	Get(url string) (response *http.Response, err error)
	// Get the amount of requests made to the API
	RequestCount() int
}

// HTTPClient defines methods needed to communicated with a service over HTTP
type HTTPClient interface {
	// Makes a HTTP GET request
	Get(url string) (response *http.Response, err error)
}

// countingHTTPApiClient implements httpAPIClient
type countingHTTPApiClient struct {
	client       HTTPClient
	requestCount int
}

//
// API Data Structure Definitions
//

// FantasyContent is the root level response containing the data from a request
// to the fantasy sports API.
type FantasyContent struct {
	XMLName xml.Name `xml:"fantasy_content"`
	League  League   `xml:"league"`
	Team    Team     `xml:"team"`
	Users   []User   `xml:"users>user"`
}

// User contains the games a user is participating in
type User struct {
	Games []Game `xml:"games>game"`
}

// Game represents a single year in the Yahoo fantasy football ecosystem. It consists
// of zero or more leagues.
type Game struct {
	Leagues []League `xml:"leagues>league"`
}

// A League is a uniquely identifiable group of players and teams. The scoring system,
// roster details, and other metadata can differ between leagues.
type League struct {
	LeagueKey   string     `xml:"league_key"`
	LeagueID    uint64     `xml:"league_id"`
	Name        string     `xml:"name"`
	URL         string     `xml:"url"`
	Players     []Player   `xml:"players>player"`
	Teams       []Team     `xml:"teams>team"`
	DraftStatus string     `xml:"draft_status"`
	CurrentWeek int        `xml:"current_week"`
	StartWeek   int        `xml:"start_week"`
	EndWeek     int        `xml:"end_week"`
	IsFinished  bool       `xml:"is_finished"`
	Standings   []Team     `xml:"standings>teams>team"`
	Scoreboard  Scoreboard `xml:"scoreboard"`
	Settings    Settings   `xml:"settings"`
}

// A Team is a participant in exactly one league.
type Team struct {
	TeamKey               string        `xml:"team_key"`
	TeamID                uint64        `xml:"team_id"`
	Name                  string        `xml:"name"`
	URL                   string        `xml:"url"`
	TeamLogos             []TeamLogo    `xml:"team_logos>team_logo"`
	IsOwnedByCurrentLogin bool          `xml:"is_owned_by_current_login"`
	WavierPriority        int           `xml:"waiver_priority"`
	NumberOfMoves         int           `xml:"number_of_moves"`
	NumberOfTrades        int           `xml:"number_of_trades"`
	Managers              []Manager     `xml:"managers>manager"`
	Matchups              []Matchup     `xml:"matchups>matchup"`
	Roster                Roster        `xml:"roster"`
	TeamPoints            Points        `xml:"team_points"`
	TeamProjectedPoints   Points        `xml:"team_projected_points"`
	TeamStandings         TeamStandings `xml:"team_standings"`
	Players               []Player      `xml:"players>player"`
}

// Settings describes how a league is configured
type Settings struct {
	DraftType        string `xml:"draft_type"`
	ScoringType      string `xml:"scoring_type"`
	UsesPlayoff      bool   `xml:"uses_playoff"`
	PlayoffStartWeek int    `xml:"playoff_start_week"`
}

// Scoreboard represents the matchups that occurred for one or more weeks.
type Scoreboard struct {
	Weeks    string    `xml:"week"`
	Matchups []Matchup `xml:"matchups>matchup"`
}

// A Roster is the set of players belonging to one team for a given week.
type Roster struct {
	CoverageType string   `xml:"coverage_type"`
	Players      []Player `xml:"players>player"`
	Week         int      `xml:"week"`
}

// A Matchup is a collection of teams paired against one another for a given
// week.
type Matchup struct {
	Week  int    `xml:"week"`
	Teams []Team `xml:"teams>team"`
}

// A Manager is a user in change of a given team.
type Manager struct {
	ManagerID      uint64 `xml:"manager_id"`
	Nickname       string `xml:"nickname"`
	GUID           string `xml:"guid"`
	IsCurrentLogin bool   `xml:"is_current_login"`
}

// Points represents scoring statistics for a time period specified by
// CoverageType.
type Points struct {
	CoverageType string `xml:"coverage_type"`
	Season       string `xml:"season"`
	Week         int    `xml:"week"`
	Total        float64
	TotalStr     string `xml:"total"`
}

// Record is the number of wins, losses, and ties for a given team in their
// league.
type Record struct {
	Wins   int `xml:"wins"`
	Losses int `xml:"losses"`
	Ties   int `xml:"ties"`
}

// TeamStandings describes how a single Team ranks in their league.
type TeamStandings struct {
	Rank          int
	RankStr       string  `xml:"rank"`
	Record        Record  `xml:"outcome_totals"`
	PointsFor     float64 `xml:"points_for"`
	PointsAgainst float64 `xml:"points_against"`
}

// TeamLogo is a image for a given team.
type TeamLogo struct {
	Size string `xml:"size"`
	URL  string `xml:"url"`
}

// A Player is a single player for the given sport.
type Player struct {
	PlayerKey          string           `xml:"player_key"`
	PlayerID           uint64           `xml:"player_id"`
	Name               Name             `xml:"name"`
	DisplayPosition    string           `xml:"display_position"`
	ElligiblePositions []string         `xml:"elligible_positions>position"`
	SelectedPosition   SelectedPosition `xml:"selected_position"`
	PlayerPoints       Points           `xml:"player_points"`
}

// SelectedPosition is the position chosen for a Player for a given week.
type SelectedPosition struct {
	CoverageType string `xml:"coverage_type"`
	Week         int    `xml:"week"`
	Position     string `xml:"position"`
}

// Name is a name of a player.
type Name struct {
	Full  string `xml:"full"`
	First string `xml:"first"`
	Last  string `xml:"last"`
}

//
// Client

// NewCachedClient creates a new fantasy client that checks and updates the
// given Cache when retrieving fantasy content.
//
// See NewLRUCache
func NewCachedClient(cache Cache, client HTTPClient) *Client {
	return &Client{
		Provider: &cachedContentProvider{
			delegate: NewClient(client).Provider,
			cache:    cache,
		},
	}
}

// NewClient creates a Client that to communicate with the Yahoo fantasy
// sports API. See the package level documentation for one way to create a
// http.Client that can authenticate with Yahoo's APIs which can be passed
// in here.
func NewClient(c HTTPClient) *Client {
	return &Client{
		Provider: &xmlContentProvider{
			client: &countingHTTPApiClient{
				client:       c,
				requestCount: 0,
			},
		},
	}
}

// GetConsumer generates an OAuth Consumer for the Yahoo fantasy sports API
func GetConsumer(clientID string, clientSecret string) *oauth.Consumer {
	return oauth.NewConsumer(
		clientID,
		clientSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   YahooRequestTokenURL,
			AuthorizeTokenUrl: YahooAuthTokenURL,
			AccessTokenUrl:    YahooGetTokenURL,
		})
}

// GetOAuth2Config generates an OAuth 2 configuration for the Yahoo fantasy
// sports API
func GetOAuth2Config(clientID string, clientSecret string, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"fspt-r"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  YahooOauth2AuthURL,
			TokenURL: YahooOauth2TokenURL,
		},
	}
}

// RequestCount returns the amount of requests made to the Yahoo API on behalf
// of the application represented by this Client.
func (c *Client) RequestCount() int {
	return c.Provider.RequestCount()
}

//
// Cache
//

// NewLRUCache creates a new Cache that caches content for the given client
// for up to the maximum duration.
//
// See NewCachedClient
func NewLRUCache(
	clientID string,
	duration time.Duration,
	cache *lru.LRUCache) *LRUCache {

	return &LRUCache{
		ClientID:        clientID,
		Duration:        duration,
		DurationSeconds: int64(duration.Seconds()),
		Cache:           cache,
	}
}

// Set specifies that the given content was retrieved for the given URL at the
// given time. The content for that URL will be available by LRUCache.Get from
// the given 'time' up to 'time + l.Duration'
func (l *LRUCache) Set(url string, time time.Time, content *FantasyContent) {
	l.Cache.Set(l.getKey(url, time), &LRUCacheValue{content: content})
}

// Get the content for the given URL at the given time.
func (l *LRUCache) Get(url string, time time.Time) (content *FantasyContent, ok bool) {
	value, ok := l.Cache.Get(l.getKey(url, time))
	if !ok {
		return nil, ok
	}
	lruCacheValue, ok := value.(*LRUCacheValue)
	if !ok {
		return nil, ok
	}
	return lruCacheValue.content, true
}

// getKey converts a base key to a key that is unique for the client of the
// LRUCache and the current time period.
//
// The created keys have the following format:
//
//    <client-id>:<originalKey>:<period>
//
// Given a client with ID "client-id-01", original key of "key-01", a current
// time of "08/17/2014 1:21pm", and a maximum cache duration of 1 hour, this
// will generate the following key:
//
//    client-id-01:key-01:391189
//
func (l *LRUCache) getKey(originalKey string, time time.Time) string {
	period := time.Unix() / l.DurationSeconds
	return fmt.Sprintf("%s:%s:%d", l.ClientID, originalKey, period)
}

// Size always returns '1'. All LRU cache values have the same size, meaning
// the backing lru.LRUCache will prune strictly based on number of cached
// content and not the total size in memory.
func (v *LRUCacheValue) Size() int {
	return 1
}

//
// ContentProvider
//

func (p *cachedContentProvider) Get(url string) (*FantasyContent, error) {
	currentTime := time.Now()
	content, ok := p.cache.Get(url, currentTime)
	if !ok {
		content, err := p.delegate.Get(url)
		if err == nil {
			p.cache.Set(url, currentTime, content)
		}
		return content, err
	}
	return content, nil
}

func (p *cachedContentProvider) RequestCount() int {
	return p.delegate.RequestCount()
}

func (p *xmlContentProvider) Get(url string) (*FantasyContent, error) {
	response, err := p.client.Get(url)

	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	bits, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var content FantasyContent
	err = xml.Unmarshal(bits, &content)
	if err != nil {
		return nil, err
	}

	return fixContent(&content), nil
}

// fixContent updates the fantasy data with content that can't be unmarshalled
// directly from XML
func fixContent(c *FantasyContent) *FantasyContent {
	fixTeam(&c.Team)
	for i := range c.League.Teams {
		fixTeam(&c.League.Teams[i])
	}
	for i := range c.League.Standings {
		fixTeam(&c.League.Standings[i])
	}
	for i := range c.League.Players {
		fixPoints(&c.League.Players[i].PlayerPoints)
	}
	for i := range c.League.Scoreboard.Matchups {
		fixTeam(&c.League.Scoreboard.Matchups[i].Teams[0])
		fixTeam(&c.League.Scoreboard.Matchups[i].Teams[1])
	}
	return c
}

func fixTeam(t *Team) {
	fixPoints(&t.TeamPoints)
	fixPoints(&t.TeamProjectedPoints)
	for i := range t.Roster.Players {
		fixPoints(&t.Roster.Players[i].PlayerPoints)
	}
	for i := range t.Players {
		fixPoints(&t.Players[i].PlayerPoints)
	}
	for i := range t.Matchups {
		fixTeam(&t.Matchups[i].Teams[0])
		fixTeam(&t.Matchups[i].Teams[1])
	}
	fixRank(&t.TeamStandings)
}

func fixRank(t *TeamStandings) {
	if t.RankStr != "" {
		rank, err := strconv.ParseInt(t.RankStr, 10, 64)
		if err == nil {
			t.Rank = int(rank)
		}
	}
}

func fixPoints(p *Points) {
	if p.TotalStr != "" {
		total, err := strconv.ParseFloat(p.TotalStr, 64)
		if err == nil {
			p.Total = total
		}
	}
}

func (p *xmlContentProvider) RequestCount() int {
	return p.client.RequestCount()
}

//
// httpAPIClient
//

// Get returns the HTTP response of a GET request to the given URL.
func (o *countingHTTPApiClient) Get(url string) (*http.Response, error) {
	o.requestCount++
	response, err := o.client.Get(url)

	// Known issue where "consumer_key_unknown" is returned for valid
	// consumer keys. If this happens, try re-requesting the content a few
	// times to see if it fixes itself
	//
	// See https://developer.yahoo.com/forum/OAuth-General-Discussion-YDN-SDKs/oauth-problem-consumer-key-unknown-/1375188859720-5cea9bdb-0642-4606-9fd5-c5f369112959
	for attempts := 0; attempts < 4 &&
		err != nil &&
		strings.Contains(err.Error(), "consumer_key_unknown"); attempts++ {

		o.requestCount++
		response, err = o.client.Get(url)
	}

	if err != nil &&
		strings.Contains(
			err.Error(),
			"You are not allowed to view this page") {
		err = ErrAccessDenied
	}

	return response, err
}

func (o *countingHTTPApiClient) RequestCount() int {
	return o.requestCount
}

//
// Yahoo interface
//

// GetFantasyContent directly access Yahoo fantasy resources.
//
// See http://developer.yahoo.com/fantasysports/guide/ for more information
func (c *Client) GetFantasyContent(url string) (*FantasyContent, error) {
	return c.Provider.Get(url)
}

//
// Convenience functions
//

// GetUserLeagues returns a list of the current user's leagues for the given
// year.
func (c *Client) GetUserLeagues(year string) ([]League, error) {
	yearKey, ok := YearKeys[year]
	if !ok {
		return nil, fmt.Errorf("data not available for year=%s", year)
	}
	content, err := c.GetFantasyContent(
		fmt.Sprintf("%s/users;use_login=1/games;game_keys=%s/leagues",
			YahooBaseURL,
			yearKey))

	if err != nil {
		return nil, err
	}

	if len(content.Users) == 0 {
		return nil, errors.New("no users returned for current user")
	}

	if len(content.Users[0].Games) == 0 ||
		content.Users[0].Games[0].Leagues == nil {
		return make([]League, 0), nil
	}

	return content.Users[0].Games[0].Leagues, nil
}

// GetPlayersStats returns a list of Players containing their stats for the
// given week in the given year.
func (c *Client) GetPlayersStats(leagueKey string, week int, players []Player) ([]Player, error) {
	playerKeys := ""
	for index, player := range players {
		if index != 0 {
			playerKeys += ","
		}
		playerKeys += player.PlayerKey
	}

	content, err := c.GetFantasyContent(
		fmt.Sprintf("%s/league/%s/players;player_keys=%s/stats;type=week;week=%d",
			YahooBaseURL,
			leagueKey,
			playerKeys,
			week))

	if err != nil {
		return nil, err
	}
	return content.League.Players, nil
}

// GetTeamRoster returns a team's roster for the given week.
func (c *Client) GetTeamRoster(teamKey string, week int) ([]Player, error) {
	content, err := c.GetFantasyContent(
		fmt.Sprintf("%s/team/%s/roster;week=%d",
			YahooBaseURL,
			teamKey,
			week))
	if err != nil {
		return nil, err
	}

	return content.Team.Roster.Players, nil
}

// GetLeagueStandings gets a league containing the current standings.
func (c *Client) GetLeagueStandings(leagueKey string) (*League, error) {
	content, err := c.GetFantasyContent(
		fmt.Sprintf("%s/league/%s;out=standings,settings",
			YahooBaseURL,
			leagueKey))
	if err != nil {
		return nil, err
	}
	return &content.League, nil
}

// GetAllTeamStats gets teams stats for a given week.
func (c *Client) GetAllTeamStats(leagueKey string, week int) ([]Team, error) {
	content, err := c.GetFantasyContent(
		fmt.Sprintf("%s/league/%s/teams/stats;type=week;week=%d",
			YahooBaseURL,
			leagueKey,
			week))
	if err != nil {
		return nil, err
	}

	return content.League.Teams, nil
}

// GetTeam returns all available information about the given team.
func (c *Client) GetTeam(teamKey string) (*Team, error) {
	content, err := c.GetFantasyContent(
		fmt.Sprintf("%s/team/%s;out=stats,metadata,players,standings,roster",
			YahooBaseURL,
			teamKey))
	if err != nil {
		return nil, err
	}

	if content.Team.TeamID == 0 {
		return nil, fmt.Errorf("no team returned for key='%s'", teamKey)
	}
	return &content.Team, nil
}

// GetLeagueMetadata returns the metadata associated with the given league.
func (c *Client) GetLeagueMetadata(leagueKey string) (*League, error) {
	content, err := c.GetFantasyContent(
		fmt.Sprintf("%s/league/%s/metadata",
			YahooBaseURL,
			leagueKey))
	if err != nil {
		return nil, err
	}
	return &content.League, nil
}

// GetAllTeams returns all teams playing in the given league.
func (c *Client) GetAllTeams(leagueKey string) ([]Team, error) {
	content, err := c.GetFantasyContent(
		fmt.Sprintf("%s/league/%s/teams", YahooBaseURL, leagueKey))
	if err != nil {
		return nil, err
	}
	return content.League.Teams, nil
}

// GetMatchupsForWeekRange returns a list of matchups for each week in the
// requested range.
func (c *Client) GetMatchupsForWeekRange(leagueKey string, startWeek, endWeek int) (map[int][]Matchup, error) {
	leagueList := strconv.Itoa(startWeek)
	for i := startWeek + 1; i <= endWeek; i++ {
		leagueList += "," + strconv.Itoa(i)
	}
	content, err := c.GetFantasyContent(
		fmt.Sprintf("%s/league/%s/scoreboard;week=%s",
			YahooBaseURL,
			leagueKey,
			leagueList))
	if err != nil {
		return nil, err
	}

	all := make(map[int][]Matchup)
	for _, matchup := range content.League.Scoreboard.Matchups {
		week := matchup.Week
		list, ok := all[week]
		if !ok {
			list = make([]Matchup, 0)
		}
		all[week] = append(list, matchup)
	}
	return all, nil
}
