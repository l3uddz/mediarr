package config

type Pvr struct {
	Type           string
	URL            string
	ApiKey         string `mapstructure:"api_key"`
	QualityProfile string `mapstructure:"quality_profile"`
	RootFolder     string `mapstructure:"root_folder"`
	Filters        PvrFilters
}

type PvrFilters struct {
	Ignores []string
}
