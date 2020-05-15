[![made-with-golang](https://img.shields.io/badge/Made%20with-Golang-blue.svg?style=flat-square)](https://golang.org/)
[![License: GPL v3](https://img.shields.io/badge/License-GPL%203-blue.svg?style=flat-square)](https://github.com/l3uddz/mediarr/blob/master/LICENSE.md)
[![last commit (develop)](https://img.shields.io/github/last-commit/l3uddz/mediarr/develop.svg?colorB=177DC1&label=Last%20Commit&style=flat-square)](https://github.com/l3uddz/mediarr/commits/develop)
[![Discord](https://img.shields.io/discord/381077432285003776.svg?colorB=177DC1&label=Discord&style=flat-square)](https://discord.io/cloudbox)
[![Contributing](https://img.shields.io/badge/Contributing-gray.svg?style=flat-square)](CONTRIBUTING.md)
[![Donate](https://img.shields.io/badge/Donate-gray.svg?style=flat-square)](#donate)

# mediarr

CLI tool to add media to different PVR's in the *arr family from different providers.

## Example Configuration

```yaml
pvr:
  sonarr:
    type: sonarr
    url: https://sonarr.domain.com
    api_key: your-api-key
    quality_profile: WEBDL-1080p
    root_folder: /mnt/unionfs/Media/TV
    filters:
      ignores:
        # tvmaze
        - 'Provider == "tvmaze" && "English" not in Languages'
        - 'Provider == "tvmaze" && "Scripted" not in Genres'

        # trakt
        - 'Provider == "trakt" && "en" not in Languages'
        - 'Provider == "trakt" && Runtime < 15'
        - 'Provider == "trakt" && Network == ""'
        - 'Provider == "trakt" && not (any(Country, {# in ["us", "gb", "au", "ca"]}))'

        # generic
        - 'Year < (Now().Year() - 5)'
        - 'Year > Now().Year()'
        - 'Title contains "WWE"'
        - 'Title contains "Wrestling"'
        - '"animation" in Genres || "talk-show" in Genres'
        - 'len(Genres) == 0'
        - 'Network in ["Twitch", "Xbox Video", "YouTube"]'
        - 'Summary contains "transgend" || Summary contains "LGBT" || Summary contains "gay"'
        - 'Title matches "(?i)ru ?paul.+drag.+"'
  radarr:
    type: radarr
    url: https://radarr.domain.com
    api_key: your-api-key
    quality_profile: Remux
    root_folder: /mnt/unionfs/Media/Movies
    filters:
      ignores:
        # trakt
        - 'Provider == "trakt" && Runtime < 60'
        - 'Provider == "trakt" && ("music" in Genres || "documentary" in Genres)'
        - 'Provider == "trakt" && "en" not in Languages'

        # tmdb
        - 'Provider == "tmdb" && "en" not in Languages'

        # generic
        - 'Year < (Now().Year() - 5)'
        - 'Year > Now().Year()'
        - 'len(Genres) == 0'
        - 'Title startsWith "Untitled"'
        - 'Title contains "WWE" || Title contains "NXT"'
        - 'Title contains "Live:" || Title contains "Concert"'
        - 'Title contains "Musical"'
        - 'Title contains " Edition)"'
        - 'Summary contains "transgend" || Summary contains "LGBT" || Summary contains "gay"'
        - 'Title matches "^UFC.?\\d.+\\:"'
provider:
  tmdb:
    api_key: your-tmdb-api-key
  trakt:
    client_id: your-trakt-app-client-id
```

## Example Commands

1. Movies

`mediarr movies radarr trakt -t person --language en --query nicolas-cage --limit 5`

`mediarr movies radarr trakt -t anticipated --language en --country en,us,gb,ca,au --limit 5`

`mediarr movies radarr trakt -t popular --language en --country en,us,gb,ca,au --genre science-fiction --limit 10`

2. TV

`mediarr shows sonarr trakt -t popular --language en --country en,us,gb,ca,au --genre science-fiction --year 2019-2020 --limit 1`

`mediarr shows sonarr trakt -t anticipated --language en --country en,us,gb,ca,au`


## Additional Details

All commands support the `--dry-run` flag to mimic the entire run process with the exception of actually adding media to the PVR.

# Planned Features

1. Additions

- Provider(new): simkl

2. Enhancements

- Provider(trakt): support lists
- Provider(tmdb): support tv


***

# Donate

If you find this project helpful, feel free to make a small donation to the developer:

  - [Monzo](https://monzo.me/today): Credit Cards, Apple Pay, Google Pay

  - [Paypal: l3uddz@gmail.com](https://www.paypal.me/l3uddz)

  - [GitHub Sponsor](https://github.com/sponsors/l3uddz): GitHub matches contributions for first 12 months.

  - BTC: 3CiHME1HZQsNNcDL6BArG7PbZLa8zUUgjL