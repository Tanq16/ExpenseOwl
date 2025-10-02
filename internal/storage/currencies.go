package storage

import "strings"

func IsValidCurrency(code string) bool {
	_, ok := currencyCatalog[strings.ToUpper(code)]
	return ok
}

// keepOnly keeps in m only the keys that appear in the allow‐set.
//
//	m        : map you want to trim         (map[string]float64)
//	allowSet : membership test in O(1)      (map[string]string{})
func KeepOnlyValidCurrencies(m map[string]float64, allowSet map[string]string) {
	for k, v := range m {
		upperCase := strings.ToUpper(k)
		_, ok := allowSet[upperCase]
		delete(m, k)
		if ok {
			m[upperCase] = v
		}
	}
}

// CurrencyCatalog maps ISO-4217 codes to their symbol.
// The comment preserves the human-readable currency name.
var currencyCatalog = map[string]string{
	"AED": "د.إ", // United Arab Emirates Dirham
	"AFN": "؋", // Afghan Afghani
	"ALL": "L", // Albanian Lek
	"AMD": "֏", // Armenian Dram
	"ANG": "ƒ", // Netherlands Antillean Guilder
	"AOA": "Kz", // Angolan Kwanza
	"ARS": "$", // Argentine Peso
	"AUD": "$", // Australian Dollar
	"AWG": "ƒ", // Aruban Florin
	"AZN": "₼", // Azerbaijani Manat
	"BAM": "KM", // Bosnia-Herzegovina Convertible Mark
	"BBD": "$", // Barbadian Dollar
	"BDT": "৳", // Bangladeshi Taka
	"BGN": "лв", // Bulgarian Lev
	"BHD": ".د.ب", // Bahraini Dinar
	"BIF": "FBu", // Burundian Franc
	"BMD": "$", // Bermudian Dollar
	"BND": "$", // Brunei Dollar
	"BOB": "Bs.", // Bolivian Boliviano
	"BRL": "R$", // Brazilian Real
	"BSD": "$", // Bahamian Dollar
	"BTN": "Nu.", // Bhutanese Ngultrum
	"BWP": "P", // Botswanan Pula
	"BYN": "Br", // Belarusian Ruble
	"BZD": "$", // Belize Dollar
	"CAD": "$", // Canadian Dollar
	"CDF": "FC", // Congolese Franc
	"CHF": "Fr", // Swiss Franc
	"CLP": "$", // Chilean Peso
	"CNY": "¥", // Chinese Yuan
	"COP": "$", // Colombian Peso
	"CRC": "₡", // Costa Rican Colón
	"CUC": "$", // Cuban Convertible Peso
	"CUP": "$", // Cuban Peso
	"CVE": "Esc", // Cape Verdean Escudo
	"CZK": "Kč", // Czech Koruna
	"DJF": "Fdj", // Djiboutian Franc
	"DKK": "kr.", // Danish Krone
	"DOP": "$", // Dominican Peso
	"DZD": "دج", // Algerian Dinar
	"EGP": "£", // Egyptian Pound
	"ERN": "Nfk", // Eritrean Nakfa
	"ETB": "Br", // Ethiopian Birr
	"EUR": "€", // Euro
	"FJD": "$", // Fijian Dollar
	"FKP": "£", // Falkland Islands Pound
	"FOK": "kr", // Faroese Króna
	"GBP": "£", // British Pound Sterling
	"GEL": "₾", // Georgian Lari
	"GGP": "£", // Guernsey Pound
	"GHS": "₵", // Ghanaian Cedi
	"GIP": "£", // Gibraltar Pound
	"GMD": "D", // Gambian Dalasi
	"GNF": "FG", // Guinean Franc
	"GTQ": "Q", // Guatemalan Quetzal
	"GYD": "$", // Guyanese Dollar
	"HKD": "$", // Hong Kong Dollar
	"HNL": "L", // Honduran Lempira
	"HRK": "€", // Croatian Kuna (legacy)
	"HTG": "G", // Haitian Gourde
	"HUF": "Ft", // Hungarian Forint
	"IDR": "Rp", // Indonesian Rupiah
	"ILS": "₪", // Israeli New Shekel
	"IMP": "£", // Isle of Man Pound
	"INR": "₹", // Indian Rupee
	"IQD": "ع.د", // Iraqi Dinar
	"IRR": "﷼", // Iranian Rial
	"ISK": "kr", // Icelandic Króna
	"JEP": "£", // Jersey Pound
	"JMD": "$", // Jamaican Dollar
	"JOD": "د.ا", // Jordanian Dinar
	"JPY": "¥", // Japanese Yen
	"KES": "KSh", // Kenyan Shilling
	"KGS": "с", // Kyrgyzstani Som
	"KHR": "៛", // Cambodian Riel
	"KID": "$", // Kiribati Dollar
	"KMF": "CF", // Comorian Franc
	"KRW": "₩", // South Korean Won
	"KWD": "د.ك", // Kuwaiti Dinar
	"KYD": "$", // Cayman Islands Dollar
	"KZT": "₸", // Kazakhstani Tenge
	"LAK": "₭", // Laotian Kip
	"LBP": "ل.ل", // Lebanese Pound
	"LKR": "Rs", // Sri Lankan Rupee
	"LRD": "$", // Liberian Dollar
	"LSL": "L", // Lesotho Loti
	"LYD": "ل.د", // Libyan Dinar
	"MAD": "د.م.", // Moroccan Dirham
	"MDL": "L", // Moldovan Leu
	"MGA": "Ar", // Malagasy Ariary
	"MKD": "ден", // Macedonian Denar
	"MMK": "Ks", // Myanmar Kyat
	"MNT": "₮", // Mongolian Tögrög
	"MOP": "P", // Macanese Pataca
	"MRU": "UM", // Mauritanian Ouguiya
	"MUR": "₨", // Mauritian Rupee
	"MVR": "Rf", // Maldivian Rufiyaa
	"MWK": "MK", // Malawian Kwacha
	"MXN": "$", // Mexican Peso
	"MYR": "RM", // Malaysian Ringgit
	"MZN": "MT", // Mozambican Metical
	"NAD": "$", // Namibian Dollar
	"NGN": "₦", // Nigerian Naira
	"NIO": "C$", // Nicaraguan Córdoba
	"NOK": "kr", // Norwegian Krone
	"NPR": "₨", // Nepalese Rupee
	"NZD": "$", // New Zealand Dollar
	"OMR": "﷼", // Omani Rial
	"PAB": "B/.", // Panamanian Balboa
	"PEN": "S/", // Peruvian Sol
	"PGK": "K", // Papua New Guinean Kina
	"PHP": "₱", // Philippine Peso
	"PKR": "₨", // Pakistani Rupee
	"PLN": "zł", // Polish Złoty
	"PYG": "₲", // Paraguayan Guaraní
	"QAR": "﷼", // Qatari Riyal
	"RON": "lei", // Romanian Leu
	"RSD": "дин.", // Serbian Dinar
	"RUB": "₽", // Russian Ruble
	"RWF": "FRw", // Rwandan Franc
	"SAR": "﷼", // Saudi Riyal
	"SBD": "$", // Solomon Islands Dollar
	"SCR": "₨", // Seychellois Rupee
	"SDG": "£", // Sudanese Pound
	"SEK": "kr", // Swedish Krona
	"SGD": "$", // Singapore Dollar
	"SHP": "£", // Saint Helena Pound
	"SLE": "Le", // Sierra Leonean Leone (2023-)
	"SLL": "Le", // Sierra Leonean Leone (legacy)
	"SOS": "Sh", // Somali Shilling
	"SRD": "$", // Surinamese Dollar
	"SSP": "£", // South Sudanese Pound
	"STN": "Db", // São Tomé & Príncipe Dobra
	"SYP": "£", // Syrian Pound
	"SZL": "E", // Swazi Lilangeni
	"THB": "฿", // Thai Baht
	"TJS": "ЅМ", // Tajikistani Somoni
	"TMT": "m", // Turkmenistani Manat
	"TND": "د.ت", // Tunisian Dinar
	"TOP": "T$", // Tongan Paʻanga
	"TRY": "₺", // Turkish Lira
	"TTD": "$", // Trinidad & Tobago Dollar
	"TVD": "$", // Tuvaluan Dollar
	"TWD": "NT$", // New Taiwan Dollar
	"TZS": "Sh", // Tanzanian Shilling
	"UAH": "₴", // Ukrainian Hryvnia
	"UGX": "USh", // Ugandan Shilling
	"USD": "$", // United States Dollar
	"UYU": "$", // Uruguayan Peso
	"UZS": "soʻm", // Uzbekistani Soʻm
	"VES": "Bs.", // Venezuelan Bolívar
	"VND": "₫", // Vietnamese Đồng
	"VUV": "Vt", // Vanuatu Vatu
	"WST": "T", // Samoan Tala
	"XAF": "FCFA", // Central African CFA Franc
	"XCD": "$", // East Caribbean Dollar
	"XDR": "SDR", // Special Drawing Rights
	"XOF": "CFA", // West African CFA Franc
	"XPF": "₣", // CFP Franc
	"YER": "﷼", // Yemeni Rial
	"ZAR": "R", // South African Rand
	"ZMW": "ZK", // Zambian Kwacha
	"ZWL": "Z$", // Zimbabwean Dollar
}

