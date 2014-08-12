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
// Test oauthHTTPClient
//

func TestOAuthHTTPClient(t *testing.T) {
	expected := &http.Response{}
	client := &oauthHTTPClient{
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

func TestOAuthHTTPClientError(t *testing.T) {
	client := &oauthHTTPClient{
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

func TestXMLContentProviderGetLeague(t *testing.T) {
	response := mockResponse(leagueXMLContent)
	client := &oauthHTTPClient{
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
	assertLeaguesEqual(t, []League{expectedLeague}, []League{league})
}

func TestXMLContentProviderGetTeam(t *testing.T) {
	response := mockResponse(teamXMLContent)
	client := &oauthHTTPClient{
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
	assertTeamsEqual(t, &expectedTeam, &team)
}

func TestXMLContentProviderGetError(t *testing.T) {
	response := mockResponse("content")
	client := &oauthHTTPClient{
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

func TestXMLContentProviderReadError(t *testing.T) {
	response := mockResponseReadErr()
	client := &oauthHTTPClient{
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

func TestXMLContentProviderParseError(t *testing.T) {
	response := mockResponse("<not-valid-xml/>")
	client := &oauthHTTPClient{
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
	expectedErr := errors.New("error retreiving content")
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
	leagues := []League{expectedLeague}
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
	content := &FantasyContent{Users: []User{}}
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
		Users: []User{
			User{
				Games: []Game{},
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
		Users: []User{
			User{
				Games: []Game{
					Game{
						Leagues: []League{},
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
	assertURLContainsParam(t, provider.lastGetURL, yearParam, "314")

	year := "2010"
	client.GetUserLeagues(year)
	assertURLContainsParam(t, provider.lastGetURL, yearParam, yearKeys[year])

	_, err := client.GetUserLeagues("1900")
	if err == nil {
		t.Fatalf("no error returned for year not supported by yahoo")
	}
}

//
// Test GetTeam
//

func TestGetTeam(t *testing.T) {
	client := mockClient(&FantasyContent{Team: expectedTeam}, nil)

	actual, err := client.GetTeam(expectedTeam.TeamKey)
	if err != nil {
		t.Fatalf("Client returned unexpected error: %s", err)
	}
	assertTeamsEqual(t, &expectedTeam, actual)
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
	client := mockClient(&FantasyContent{League: expectedLeague}, nil)

	actual, err := client.GetLeagueMetadata(expectedLeague.LeagueKey)
	if err != nil {
		t.Fatalf("Client returned unexpected error: %s", err)
	}

	assertLeaguesEqual(t, []League{expectedLeague}, []League{*actual})
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
			Players: players,
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
			Players: players,
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
				Players: players,
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

	assertURLContainsParam(t, provider.lastGetURL, "player_keys", players[0].PlayerKey)
	assertURLContainsParam(t, provider.lastGetURL, "week", fmt.Sprintf("%d", week))
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
				Players: players,
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
				Players: players,
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
	content := &FantasyContent{
		League: League{
			Teams: []Team{
				expectedTeam,
			},
		},
	}
	client := mockClient(content, nil)
	actual, err := client.GetAllTeamStats("123", 12)
	if err != nil {
		t.Fatalf("Client returned unexpected error: %s", err)
	}

	assertTeamsEqual(t, &expectedTeam, &actual[0])
}

func TestGetAllTeamStatsError(t *testing.T) {
	team := Team{TeamKey: "key1", TeamID: 1, Name: "name1"}
	content := &FantasyContent{
		League: League{
			Teams: []Team{
				team,
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
			Teams: []Team{
				team,
			},
		},
	}
	week := 12
	provider := &mockedContentProvider{content: content, err: nil}
	client := &Client{provider: provider}
	client.GetAllTeamStats("123", week)
	assertURLContainsParam(
		t,
		provider.lastGetURL,
		"week",
		fmt.Sprintf("%d", week))
}

//
// Test GetAllTeams
//

func TestGetAllTeams(t *testing.T) {
	content := &FantasyContent{
		League: League{
			Teams: []Team{
				expectedTeam,
			},
		},
	}
	client := mockClient(content, nil)
	actual, err := client.GetAllTeams("123")

	if err != nil {
		t.Fatalf("Client returned unexpected error: %s", err)
	}

	assertTeamsEqual(t, &expectedTeam, &actual[0])
}

func TestGetAllTeamsError(t *testing.T) {
	team := Team{TeamKey: "key1", TeamID: 1, Name: "name1"}
	content := &FantasyContent{
		League: League{
			Teams: []Team{
				team,
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

func assertURLContainsParam(t *testing.T, url string, param string, value string) {
	if !strings.Contains(url, param+"="+value) {
		t.Fatalf("Could not locate paramater in request URL\n"+
			"\tparamter: %s\n\tvalue: %s\n\turl: %s",
			param,
			value,
			url)
	}
}

func assertTeamsEqual(t *testing.T, expectedTeam *Team, actualTeam *Team) {
	assertStringEquals(t, expectedTeam.TeamKey, actualTeam.TeamKey)
	assertUintEquals(t, expectedTeam.TeamID, actualTeam.TeamID)
	assertFloatEquals(t, expectedTeam.TeamPoints.Total, actualTeam.TeamPoints.Total)
	assertFloatEquals(
		t,
		expectedTeam.TeamProjectedPoints.Total,
		actualTeam.TeamProjectedPoints.Total)
	assertStringEquals(t, expectedTeam.Name, actualTeam.Name)
	assertUintEquals(
		t,
		expectedTeam.Managers[0].ManagerID,
		actualTeam.Managers[0].ManagerID)
	assertStringEquals(
		t,
		expectedTeam.Managers[0].Nickname,
		actualTeam.Managers[0].Nickname)
	assertStringEquals(t, expectedTeam.Managers[0].Guid, actualTeam.Managers[0].Guid)
	assertStringEquals(t, expectedTeam.TeamLogos[0].Size, actualTeam.TeamLogos[0].Size)
	assertStringEquals(t, expectedTeam.TeamLogos[0].URL, actualTeam.TeamLogos[0].URL)
}

func assertLeaguesEqual(t *testing.T, expectedLeagues []League, actualLeagues []League) {
	for i := range expectedLeagues {
		assertStringEquals(t, expectedLeagues[i].LeagueKey, actualLeagues[i].LeagueKey)
		assertUintEquals(t, expectedLeagues[i].LeagueID, actualLeagues[i].LeagueID)
		assertStringEquals(t, expectedLeagues[i].Name, actualLeagues[i].Name)
		assertIntEquals(t, expectedLeagues[i].CurrentWeek, actualLeagues[i].CurrentWeek)
		assertIntEquals(t, expectedLeagues[i].StartWeek, actualLeagues[i].StartWeek)
		assertIntEquals(t, expectedLeagues[i].EndWeek, actualLeagues[i].EndWeek)
		assertBoolEquals(t, expectedLeagues[i].IsFinished, actualLeagues[i].IsFinished)
	}
}

func assertStringEquals(t *testing.T, expected string, actual string) {
	if actual != expected {
		t.Fatalf("Unexpected content\n"+
			"\tactual: %s\n"+
			"\texpected: %s",
			actual,
			expected)
	}
}

func assertFloatEquals(t *testing.T, expected float64, actual float64) {
	if actual != expected {
		t.Fatalf("Unexpected content\n"+
			"\tactual: %f\n"+
			"\texpected: %f",
			actual,
			expected)
	}
}

func assertUintEquals(t *testing.T, expected uint64, actual uint64) {
	if actual != expected {
		t.Fatalf("Unexpected content\n"+
			"\tactual: %d\n"+
			"\texpected: %d",
			actual,
			expected)
	}
}

func assertIntEquals(t *testing.T, expected int, actual int) {
	if actual != expected {
		t.Fatalf("Unexpected content\n"+
			"\tactual: %d\n"+
			"\texpected: %d",
			actual,
			expected)
	}
}

func assertBoolEquals(t *testing.T, expected bool, actual bool) {
	if actual != expected {
		t.Fatalf("Unexpected content\n"+
			"\tactual: %t\n"+
			"\texpected: %t",
			actual,
			expected)
	}
}

//
// Mocks
//

func createLeagueList(leagues ...League) *FantasyContent {
	return &FantasyContent{
		Users: []User{
			User{
				Games: []Game{
					Game{
						Leagues: leagues,
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
	lastGetURL string
	content    *FantasyContent
	err        error
}

func (m *mockedContentProvider) Get(url string) (*FantasyContent, error) {
	m.lastGetURL = url
	return m.content, m.err
}

// mockHTTPClient creates a httpClient that always returns the given response
// and error whenever httpClient.Get is called.
func mockHTTPClient(resp *http.Response, e error) httpClient {
	return &mockedHTTPClient{
		response: resp,
		err:      e,
	}
}

type mockedHTTPClient struct {
	lastGetURL string
	response   *http.Response
	err        error
}

func (m *mockedHTTPClient) Get(url string) (resp *http.Response, err error) {
	m.lastGetURL = url
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
// Test Data
//

var expectedTeam = Team{
	TeamKey: "223.l.431.t.1",
	TeamID:  1,
	Name:    "Team Name",
	Managers: []Manager{
		Manager{
			ManagerID: 13,
			Nickname:  "Nickname",
			Guid:      "1234567890",
		},
	},
	TeamPoints: Points{
		CoverageType: "week",
		Week:         16,
		Total:        123.45,
	},
	TeamProjectedPoints: Points{
		CoverageType: "week",
		Week:         16,
		Total:        543.21,
	},
	TeamLogos: []TeamLogo{
		TeamLogo{
			Size: "medium",
			URL:  "http://example.com/logo.png",
		},
	},
}
var teamXMLContent = `
<?xml version="1.0" encoding="UTF-8"?>
<fantasy_content xmlns:yahoo="http://www.yahooapis.com/v1/base.rng" xmlns="http://fantasysports.yahooapis.com/fantasy/v2/base.rng" xml:lang="en-US" yahoo:uri="http://fantasysports.yahooapis.com/fantasy/v2/team/223.l.431.t.1" time="426.26690864563ms" copyright="Data provided by Yahoo! and STATS, LLC">
  <team>
    <team_key>` + expectedTeam.TeamKey + `</team_key>
    <team_id>` + fmt.Sprintf("%d", expectedTeam.TeamID) + `</team_id>
    <name>` + expectedTeam.Name + `</name>
    <url>http://football.fantasysports.yahoo.com/archive/pnfl/2009/431/1</url>
    <team_logos>
      <team_logo>
        <size>` + expectedTeam.TeamLogos[0].Size + `</size>
        <url>` + expectedTeam.TeamLogos[0].URL + `</url>
      </team_logo>
    </team_logos>
    <division_id>2</division_id>
    <faab_balance>22</faab_balance>
    <managers>
      <manager>
        <manager_id>` + fmt.Sprintf("%d", expectedTeam.Managers[0].ManagerID) +
	`</manager_id>
        <nickname>` + expectedTeam.Managers[0].Nickname + `</nickname>
        <guid>` + expectedTeam.Managers[0].Guid + `</guid>
      </manager>
    </managers>
    <team_points>  
        <coverage_type>` + expectedTeam.TeamPoints.CoverageType + `</coverage_type>  
        <week>` + fmt.Sprintf("%d", expectedTeam.TeamPoints.Week) + `</week>  
        <total>` + fmt.Sprintf("%f", expectedTeam.TeamPoints.Total) + `</total>  
    </team_points>  
    <team_projected_points>  
        <coverage_type>` + expectedTeam.TeamProjectedPoints.CoverageType +
	`</coverage_type>  
        <week>` + fmt.Sprintf("%d", expectedTeam.TeamProjectedPoints.Week) + `</week>  
        <total>` + fmt.Sprintf("%f", expectedTeam.TeamProjectedPoints.Total) + `</total>
    </team_projected_points> 
  </team>
</fantasy_content> `

var expectedLeague = League{
	LeagueKey:   "223.l.431",
	LeagueID:    341,
	Name:        "League Name",
	CurrentWeek: 16,
	StartWeek:   1,
	EndWeek:     16,
	IsFinished:  true,
}
var leagueXMLContent = `
    <?xml version="1.0" encoding="UTF-8"?>
    <fantasy_content xml:lang="en-US" yahoo:uri="http://fantasysports.yahooapis.com/fantasy/v2/league/223.l.431" xmlns:yahoo="http://www.yahooapis.com/v1/base.rng" time="181.80584907532ms" copyright="Data provided by Yahoo! and STATS, LLC" xmlns="http://fantasysports.yahooapis.com/fantasy/v2/base.rng">
      <league>
        <league_key>` + expectedLeague.LeagueKey + `</league_key>
        <league_id>` + fmt.Sprintf("%d", expectedLeague.LeagueID) + `</league_id>
        <name>` + expectedLeague.Name + `</name>
        <url>http://football.fantasysports.yahoo.com/archive/pnfl/2009/431</url>
        <draft_status>postdraft</draft_status>
        <num_teams>14</num_teams>
        <edit_key>17</edit_key>
        <weekly_deadline/>
        <league_update_timestamp>1262595518</league_update_timestamp>
        <scoring_type>head</scoring_type>
        <current_week>` + fmt.Sprintf("%d", expectedLeague.CurrentWeek) +
	`</current_week>
        <start_week>` + fmt.Sprintf("%d", expectedLeague.StartWeek) +
	`</start_week>
        <end_week>` + fmt.Sprintf("%d", expectedLeague.EndWeek) + `</end_week>
        <is_finished>` + fmt.Sprintf("%t", expectedLeague.IsFinished) + `</is_finished>
      </league>
    </fantasy_content>`
