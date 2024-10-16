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
	LangSlovak     Lang = "sk"
	LangPolish     Lang = "pl"
	LangRomanian   Lang = "ro"
	LangHungarian  Lang = "hu"
	LangFinnish    Lang = "fi"
	LangTurkish    Lang = "tr"
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
	LangSlovak,
	LangPolish,
	LangRomanian,
	LangHungarian,
	LangFinnish,
	LangTurkish,
}

func FullLangName(lang Lang) string {
	switch lang {
	case LangEnglish:
		return "English"
	case LangSpanish:
		return "Spanish"
	case LangChinese:
		return "Chinese"
	case LangKorean:
		return "Korean"
	case LangJapanese:
		return "Japanese"
	case LangGerman:
		return "German"
	case LangRussian:
		return "Russian"
	case LangFrench:
		return "French"
	case LangDutch:
		return "Dutch"
	case LangItalian:
		return "Italian"
	case LangIndonesian:
		return "Indonesian"
	case LangPortuguese:
		return "Portuguese"
	case LangSwedish:
		return "Swedish"
	case LangCzech:
		return "Czech"
	case LangSlovak:
		return "Slovak"
	case LangPolish:
		return "Polish"
	case LangRomanian:
		return "Romanian"
	case LangHungarian:
		return "Hungarian"
	case LangFinnish:
		return "Finnish"
	case LangTurkish:
		return "Turkish"
	default:
		return "Unknown"
	}
}
