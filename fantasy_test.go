package goff

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	lru "vitess.io/vitess/go/cache"
)

//
// Test NewClient
//

func TestNewClient(t *testing.T) {

	client := NewClient(&mockHTTPClient{})

	if client == nil {
		t.Fatal("No client returned")
	}

	if client.RequestCount() != 0 {
		t.Fatalf("Invalid request count after initialization\n"+
			"\texpected: 0\n\tactual: %d",
			client.RequestCount())
	}
}

//
// Test NewCachedClient
//

func TestNewCachedClient(t *testing.T) {

	client := NewCachedClient(mockCache(), &mockHTTPClient{})

	if client == nil {
		t.Fatal("No client returned")
	}

	if client.RequestCount() != 0 {
		t.Fatalf("Invalid request count after initialization\n"+
			"\texpected: 0\n\tactual: %d",
			client.RequestCount())
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
// Test GetOAuth2Config
//

func TestGetOAuth2Config(t *testing.T) {
	clientID := "clientID"
	clientSecret := "clientSecret"
	redirectURL := "http://example.com"

	config := GetOAuth2Config(clientID, clientSecret, redirectURL)
	if config == nil {
		t.Fatal("No config returned")
	}

	if clientID != config.ClientID ||
		clientSecret != config.ClientSecret ||
		redirectURL != config.RedirectURL {
		t.Fatalf("Invalid config returned: %+v", config)
	}
}

//
// Test lruCache
//

func TestNewLRUCache(t *testing.T) {
	clientID := "clientID"
	duration := time.Hour
	lruCache := &lru.LRUCache{}

	cache := NewLRUCache(clientID, duration, lruCache)

	if cache == nil {
		t.Fatal("No cache returned")
	}

	if cache.ClientID != clientID {
		t.Fatalf("Unexpected client ID in cache\n\t"+
			"expected: %s\n\tactual: %s",
			clientID,
			cache.ClientID)
	}

	if cache.Duration != duration {
		t.Fatalf("Unexpected duration in cache\n\t"+
			"expected: %+v\n\tactual: %+v",
			duration,
			cache.Duration)
	}

	if cache.Cache != lruCache {
		t.Fatalf("Unexpected LRU cache in cache\n\t"+
			"expected: %+v\n\tactual: %+v",
			lruCache,
			cache.Cache)
	}
}

func TestGetKey(t *testing.T) {
	clientID := "clientID"
	duration := time.Hour
	lruCache := &lru.LRUCache{}
	cache := NewLRUCache(clientID, duration, lruCache)

	originalKey := "key"
	time := time.Unix(1408281677, 0)
	expectedKey := fmt.Sprintf("%s:%s:%s", clientID, originalKey, "391189")

	key := cache.getKey(originalKey, time)

	if key != expectedKey {
		t.Fatalf("Did not received expected key\n\texpected: %s"+
			"\n\tactual: %s",
			expectedKey,
			key)
	}
}

func TestGetNoContent(t *testing.T) {
	clientID := "clientID"
	duration := time.Hour
	lruCache := lru.NewLRUCache(10, func(_ any) int64 {
		return 1
	})
	cache := NewLRUCache(clientID, duration, lruCache)

	time := time.Unix(1408281677, 0)
	content, ok := cache.Get("http://example.com/fantasy", time)

	if ok {
		t.Fatalf("Cache returned content when it should not have been cached"+
			"content: %+v",
			content)
	}
}

func TestGetContentOfWrongType(t *testing.T) {
	clientID := "clientID"
	duration := time.Hour
	lruCache := lru.NewLRUCache(10, func(_ any) int64 {
		return 1
	})
	cache := NewLRUCache(clientID, duration, lruCache)

	time := time.Unix(1408281677, 0)
	url := "http://example.com/fantasy"

	cacheKey := cache.getKey(url, time)
	lruCache.Set(cacheKey, mockedValue{})

	content, ok := cache.Get(url, time)
	if ok {
		t.Fatalf("Cache returned content when it the wrong type had been cached"+
			"content: %+v",
			content)
	}
}

func TestGetWithContent(t *testing.T) {
	clientID := "clientID"
	duration := time.Hour
	lruCache := lru.NewLRUCache(10, func(_ any) int64 {
		return 1
	})
	cache := NewLRUCache(clientID, duration, lruCache)

	time := time.Unix(1408281677, 0)
	url := "http://example.com/fantasy"

	cacheKey := cache.getKey(url, time)
	expectedContent := createLeagueList(League{LeagueKey: "123"})
	lruCache.Set(cacheKey, &LRUCacheValue{content: expectedContent})

	content, ok := cache.Get(url, time)
	if !ok {
		t.Fatal("Cache did not return content")
	}

	if content != expectedContent {
		t.Fatalf("Cache did not return expected content\n\texpected: %+v"+
			"\n\tactual: %+v",
			expectedContent,
			content)
	}
}

func TestSet(t *testing.T) {
	clientID := "clientID"
	duration := time.Hour
	lruCache := lru.NewLRUCache(10, func(_ any) int64 {
		return 1
	})
	cache := NewLRUCache(clientID, duration, lruCache)

	time := time.Unix(1408281677, 0)
	url := "http://example.com/fantasy"
	expectedContent := createLeagueList(League{LeagueKey: "123"})
	cache.Set(url, time, expectedContent)

	cacheKey := cache.getKey(url, time)
	value, ok := lruCache.Get(cacheKey)
	if !ok {
		t.Fatal("Content not set in LRU cache correctly")
	}

	lruCacheValue, ok := value.(*LRUCacheValue)
	if !ok {
		t.Fatalf("Incorrect type used in LRU cache: %T", value)
	}

	if lruCacheValue.content != expectedContent {
		t.Fatalf("Unepxected content in cache\n\texpected: %+v\n\t"+
			"actual: %+v",
			expectedContent,
			lruCacheValue.content)
	}
}

func TestLRUCacheValueSize(t *testing.T) {
	value := LRUCacheValue{}
	if value.Size() != 1 {
		t.Fatalf("Incorrect size returned for LRU cache value\n\t"+
			"expected: %d\n\tactual: %d",
			1,
			value.Size())
	}
}

type mockedValue struct{}

func (m mockedValue) Size() int {
	return 1
}

//
// Test countingHTTPApiClient
//

func TestCountingHTTPClient(t *testing.T) {
	expected := &http.Response{}
	client := &countingHTTPApiClient{
		client: &mockHTTPClient{
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

func TestCountingHTTPClientMultipleErrorsConsumerKeyUnknown(t *testing.T) {
	expected := &http.Response{}
	client := &countingHTTPApiClient{
		client: &mockHTTPClient{
			Response:   expected,
			Error:      errors.New("consumer_key_unknown"),
			ErrorCount: 5,
		},
	}

	_, err := client.Get("http://example.com")
	if err == nil {
		t.Fatalf("no error returned from client when consumer failed")
	}
}

func TestCountingHTTPClientInitialErrorConsumerKeyUnknown(t *testing.T) {
	expected := &http.Response{}
	client := &countingHTTPApiClient{
		client: &mockHTTPClient{
			Response:   expected,
			Error:      errors.New("consumer_key_unknown"),
			ErrorCount: 4,
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

func TestCountingHTTPClientError(t *testing.T) {
	client := &countingHTTPApiClient{
		client: &mockHTTPClient{
			Response:   &http.Response{},
			Error:      errors.New("error"),
			ErrorCount: 5,
		},
	}

	_, err := client.Get("http://example.com")
	if err == nil {
		t.Fatalf("no error returned from client when consumer failed")
	}
}

func TestCountingHTTPClientAccessDeniedError(t *testing.T) {
	client := &countingHTTPApiClient{
		client: &mockHTTPClient{
			Response:   nil,
			Error:      errors.New("You are not allowed to view this page"),
			ErrorCount: 1,
		},
	}

	content, actualErr := client.Get("http://example.com")
	if content != nil {
		t.Fatalf("OAauth HTTP client returned unexpected content: %+v", content)
	}

	if actualErr != ErrAccessDenied {
		t.Fatalf("Unexpected error returned:\n\tExpected: %s\n\tActual: %s",
			ErrAccessDenied,
			actualErr)
	}
}

//
// Test cachedContentProvider
//

func TestCachedGetNoContentInCache(t *testing.T) {
	cache := mockCache()
	expectedContent := createLeagueList(League{LeagueKey: "123"})
	delegate := &mockedContentProvider{content: expectedContent, err: nil}
	provider := &cachedContentProvider{
		delegate: delegate,
		cache:    cache,
	}

	url := "http://example.com/fantasy"
	actualContent, err := provider.Get(url)

	if actualContent != expectedContent {
		t.Fatalf("Actual content did not equal expected content\n"+
			"\texpected: %+v\n\tactual: %+v",
			expectedContent,
			actualContent)
	}

	if cache.lastSetURL != url {
		t.Fatalf("Cache was not updated for correct URL\n\texpected: %s\n\t"+
			"actual: %s",
			url,
			cache.lastSetURL)
	}

	if cache.lastSetContent != expectedContent {
		t.Fatalf("Cache was not updated with correct Content\n\texpected: %+v"+
			"\n\tactual: %+v",
			expectedContent,
			cache.lastSetContent)
	}

	if err != nil {
		t.Fatalf("Cached provider returned error: %s", err)
	}
}

func TestCachedGetWithContentInCache(t *testing.T) {
	cache := mockCache()
	expectedContent := createLeagueList(League{LeagueKey: "123"})
	unexpectedContent := createLeagueList(League{LeagueKey: "456"})
	delegate := &mockedContentProvider{content: unexpectedContent, err: nil}
	provider := &cachedContentProvider{
		delegate: delegate,
		cache:    cache,
	}

	url := "http://example.com/fantasy"
	cache.data[url] = expectedContent
	actualContent, err := provider.Get(url)

	if actualContent != expectedContent {
		t.Fatalf("Actual content did not equal expected content\n"+
			"\texpected: %+v\n\tactual: %+v",
			expectedContent,
			actualContent)
	}

	if cache.lastSetURL != "" ||
		!cache.lastSetTime.IsZero() ||
		cache.lastSetContent != nil {
		t.Fatalf("Cache was updated for cached data\n\turl: %s\n\t"+
			"time: %+v\n\tcontent: %+v",
			cache.lastSetURL,
			cache.lastSetTime,
			cache.lastSetContent)
	}

	if err != nil {
		t.Fatalf("Cached provider returned error: %s", err)
	}
}

func TestCachedGetNoContentInCacheErrorReturnedCacheNotSet(t *testing.T) {
	cache := mockCache()
	err := errors.New("error")
	delegate := &mockedContentProvider{content: nil, err: err}
	provider := &cachedContentProvider{
		delegate: delegate,
		cache:    cache,
	}

	url := "http://example.com/fantasy"
	_, actualErr := provider.Get(url)

	if actualErr != err {
		t.Fatalf("Cached provider did not return expected error: \n\t"+
			"expected: %s\n\tactual: %s",
			err,
			actualErr)
	}

	if cache.lastSetURL != "" ||
		!cache.lastSetTime.IsZero() ||
		cache.lastSetContent != nil {
		t.Fatalf("Cache was updated after error\n\turl: %s\n\t"+
			"time: %+v\n\tcontent: %+v",
			cache.lastSetURL,
			cache.lastSetTime,
			cache.lastSetContent)
	}
}

//
// Test xmlContentProvider
//

func TestXMLContentProviderGetLeague(t *testing.T) {
	response := mockResponse(leagueXMLContent)
	client := &countingHTTPApiClient{
		client: &mockHTTPClient{
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
	client := &countingHTTPApiClient{
		client: &mockHTTPClient{
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
	client := &countingHTTPApiClient{
		client: &mockHTTPClient{
			Response:   response,
			Error:      errors.New("error"),
			ErrorCount: 1,
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
	client := &countingHTTPApiClient{
		client: &mockHTTPClient{
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
	client := &countingHTTPApiClient{
		client: &mockHTTPClient{
			Response: response,
		},
	}

	provider := &xmlContentProvider{client: client}
	_, err := provider.Get("http://example.com")

	if err == nil {
		t.Fatalf("error not returned when parse fails")
	}
}

func TestXMLContentProviderEmptyTagsForNumberFields(t *testing.T) {
	response := mockResponse(`
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
        <guid>` + expectedTeam.Managers[0].GUID + `</guid>
      </manager>
    </managers>
    <team_points>  
        <coverage_type>` + expectedTeam.TeamPoints.CoverageType + `</coverage_type>  
        <week>` + fmt.Sprintf("%d", expectedTeam.TeamPoints.Week) + `</week>  
        <total/>
    </team_points>  
    <team_projected_points>  
        <coverage_type>` + expectedTeam.TeamProjectedPoints.CoverageType +
		`</coverage_type>  
        <week>` + fmt.Sprintf("%d", expectedTeam.TeamProjectedPoints.Week) + `</week>  
        <total/>
    </team_projected_points> 
	<team_standings>
	    <rank/>
	</team_standings>
  </team>
</fantasy_content> `)
	client := &countingHTTPApiClient{
		client: &mockHTTPClient{
			Response: response,
		},
	}

	provider := &xmlContentProvider{client: client}
	content, err := provider.Get("http://example.com")

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if content.Team.TeamPoints.Total != 0 {
		t.Fatalf("expected zero team points, got: %f",
			content.Team.TeamPoints.Total)
	}

	if content.Team.TeamProjectedPoints.Total != 0 {
		t.Fatalf("expected zero team projected points, got: %f",
			content.Team.TeamProjectedPoints.Total)
	}

	if content.Team.TeamStandings.Rank != 0 {
		t.Fatalf("expected zero for team rank, got: %d",
			content.Team.TeamStandings.Rank)
	}
}

func TestFixContent(t *testing.T) {
	expectedRank := 5
	expectedRankStr := "5"
	expectedPoints := 543.21
	expectedPointsStr := "543.21"
	team := Team{
		TeamKey: "223.l.431.t.1",
		TeamID:  1,
		Name:    "Team Name",
		Managers: []Manager{
			Manager{
				ManagerID: 13,
				Nickname:  "Nickname",
				GUID:      "1234567890",
			},
		},
		TeamPoints: Points{
			CoverageType: "week",
			Week:         16,
			TotalStr:     expectedPointsStr,
		},
		TeamProjectedPoints: Points{
			CoverageType: "week",
			Week:         16,
			TotalStr:     expectedPointsStr,
		},
		TeamStandings: TeamStandings{
			RankStr: expectedRankStr,
		},
		TeamLogos: []TeamLogo{
			TeamLogo{
				Size: "medium",
				URL:  "http://example.com/logo.png",
			},
		},
		Roster: Roster{
			Players: []Player{
				Player{PlayerPoints: Points{TotalStr: expectedPointsStr}},
			},
		},
		Players: []Player{
			Player{PlayerPoints: Points{TotalStr: expectedPointsStr}},
		},
		Matchups: []Matchup{
			Matchup{Teams: []Team{expectedTeam, expectedTeam}},
		},
	}

	content := &FantasyContent{
		Team: team,
		League: League{
			Teams:     []Team{team},
			Standings: []Team{team},
			Players: []Player{
				Player{PlayerPoints: Points{TotalStr: expectedPointsStr}},
			},
			Scoreboard: Scoreboard{
				Matchups: []Matchup{Matchup{Teams: []Team{team, team}}},
			},
		},
	}

	actual := fixContent(content)
	if actual != content {
		t.Fatalf("Returned pointer should be the same as input:\n\t"+
			"expected: %+v\n\actual: %+v", content, actual)
	}

	checkTeam(t, &actual.Team, expectedPoints, expectedRank)
	checkTeam(t, &actual.League.Teams[0], expectedPoints, expectedRank)
	checkTeam(t, &actual.League.Standings[0], expectedPoints, expectedRank)
	checkTeam(t, &actual.League.Standings[0], expectedPoints, expectedRank)
	if actual.League.Players[0].PlayerPoints.Total != expectedPoints {
		t.Fatalf("Fantasy content not fixed for %s\n\tactual: %f\n\t"+
			"expected: %f",
			"League.Players.PlayerPoints.Total",
			actual.League.Players[0].PlayerPoints.Total,
			expectedPoints)
	}
}

func checkTeam(t *testing.T, team *Team, expectedPoints float64, expectedRank int) {
	if team.TeamPoints.Total != expectedPoints {
		t.Fatalf("Fantasy content not fixed for %s\n\tactual: %f\n\t"+
			"expected: %f",
			"TeamPoints.Total",
			team.TeamPoints.Total,
			expectedPoints)
	}

	if team.TeamProjectedPoints.Total != expectedPoints {
		t.Fatalf("Fantasy content not fixed for %s\n\tactual: %f\n\t"+
			"expected: %f",
			"TeamProjectedPoints.Total",
			team.TeamProjectedPoints.Total,
			expectedPoints)
	}

	if team.Roster.Players[0].PlayerPoints.Total != expectedPoints {
		t.Fatalf("Fantasy content not fixed for %s\n\tactual: %f\n\t"+
			"expected: %f",
			"Roster.Players.PlayerPoints.Total",
			team.Roster.Players[0].PlayerPoints.Total,
			expectedPoints)
	}

	if team.Players[0].PlayerPoints.Total != expectedPoints {
		t.Fatalf("Fantasy content not fixed for %s\n\tactual: %f\n\t"+
			"expected: %f",
			"Players.PlayerPoints.Total",
			team.Players[0].PlayerPoints.Total,
			expectedPoints)
	}

	if team.TeamStandings.Rank != expectedRank {
		t.Fatalf("Fantasy content not fixed for %s\n\tactual: %d\n\t"+
			"expected: %d",
			"TeamStandings.Rank",
			team.TeamStandings.Rank,
			expectedRank)
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
		t.Fatalf("Actual content did not equal expected content\n"+
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

	if actualErr != expectedErr {
		t.Fatalf("Unexpected error returned:\n\tExpected: %s\n\tActual: %s",
			expectedErr,
			actualErr)
	}
}

func TestGetFantasyContentRequestcount(t *testing.T) {
	client := mockClient(&FantasyContent{}, nil)
	client.GetFantasyContent("http://example.com/RequestOne")
	if client.RequestCount() != 1 {
		t.Fatalf("Fantasy client returned incorrect request count.\n"+
			"\texpected: 1\n\tactual: %d",
			client.RequestCount())
	}
	client.GetFantasyContent("http://example.com/RequestTwo")
	if client.RequestCount() != 2 {
		t.Fatalf("Fantasy client returned incorrect request count.\n"+
			"\texpected: 2\n\tactual: %d",
			client.RequestCount())
	}
	client.GetFantasyContent("http://example.com/RequestOne")
	if client.RequestCount() != 3 {
		t.Fatalf("Fantasy client returned incorrect request count.\n"+
			"\texpected: 3\n\tactual: %d",
			client.RequestCount())
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
		t.Fatalf("Client did not return error when no users were found\n"+
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
		t.Fatalf("Client should not have returned leagues\n"+
			"\tcontent: %+v",
			actual)
	}
}

func TestGetUserLeaguesMapsYear(t *testing.T) {
	content := createLeagueList(League{LeagueKey: "123"})
	provider := &mockedContentProvider{content: content, err: nil}
	client := &Client{
		Provider: provider,
	}

	client.GetUserLeagues("2013")
	yearParam := "game_keys"
	assertURLContainsParam(t, provider.lastGetURL, yearParam, "314")

	year := "2010"
	client.GetUserLeagues(year)
	assertURLContainsParam(t, provider.lastGetURL, yearParam, YearKeys[year])

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
// Test GetLeagueStandings
//

func TestGetLeagueStandings(t *testing.T) {
	client := mockClient(&FantasyContent{League: expectedLeague}, nil)

	actual, err := client.GetLeagueStandings(expectedLeague.LeagueKey)
	if err != nil {
		t.Fatalf("Client returned unexpected error: %s", err)
	}

	assertLeaguesEqual(t, []League{expectedLeague}, []League{*actual})
}

func TestGetLeagueStandingsError(t *testing.T) {
	client := mockClient(&FantasyContent{League: expectedLeague}, errors.New("error"))

	_, err := client.GetLeagueStandings(expectedLeague.LeagueKey)
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
		Provider: provider,
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
	client := &Client{Provider: provider}
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
// Test GetMatchupsForWeekRange
//

func TestGetMatchupsForWeekRange(t *testing.T) {
	week1 := []Matchup{
		Matchup{
			Week: 1,
		},
		Matchup{
			Week: 1,
		},
	}
	week2 := []Matchup{
		Matchup{
			Week: 2,
		},
		Matchup{
			Week: 2,
		},
	}
	week3 := []Matchup{
		Matchup{
			Week: 3,
		},
		Matchup{
			Week: 3,
		},
	}
	content := &FantasyContent{
		League: League{
			Scoreboard: Scoreboard{
				Weeks: "1,2,3",
				Matchups: []Matchup{
					week1[0],
					week1[1],
					week2[0],
					week2[1],
					week3[0],
					week3[1],
				},
			},
		},
	}
	provider := &mockedContentProvider{content: content, err: nil, count: 0}
	client := &Client{Provider: provider}
	actual, err := client.GetMatchupsForWeekRange("leagueKey", 1, 3)

	if err != nil {
		t.Fatalf("Client returned unexpected error: %s", err)
	}

	if len(actual[1]) != 2 ||
		len(actual[2]) != 2 ||
		len(actual[3]) != 2 {

		t.Fatalf("Unexected content returned after getting matchups: %+v",
			actual)
	}

	if !strings.Contains(provider.lastGetURL, "week=1,2,3") {
		t.Fatal("Did not generate proper request")
	}
}

func TestGetMatchupsForWeekRangeOneWeek(t *testing.T) {
	provider := &mockedContentProvider{
		content: nil,
		err:     errors.New("not tested"),
		count:   0,
	}
	client := &Client{Provider: provider}
	_, _ = client.GetMatchupsForWeekRange("leagueKey", 2, 2)

	if !strings.Contains(provider.lastGetURL, "week=2") {
		t.Fatal("Did not generate proper request")
	}
}

func TestGetMatchupsForWeekRangeError(t *testing.T) {
	client := mockClient(nil, errors.New("error"))
	_, err := client.GetMatchupsForWeekRange("leagueKey", 1, 3)

	if err == nil {
		t.Fatalf("Client did not return error")
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
	assertStringEquals(t, expectedTeam.Managers[0].GUID, actualTeam.Managers[0].GUID)
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
		Provider: &mockedContentProvider{content: f, err: e, count: 0},
	}
}

// mockedContentProvider creates a goff.ContentProvider that returns the
// given content and error whenever Provider.Get is called.
type mockedContentProvider struct {
	lastGetURL string
	content    *FantasyContent
	err        error
	count      int
}

func (m *mockedContentProvider) Get(url string) (*FantasyContent, error) {
	m.lastGetURL = url
	m.count++
	return m.content, m.err
}

func (m *mockedContentProvider) RequestCount() int {
	return m.count
}

type mockedCache struct {
	data           map[string](*FantasyContent)
	lastSetURL     string
	lastSetTime    time.Time
	lastSetContent *FantasyContent

	lastGetURL  string
	lastGetTime time.Time
}

func mockCache() *mockedCache {
	return &mockedCache{
		data: make(map[string](*FantasyContent)),
	}
}

func (c *mockedCache) Set(url string, time time.Time, content *FantasyContent) {
	c.lastSetURL = url
	c.lastSetTime = time
	c.lastSetContent = content
}

func (c *mockedCache) Get(
	url string,
	time time.Time) (content *FantasyContent, ok bool) {

	c.lastGetURL = url
	c.lastGetTime = time

	content, ok = c.data[url]
	return content, ok
}

type mockHTTPClient struct {
	Response   *http.Response
	Error      error
	ErrorCount int
	LastURL    string

	RequestCount int
}

func (m *mockHTTPClient) Get(url string) (*http.Response, error) {
	m.LastURL = url
	m.RequestCount++
	err := m.Error
	if m.RequestCount > m.ErrorCount {
		err = nil
	}
	return m.Response, err
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
			GUID:      "1234567890",
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
        <guid>` + expectedTeam.Managers[0].GUID + `</guid>
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
