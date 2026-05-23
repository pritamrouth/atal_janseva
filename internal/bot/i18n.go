package bot

// Strings holds all localised UI text for one language.
type Strings struct {
	Greeting          string
	PinPrompt         string
	InvalidPin        string
	WardPrompt        string
	NagarsevakPrompt  string
	Welcome           string

	// Main-menu button labels
	LabelSOS      string
	LabelRegister string
	LabelTrack    string

	// Generic labels
	LabelEnglish string
	LabelMarathi string
	LabelHindi   string
}

// I18n maps a language code to its Strings.
var I18n = map[string]Strings{
	"en": {
		Greeting: `👋 Hi *+91 98XXX XXXXX*
Thank you for connecting with *Ataljanseva Citizen Service*.

Please select your preferred language:`,
		PinPrompt:  `Great! You selected *English*. ✅

To locate your Nagarsevak, please type your *6-digit PIN code* below:`,
		InvalidPin: `⚠️ PIN code not found in our database.

Please try one of these demo PINs:
• *411001* – Pune
• *400001* – Mumbai
• *440001* – Nagpur
• *421301* – Thane

Type your 6-digit PIN again:`,
		WardPrompt: `📍 Location identified!

*State:* %s
*District:* %s

Please select your *Ward* from the list below:`,
		NagarsevakPrompt: `Here are the Nagarsevaks for your ward.
Please select one to continue 👇`,
		Welcome: `✅ *Thank you!* Your Nagarsevak has been connected.

This is the official *Ataljanseva Citizen Service* chatbot. You can:
• 🆘 Register emergency SOS complaints
• 📝 Register general complaints
• 🔍 Track complaint status
• 📊 View work reports & events

🌐 ataljanseva.in

Please select an option below 👇`,
		LabelSOS:      "🆘 SOS Emergency Complaint",
		LabelRegister: "📝 Register Complaint",
		LabelTrack:    "🔍 Track Your Complaint",
		LabelEnglish:  "🇬🇧 English",
		LabelMarathi:  "🇮🇳 मराठी",
		LabelHindi:    "🇮🇳 हिंदी",
	},
	"mr": {
		Greeting: `👋 नमस्कार *+91 98XXX XXXXX*
अटलजनसेवा नागरिक सेवेशी जोडल्याबद्दल धन्यवाद.

कृपया आपली पसंतीची भाषा निवडा:`,
		PinPrompt:  `उत्तम! आपण *मराठी* निवडली. ✅

आपला नगरसेवक शोधण्यासाठी, कृपया आपला *६-अंकी पिन कोड* टाइप करा:`,
		InvalidPin: `⚠️ हा पिन कोड आमच्या डेटाबेसमध्ये सापडला नाही.

कृपया हे डेमो पिन वापरा:
• *411001* – पुणे
• *400001* – मुंबई
• *440001* – नागपूर
• *421301* – ठाणे

पुन्हा ६-अंकी पिन टाइप करा:`,
		WardPrompt: `📍 आपले स्थान सापडले!

*राज्य:* %s
*जिल्हा:* %s

कृपया खाली यादीतून आपला *प्रभाग* निवडा:`,
		NagarsevakPrompt: `आपल्या प्रभागातील नगरसेवकांची यादी खाली आहे.
एक निवडा 👇`,
		Welcome: `✅ *धन्यवाद!* आपला नगरसेवक जोडला गेला.

आपण येथे:
• 🆘 आपत्कालीन SOS तक्रार नोंदवू शकता
• 📝 साधारण तक्रार नोंदवू शकता
• 🔍 तक्रारीची स्थिती ट्रॅक करू शकता

🌐 ataljanseva.in

कृपया खालील पर्यायांपैकी एक निवडा 👇`,
		LabelSOS:      "🆘 SOS आपत्कालीन तक्रार",
		LabelRegister: "📝 तक्रार नोंदवा",
		LabelTrack:    "🔍 आपली तक्रार ट्रॅक करा",
		LabelEnglish:  "🇬🇧 English",
		LabelMarathi:  "🇮🇳 मराठी",
		LabelHindi:    "🇮🇳 हिंदी",
	},
	"hi": {
		Greeting: `👋 नमस्ते *+91 98XXX XXXXX*
अटलजनसेवा नागरिक सेवा से जुड़ने के लिए धन्यवाद.

कृपया नीचे से अपनी पसंदीदा भाषा चुनें:`,
		PinPrompt:  `बढ़िया! आपने *हिंदी* चुनी. ✅

आपके नगरसेवक को खोजने के लिए, कृपया अपना *६-अंकीय पिन कोड* दर्ज करें:`,
		InvalidPin: `⚠️ यह पिन कोड हमारे डेटाबेस में नहीं मिला.

कृपया इन डेमो पिन का उपयोग करें:
• *411001* – पुणे
• *400001* – मुंबई
• *440001* – नागपुर
• *421301* – ठाणे

कृपया फिर से 6 अंकों का पिन दर्ज करें:`,
		WardPrompt: `📍 आपका स्थान मिल गया!

*राज्य:* %s
*जिल्ला:* %s

कृपया नीचे सूची से अपना *वार्ड* चुनें:`,
		NagarsevakPrompt: `आपके वार्ड के नगरसेवकों की सूची यहाँ है.
आगे बढ़ने के लिए एक चुनें 👇`,
		Welcome: `✅ *धन्यवाद!* आपके नगरसेवक से संपर्क हो गया.

आप यहाँ:
• 🆘 आपातकालीन SOS शिकायत दर्ज कर सकते हैं
• 📝 सामान्य शिकायत दर्ज कर सकते हैं
• 🔍 शिकायत की स्थिति ट्रैक कर सकते हैं

🌐 ataljanseva.in

कृपया नीचे एक विकल्प चुनें 👇`,
		LabelSOS:      "🆘 SOS आपातकालीन शिकायत",
		LabelRegister: "📝 शिकायत दर्ज करें",
		LabelTrack:    "🔍 अपनी शिकायत ट्रैक करें",
		LabelEnglish:  "🇬🇧 English",
		LabelMarathi:  "🇮🇳 मराठी",
		LabelHindi:    "🇮🇳 हिंदी",
	},
}
