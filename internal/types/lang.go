package types

type Lang = string

const (
	LangEnglish    Lang = "en"
	LangSpanish    Lang = "es"
	LangChinese    Lang = "zh"
	LangKorean     Lang = "ko"
	LangJapanese   Lang = "ja"
	LangGerman     Lang = "de"
	LangRussian    Lang = "ru"
	LangFrench     Lang = "fr"
	LangDutch      Lang = "nl"
	LangItalian    Lang = "it"
	LangIndonesian Lang = "id"
	LangPortuguese Lang = "pt"
	LangSwedish    Lang = "sv"
	LangCzech      Lang = "cs"
)

var SupportedLanguages = []Lang{
	LangEnglish,
	LangSpanish,
	LangChinese,
	LangKorean,
	LangJapanese,
	LangGerman,
	LangRussian,
	LangFrench,
	LangDutch,
	LangItalian,
	LangIndonesian,
	LangPortuguese,
	LangSwedish,
	LangCzech,
}
