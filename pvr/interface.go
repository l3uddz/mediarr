package pvr

type Interface interface {
	Init(MediaType) error

	GetQualityProfileId(string) (int, error)
}
