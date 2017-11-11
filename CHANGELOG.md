# Changelog #

## 0.4.0 (TBD) ##

- Added OAuth 2.0 as the preferred authentication option.
- Added support for 2015-2017 leagues.
- Added `URL`, `Settings`, and `Scoreboard` to `League`.
- Added `GetMatchupsForWeekRange` function to `Client`.
- Updated package to stop using deprecated methods in github.com/mrjones/oauth
    - Removed NewOAuthClient and NewCachedOAuthClient functions
    - Remove OAuthConsumer interface
    - Added NewClient and NewCachedClient
    - Added HTTPClient interface

## 0.3.0 (2015-01-09) ##

- Added `DraftStatus` to `League`.
- Added `Standings` to `League` and `GetLeagueStandings` function to `Client`.

## 0.2.0 (2014-08-17) ##

- Added debug package to test OAuth `GET` requests.
- Fixed support for 2013 leagues.
- Added support for 2014 leagues.
- Added optional caching of fantasy content.

## 0.1.0 (2014-03-02) ##

Initial public release

- OAuth 1.0 for authentication.
- Partial support for the available data types.
