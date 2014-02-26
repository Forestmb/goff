package goff

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mrjones/oauth"
)

//
// Test NewOAuthClient
//

func TestNewOAuthClient(t *testing.T) {
	clientID := "clientID"
	clientSecret := "clientSecret"
	consumer := GetConsumer(clientID, clientSecret)

	client := NewOAuthClient(consumer, &oauth.AccessToken{})

	if client == nil {
		t.Fatal("No client returned")
	}

	if client.RequestCount != 0 {
		t.Fatalf("Invalid request count after initialization\n"+
			"\texpected: 0\n\tactual: %d",
			client.RequestCount)
	}
}

//
// Test GetConsumer
//

func TestGetConsumer(t *testing.T) {
	clientID := "clientID"
	clientSecret := "clientSecret"
	consumer := GetConsumer(clientID, clientSecret)
	if consumer == nil {
		t.Fatal("No consumer returned")
	}
}

//
// Test oauthHttpClient
//

func TestOAuthHttpClient(t *testing.T) {
	expected := &http.Response{}
	client := &oauthHttpClient{
		token: &oauth.AccessToken{},
		consumer: &mockOAuthConsumer{
			Response: expected,
			Error:    nil,
		},
	}

	response, err := client.Get("http://example.com")
	if err != nil {
		t.Fatalf("error retrieving response: %s", err)
	}

	if response != expected {
		t.Fatalf("received unexpected response from client")
	}
}

func TestOAuthHttpClientError(t *testing.T) {
	client := &oauthHttpClient{
		token: &oauth.AccessToken{},
		consumer: &mockOAuthConsumer{
			Response: &http.Response{},
			Error:    errors.New("error"),
		},
	}

	_, err := client.Get("http://example.com")
	if err == nil {
		t.Fatalf("no error returned from client when consumer failed")
	}
}

//
// Test xmlContentProvider
//

func TestXmlContentProviderGetLeague(t *testing.T) {
	response := mockResponse(leagueXmlContent)
	client := &oauthHttpClient{
		token: &oauth.AccessToken{},
		consumer: &mockOAuthConsumer{
			Response: response,
			Error:    nil,
		},
	}

	provider := &xmlContentProvider{client: client}
	content, err := provider.Get("http://example.com")

	if err != nil {
		t.Fatalf("unexpected error returned: %s", err)
	}

	league := content.League
	if league.LeagueKey != "223.l.431" ||
		league.LeagueID != 431 ||
		league.Name != "League Name" ||
		league.CurrentWeek != 16 ||
		league.IsFinished != true {

		t.Fatalf("unexpected league content returned\n"+
			"\tcontent: %+v", league)
	}
}

func TestXmlContentProviderGetTeam(t *testing.T) {
	response := mockResponse(teamXmlContent)
	client := &oauthHttpClient{
		token: &oauth.AccessToken{},
		consumer: &mockOAuthConsumer{
			Response: response,
			Error:    nil,
		},
	}

	provider := &xmlContentProvider{client: client}
	content, err := provider.Get("http://example.com")

	if err != nil {
		t.Fatalf("unexpected error returned: %s", err)
	}

	team := content.Team
	if team.TeamKey != "223.l.431.t.1" ||
		team.TeamID != 1 ||
		team.Name != "Team Name" ||
		team.Managers.List[0].ManagerID != 13 ||
		team.Managers.List[0].Nickname != "Nickname" ||
		team.Managers.List[0].Guid != "1234567890" {

		t.Fatalf("unexpected team content returned\n"+
			"\tcontent: %+v", team)
	}
}

func TestXmlContentProviderGetError(t *testing.T) {
	response := mockResponse("content")
	client := &oauthHttpClient{
		token: &oauth.AccessToken{},
		consumer: &mockOAuthConsumer{
			Response: response,
			Error:    errors.New("error"),
		},
	}

	provider := &xmlContentProvider{client: client}
	_, err := provider.Get("http://example.com")

	if err == nil {
		t.Fatalf("error not returned when consumer fails")
	}
}

func TestXmlContentProviderReadError(t *testing.T) {
	response := mockResponseReadErr()
	client := &oauthHttpClient{
		token: &oauth.AccessToken{},
		consumer: &mockOAuthConsumer{
			Response: response,
		},
	}

	provider := &xmlContentProvider{client: client}
	_, err := provider.Get("http://example.com")

	if err == nil {
		t.Fatalf("error not returned when read fails")
	}
}

func TestXmlContentProviderParseError(t *testing.T) {
	response := mockResponse("<not-valid-xml/>")
	client := &oauthHttpClient{
		token: &oauth.AccessToken{},
		consumer: &mockOAuthConsumer{
			Response: response,
		},
	}

	provider := &xmlContentProvider{client: client}
	_, err := provider.Get("http://example.com")

	if err == nil {
		t.Fatalf("error not returned when parse fails")
	}
}

type mockReaderCloser struct {
	Reader    io.Reader
	ReadError error
	WasClosed bool
}

func mockResponse(content string) *http.Response {
	return &http.Response{
		Body: &mockReaderCloser{
			Reader:    strings.NewReader(content),
			WasClosed: false,
		},
	}
}

func mockResponseReadErr() *http.Response {
	return &http.Response{
		Body: &mockReaderCloser{
			ReadError: errors.New("error"),
			WasClosed: false,
		},
	}
}

func (m *mockReaderCloser) Read(p []byte) (n int, err error) {
	if m.ReadError != nil {
		return 0, m.ReadError
	}
	return m.Reader.Read(p)
}

func (m *mockReaderCloser) Close() error {
	m.WasClosed = true
	return nil
}

//
// Test GetFantasyContent
//

func TestGetFantasyContent(t *testing.T) {
	expectedContent := &FantasyContent{}
	client := mockClient(expectedContent, nil)
	actualContent, err := client.GetFantasyContent("http://example.com")
	if actualContent != expectedContent {
		t.Fatal("Actual content did not equal expected content\n"+
			"\texpected: %+v\n\tactual: %+v",
			expectedContent,
			actualContent)
	}

	if err != nil {
		t.Fatalf("Client returned error: %s", err)
	}
}

func TestGetFantasyContentError(t *testing.T) {
	expectedErr := errors.New("Error retreiving content")
	client := mockClient(nil, expectedErr)
	content, actualErr := client.GetFantasyContent("http://example.com")
	if content != nil {
		t.Fatalf("Fantasy client returned unexpected content: %+v", content)
	}

	if actualErr == nil {
		t.Fatal("Nil error returned.")
	}
}

func TestGetFantasyContentRequestcount(t *testing.T) {
	client := mockClient(&FantasyContent{}, nil)
	client.GetFantasyContent("http://example.com/RequestOne")
	if client.RequestCount != 1 {
		t.Fatalf("Fantasy client returned incorrect request count.\n"+
			"\texpected: 1\n\tactual: %d",
			client.RequestCount)
	}
	client.GetFantasyContent("http://example.com/RequestTwo")
	if client.RequestCount != 2 {
		t.Fatalf("Fantasy client returned incorrect request count.\n"+
			"\texpected: 2\n\tactual: %d",
			client.RequestCount)
	}
	client.GetFantasyContent("http://example.com/RequestOne")
	if client.RequestCount != 3 {
		t.Fatalf("Fantasy client returned incorrect request count.\n"+
			"\texpected: 3\n\tactual: %d",
			client.RequestCount)
	}
}

//
// Test GetUserLeagues
//

func TestGetUserLeagues(t *testing.T) {
	leagues := []League{
		League{
			LeagueKey: "key1",
			LeagueID:  1,
			Name:      "name1",
		},
	}
	content := createLeagueList(leagues...)
	client := mockClient(content, nil)
	l, err := client.GetUserLeagues("2013")
	if err != nil {
		t.Fatalf("Client returned error: %s", err)
	}

	assertLeaguesEqual(t, leagues, l)
}

func TestGetUserLeaguesError(t *testing.T) {
	content := createLeagueList(League{LeagueKey: "123"})
	client := mockClient(content, errors.New("error"))
	_, err := client.GetUserLeagues("2013")
	if err == nil {
		t.Fatal("Client did not return error")
	}
}

func TestGetUserLeaguesNoUsers(t *testing.T) {
	content := &FantasyContent{Users: Users{List: []User{}}}
	client := mockClient(content, nil)
	actual, err := client.GetUserLeagues("2013")
	if err == nil {
		t.Fatal("Client did not return error when no users were found\n"+
			"\tcontent: %+v",
			actual)
	}
}

func TestGetUserLeaguesNoGames(t *testing.T) {
	content := &FantasyContent{
		Users: Users{
			List: []User{
				User{
					Games: Games{
						List: []Game{},
					},
				},
			},
		},
	}
	client := mockClient(content, nil)
	actual, err := client.GetUserLeagues("2013")
	if err != nil {
		t.Fatalf("Client returned error: %s", err)
	}

	if len(actual) != 0 {
		t.Fatalf("Client returned leagues when no games exist: %+v", actual)
	}
}

func TestGetUserLeaguesNoLeagues(t *testing.T) {
	content := &FantasyContent{
		Users: Users{
			List: []User{
				User{
					Games: Games{
						List: []Game{
							Game{
								Leagues: Leagues{
									List: []League{},
								},
							},
						},
					},
				},
			},
		},
	}
	client := mockClient(content, nil)
	actual, err := client.GetUserLeagues("2013")
	if err != nil {
		t.Fatalf("Client returned unexpected error: %s", err)
	}

	if len(actual) != 0 {
		t.Fatal("Client should not have returned leagues\n"+
			"\tcontent: %+v",
			actual)
	}
}

func TestGetUserLeaguesMapsYear(t *testing.T) {
	content := createLeagueList(League{LeagueKey: "123"})
	provider := &mockedContentProvider{content: content, err: nil}
	client := &Client{
		RequestCount: 0,
		provider:     provider,
	}

	client.GetUserLeagues("2013")
	yearParam := "game_keys"
	assertUrlContainsParam(t, provider.lastGetUrl, yearParam, "nfl")

	year := "2010"
	client.GetUserLeagues(year)
	assertUrlContainsParam(t, provider.lastGetUrl, yearParam, yearKeys[year])

	_, err := client.GetUserLeagues("1900")
	if err == nil {
		t.Fatalf("no error returned for year not supported by yahoo")
	}
}

//
// Test GetTeam
//

func TestGetTeam(t *testing.T) {
	team := Team{
		TeamKey: "teamKey1",
		TeamID:  1,
		Name:    "name1",
	}
	client := mockClient(&FantasyContent{Team: team}, nil)

	actual, err := client.GetTeam(team.TeamKey)
	if err != nil {
		t.Fatalf("Client returned unexpected error: %s", err)
	}
	assertTeamsEqual(t, &team, actual)
}

func TestGetTeamError(t *testing.T) {
	team := Team{
		TeamKey: "teamKey1",
		TeamID:  1,
		Name:    "name1",
	}
	client := mockClient(&FantasyContent{Team: team}, errors.New("error"))

	_, err := client.GetTeam(team.TeamKey)
	if err == nil {
		t.Fatalf("Error not returned by client.")
	}
}

func TestGetTeamNoTeamFound(t *testing.T) {
	client := mockClient(&FantasyContent{}, nil)
	content, err := client.GetTeam("123")
	if err == nil {
		t.Fatalf("No error returned by client.\n\tcontent: %+v", content)
	}
}

//
// Test GetLeagueMetadata
//

func TestGetLeagueMetadata(t *testing.T) {
	league := League{
		LeagueKey:   "key1",
		LeagueID:    1,
		Name:        "name1",
		CurrentWeek: 2,
		IsFinished:  false,
	}

	client := mockClient(&FantasyContent{League: league}, nil)

	actual, err := client.GetLeagueMetadata(league.LeagueKey)
	if err != nil {
		t.Fatalf("Client returned unexpected error: %s", err)
	}

	assertLeaguesEqual(t, []League{league}, []League{*actual})
}

func TestGetLeagueMetadataError(t *testing.T) {
	league := League{
		LeagueKey:   "key1",
		LeagueID:    1,
		Name:        "name1",
		CurrentWeek: 2,
		IsFinished:  false,
	}

	client := mockClient(&FantasyContent{League: league}, errors.New("error"))

	_, err := client.GetLeagueMetadata(league.LeagueKey)
	if err == nil {
		t.Fatalf("Client did not return  error.")
	}
}

//
// Test GetPlayersStats
//

func TestGetPlayerStats(t *testing.T) {
	players := []Player{
		Player{
			PlayerKey: "key1",
			PlayerID:  1,
			Name: Name{
				Full:  "Firstname Lastname",
				First: "Firstname",
				Last:  "Lastname",
			},
		},
		Player{
			PlayerKey: "key2",
			PlayerID:  1,
			Name: Name{
				Full:  "Firstname2 Lastname2",
				First: "Firstname2",
				Last:  "Lastname2",
			},
		},
	}

	client := mockClient(&FantasyContent{
		League: League{
			Players: Players{
				List: players,
			},
		},
	},
		nil)

	week := 10
	actual, err := client.GetPlayersStats("123", week, players)
	if err != nil {
		t.Fatalf("Client returned unexpected error: %s", err)
	}
	assertPlayersEqual(t, &players[0], &actual[0])
	assertPlayersEqual(t, &players[1], &actual[1])
}

func TestGetPlayerStatsError(t *testing.T) {
	players := []Player{
		Player{
			PlayerKey: "key1",
			PlayerID:  1,
			Name: Name{
				Full:  "Firstname Lastname",
				First: "Firstname",
				Last:  "Lastname",
			},
		},
	}

	client := mockClient(&FantasyContent{
		League: League{
			Players: Players{
				List: players,
			},
		},
	},
		errors.New("error"))

	week := 10
	_, err := client.GetPlayersStats("123", week, players)
	if err == nil {
		t.Fatalf("Client did not return error")
	}
}

func TestGetPlayerStatsParams(t *testing.T) {
	players := []Player{
		Player{
			PlayerKey: "key1",
			PlayerID:  1,
			Name: Name{
				Full:  "Firstname Lastname",
				First: "Firstname",
				Last:  "Lastname",
			},
		},
	}

	provider := &mockedContentProvider{
		content: &FantasyContent{
			League: League{
				Players: Players{
					List: players,
				},
			},
		},
		err: nil,
	}
	client := &Client{
		RequestCount: 0,
		provider:     provider,
	}

	week := 10
	client.GetPlayersStats("123", week, players)

	assertUrlContainsParam(t, provider.lastGetUrl, "player_keys", players[0].PlayerKey)
	assertUrlContainsParam(t, provider.lastGetUrl, "week", fmt.Sprintf("%d", week))
}

//
// Test GetTeamRoster
//

func TestGetTeamRoster(t *testing.T) {
	players := []Player{
		Player{
			PlayerKey: "key1",
			PlayerID:  1,
			Name: Name{
				Full:  "Firstname Lastname",
				First: "Firstname",
				Last:  "Lastname",
			},
		},
	}

	client := mockClient(&FantasyContent{
		Team: Team{
			Roster: Roster{
				Players: Players{
					List: players,
				},
			},
		},
	},
		nil)
	actual, err := client.GetTeamRoster("123", 2)
	if err != nil {
		t.Fatalf("Client returned unexpected error: %s", err)
	}

	assertPlayersEqual(t, &players[0], &actual[0])
}

func TestGetTeamRosterError(t *testing.T) {
	players := []Player{
		Player{
			PlayerKey: "key1",
			PlayerID:  1,
			Name: Name{
				Full:  "Firstname Lastname",
				First: "Firstname",
				Last:  "Lastname",
			},
		},
	}

	client := mockClient(&FantasyContent{
		Team: Team{
			Roster: Roster{
				Players: Players{
					List: players,
				},
			},
		},
	},
		errors.New("error"))
	_, err := client.GetTeamRoster("123", 2)
	if err == nil {
		t.Fatalf("Client did not return error")
	}
}

//
// Test GetAllTeamStats
//

func TestGetAllTeamStats(t *testing.T) {
	team := Team{TeamKey: "key1", TeamID: 1, Name: "name1"}
	content := &FantasyContent{
		League: League{
			Teams: Teams{
				List: []Team{
					team,
				},
			},
		},
	}
	client := mockClient(content, nil)
	actual, err := client.GetAllTeamStats("123", 12)
	if err != nil {
		t.Fatalf("Client returned unexpected error: %s", err)
	}

	assertTeamsEqual(t, &team, &actual[0])
}

func TestGetAllTeamStatsError(t *testing.T) {
	team := Team{TeamKey: "key1", TeamID: 1, Name: "name1"}
	content := &FantasyContent{
		League: League{
			Teams: Teams{
				List: []Team{
					team,
				},
			},
		},
	}
	client := mockClient(content, errors.New("error"))
	actual, err := client.GetAllTeamStats("123", 12)
	if err == nil {
		t.Fatalf("Client did not return expected error\n\tcontent: %+v",
			actual)
	}
}

func TestGetAllTeamStatsParam(t *testing.T) {
	team := Team{TeamKey: "key1", TeamID: 1, Name: "name1"}
	content := &FantasyContent{
		League: League{
			Teams: Teams{
				List: []Team{
					team,
				},
			},
		},
	}
	week := 12
	provider := &mockedContentProvider{content: content, err: nil}
	client := &Client{provider: provider}
	client.GetAllTeamStats("123", week)
	assertUrlContainsParam(
		t,
		provider.lastGetUrl,
		"week",
		fmt.Sprintf("%d", week))
}

//
// Test GetAllTeams
//

func TestGetAllTeams(t *testing.T) {
	team := Team{TeamKey: "key1", TeamID: 1, Name: "name1"}
	content := &FantasyContent{
		League: League{
			Teams: Teams{
				List: []Team{
					team,
				},
			},
		},
	}
	client := mockClient(content, nil)
	actual, err := client.GetAllTeams("123")

	if err != nil {
		t.Fatalf("Client returned unexpected error: %s", err)
	}

	assertTeamsEqual(t, &team, &actual[0])
}

func TestGetAllTeamsError(t *testing.T) {
	team := Team{TeamKey: "key1", TeamID: 1, Name: "name1"}
	content := &FantasyContent{
		League: League{
			Teams: Teams{
				List: []Team{
					team,
				},
			},
		},
	}
	client := mockClient(content, errors.New("error"))
	actual, err := client.GetAllTeams("123")

	if err == nil {
		t.Fatalf("Client did not return expected error\n\tcontent: %+v",
			actual)
	}
}

//
// Assert
//

func assertPlayersEqual(t *testing.T, expected *Player, actual *Player) {
	if expected.PlayerKey != actual.PlayerKey ||
		expected.PlayerID != actual.PlayerID ||
		expected.Name.Full != actual.Name.Full {
		t.Fatalf("Actual player did not match expected player\n"+
			"\texpected: %+v\n\tactual:%+v",
			expected,
			actual)
	}
}

func assertUrlContainsParam(t *testing.T, url string, param string, value string) {
	if !strings.Contains(url, param+"="+value) {
		t.Fatalf("Could not locate paramater in request URL\n"+
			"\tparamter: %s\n\tvalue: %s\n\turl: %s",
			param,
			value,
			url)
	}
}

func assertTeamsEqual(t *testing.T, expectedTeam *Team, actualTeam *Team) {
	if expectedTeam.TeamKey != actualTeam.TeamKey ||
		expectedTeam.TeamID != actualTeam.TeamID ||
		expectedTeam.Name != actualTeam.Name {
		t.Fatalf("Actual team does not equal expected team\n"+
			"\texpected: %+v\n\tactual: %+v",
			expectedTeam,
			actualTeam)
	}
}

func assertLeaguesEqual(t *testing.T, expectedLeagues []League, actualLeagues []League) {
	for i := range expectedLeagues {
		if expectedLeagues[i].LeagueKey != actualLeagues[i].LeagueKey ||
			expectedLeagues[i].LeagueID != actualLeagues[i].LeagueID ||
			expectedLeagues[i].Name != actualLeagues[i].Name {
			t.Fatalf("Actual league did not equal expected league.\n"+
				"\texpected: %+v\n\tactual: %+v",
				expectedLeagues[i],
				actualLeagues[i])
		}
	}
}

//
// Mocks
//

func createLeagueList(leagues ...League) *FantasyContent {
	return &FantasyContent{
		Users: Users{
			List: []User{
				User{
					Games: Games{
						List: []Game{
							Game{
								Leagues: Leagues{
									List: leagues,
								},
							},
						},
					},
				},
			},
		},
	}
}

// mockClient creates a goff.Client that returns the given content and error
// whenever client.GetFantasyContent is called.
func mockClient(f *FantasyContent, e error) *Client {
	return &Client{
		RequestCount: 0,
		provider:     &mockedContentProvider{content: f, err: e},
	}
}

// mockedContentProvider creates a goff.contentProvider that returns the
// given content and error whenever provider.Get is called.
type mockedContentProvider struct {
	lastGetUrl string
	content    *FantasyContent
	err        error
}

func (m *mockedContentProvider) Get(url string) (*FantasyContent, error) {
	m.lastGetUrl = url
	return m.content, m.err
}

// mockHttpClient creates a httpClient that always returns the given response
// and error whenever httpClient.Get is called.
func mockHttpClient(resp *http.Response, e error) httpClient {
	return &mockedHttpClient{
		response: resp,
		err:      e,
	}
}

type mockedHttpClient struct {
	lastGetUrl string
	response   *http.Response
	err        error
}

func (m *mockedHttpClient) Get(url string) (resp *http.Response, err error) {
	m.lastGetUrl = url
	return m.response, m.err
}

type mockOAuthConsumer struct {
	Response *http.Response
	Error    error
	LastURL  string
}

func (m *mockOAuthConsumer) Get(url string, data map[string]string, a *oauth.AccessToken) (*http.Response, error) {
	m.LastURL = url
	return m.Response, m.Error
}

//
// Large XML
//

var teamXmlContent = `
<?xml version="1.0" encoding="UTF-8"?>
<fantasy_content xmlns:yahoo="http://www.yahooapis.com/v1/base.rng" xmlns="http://fantasysports.yahooapis.com/fantasy/v2/base.rng" xml:lang="en-US" yahoo:uri="http://fantasysports.yahooapis.com/fantasy/v2/team/223.l.431.t.1" time="426.26690864563ms" copyright="Data provided by Yahoo! and STATS, LLC">
  <team>
    <team_key>223.l.431.t.1</team_key>
    <team_id>1</team_id>
    <name>Team Name</name>
    <url>http://football.fantasysports.yahoo.com/archive/pnfl/2009/431/1</url>
    <team_logos>
      <team_logo>
        <size>medium</size>
        <url>http://l.yimg.com/a/i/us/sp/fn/default/full/nfl/icon_01_48.gif</url>
      </team_logo>
    </team_logos>
    <division_id>2</division_id>
    <faab_balance>22</faab_balance>
    <managers>
      <manager>
        <manager_id>13</manager_id>
        <nickname>Nickname</nickname>
        <guid>1234567890</guid>
      </manager>
    </managers>
  </team>
</fantasy_content> `

var leagueXmlContent = `
    <?xml version="1.0" encoding="UTF-8"?>
    <fantasy_content xml:lang="en-US" yahoo:uri="http://fantasysports.yahooapis.com/fantasy/v2/league/223.l.431" xmlns:yahoo="http://www.yahooapis.com/v1/base.rng" time="181.80584907532ms" copyright="Data provided by Yahoo! and STATS, LLC" xmlns="http://fantasysports.yahooapis.com/fantasy/v2/base.rng">
      <league>
        <league_key>223.l.431</league_key>
        <league_id>431</league_id>
        <name>League Name</name>
        <url>http://football.fantasysports.yahoo.com/archive/pnfl/2009/431</url>
        <draft_status>postdraft</draft_status>
        <num_teams>14</num_teams>
        <edit_key>17</edit_key>
        <weekly_deadline/>
        <league_update_timestamp>1262595518</league_update_timestamp>
        <scoring_type>head</scoring_type>
        <current_week>16</current_week>
        <start_week>1</start_week>
        <end_week>16</end_week>
        <is_finished>1</is_finished>
      </league>
    </fantasy_content>`
