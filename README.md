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