package types

type TvShow struct {
	Name string
	LocalizedName string
	Id string
}

func NewTvShow(id string, name string, localizedName string) *TvShow{
	return &TvShow{
		Id: id,
		Name: name,
		LocalizedName: localizedName,
	}
}

type Episode struct {
	TvShow TvShow
	Season int
	Episode int
	Links []DownloadLink
}

type DownloadLink struct {
	Quality string
	Translation string
    EpisodeId string
	EpisodeHash string
	ShowId string
}

const (
	QualityStandard = "Standard"
	QualityHD = "HD"
	QualityFullHD = "FullHD"
	QualityUHD = "UHD"
	QualityUnknown = "Unknown"
)

const (
	AudioOriginal = "Original"
	AudioOriginalSubtitles = "Original Subtitles"
	AudioLocalizedSubtitles = "Subtitles"
	AudioLocalized = "Localized"
	AudioUnknown = "Unknown"
)