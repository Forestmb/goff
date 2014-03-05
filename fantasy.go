// Package goff provides a basic Yahoo Fantasy Sports API client.
//
// This package is designed to facilitate communication with the Yahoo Fantasy
// Sports API. The steps required to get a new client up and running are as
// follows:
//
//    1. Obtain an API key for your application.
//         See https://developer.apps.yahoo.com/dashboard/createKey.html
//    2. Call goff.GetConsumer(clientID, clientSecret) using your client's
//       information.
//    3. Use oauth.Consumer to obtain an oauth.AccessToken.
//         See https://godoc.org/github.com/mrjones/oauth
//    4. Call goff.NewOAuthClient(consumer, accessToken) with the consumer and
//       access token.
//    5. Use the returned client to make direct API requests with
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

	"github.com/mrjones/oauth"
)

//
// API Access Definitions
//

const (
	// NflGameKey represents the current year's Yahoo fantasy football game
	NflGameKey = "nfl"

	// YahooBaseURL is the base URL for all calls to Yahoo's fantasy sports API
	YahooBaseURL = "http://fantasysports.yahooapis.com/fantasy/v2"

	// YahooRequestTokenURL is used to create OAuth request tokens
	YahooRequestTokenURL = "https://api.login.yahoo.com/oauth/v2/get_request_token"

	// YahooAuthTokenURL is used to create OAuth authorization tokens
	YahooAuthTokenURL = "https://api.login.yahoo.com/oauth/v2/request_auth"

	// YahooGetTokenURL is used to get the OAuth access token used when making
	// calls to the fantasy sports API.
	YahooGetTokenURL = "https://api.login.yahoo.com/oauth/v2/get_token"
)

// yearKeys is map of a string year to the string Yahoo uses to identify the
// fantasy football game for that year.
var yearKeys = map[string]string{
	"nfl":  NflGameKey,
	"2013": NflGameKey,
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
	// The amount of requests made to the Yahoo API on behalf of the application
	// represented by this Client.
	RequestCount int

	// Provides fantasy content for this application.
	provider ContentProvider
}

// ContentProvider returns the data from an API request.
type ContentProvider interface {
	Get(url string) (content *FantasyContent, err error)
}

// xmlContentProvider implements ContentProvider and translates XML responses
// from an httpClient into the appropriate data.
type xmlContentProvider struct {
	// Makes HTTP requests to the API
	client httpClient
}

// httpClient defines methods needed to communicate with the Yahoo fantasy
// sports API over HTTP
type httpClient interface {
	// Makes HTTP request to the API
	Get(url string) (response *http.Response, err error)
}

// oauthHTTPClient implements httpClient using OAuth 1.0 for authentication
type oauthHTTPClient struct {
	token    *oauth.AccessToken
	consumer OAuthConsumer
}

// OAuthConsumer returns data from an oauth provider
type OAuthConsumer interface {
	Get(url string, data map[string]string, token *oauth.AccessToken) (*http.Response, error)
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
	LeagueKey   string   `xml:"league_key"`
	LeagueID    uint64   `xml:"league_id"`
	Name        string   `xml:"name"`
	Players     []Player `xml:"players>player"`
	Teams       []Team   `xml:"teams>team"`
	CurrentWeek int      `xml:"current_week"`
	IsFinished  bool     `xml:"is_finished"`
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
	TeamStandings         TeamStandings `xml:"team_standings"`
	Players               []Player      `xml:"players>player"`
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
	Guid           string `xml:"guid"`
	IsCurrentLogin bool   `xml:"is_current_login"`
}

// Points represents scoring statistics for a time period specified by
// CoverageType.
type Points struct {
	CoverageType string  `xml:"coverage_type"`
	Season       string  `xml:"season"`
	Week         uint64  `xml:"week"`
	Total        float64 `xml:"total"`
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
	Rank          int     `xml:"rank"`
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
//

// NewOAuthClient creates a Client that uses oauth authentication to communicate with
// the Yahoo fantasy sports API. The consumer can be created with `GetConsumer` and
// then used to obtain the access token passed in here.
func NewOAuthClient(consumer OAuthConsumer, accessToken *oauth.AccessToken) *Client {
	return &Client{
		provider: &xmlContentProvider{
			client: &oauthHTTPClient{
				token:    accessToken,
				consumer: consumer,
			},
		},
		RequestCount: 0,
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

//
// ContentProvider
//

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

	return &content, nil
}

//
// httpClient
//

// Get returns the HTTP response of a GET request to the given URL.
func (o *oauthHTTPClient) Get(url string) (*http.Response, error) {
	return o.consumer.Get(url, map[string]string{}, o.token)
}

//
// Yahoo interface
//

// GetFantasyContent directly access Yahoo fantasy resources.
//
// See http://developer.yahoo.com/fantasysports/guide/ for more information
func (c *Client) GetFantasyContent(url string) (*FantasyContent, error) {
	c.RequestCount++
	return c.provider.Get(url)
}

//
// Convenience functions
//

// GetUserLeagues returns a list of the current user's leagues for the given
// year.
func (c *Client) GetUserLeagues(year string) ([]League, error) {
	yearKey, ok := yearKeys[year]
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

	if len(content.Users[0].Games) == 0 {
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
