package bot

import "fmt"

// Strings holds all localised UI text for one language.
type Strings struct {
	Greeting         string
	PinPrompt        string
	InvalidPin       string
	WardPrompt       string
	NagarsevakPrompt string
	Welcome          string

	// Main-menu button labels
	LabelSOS      string
	LabelRegister string
	LabelTrack    string

	// Generic labels
	LabelEnglish string
	LabelMarathi string
	LabelHindi   string
}

// GreetingFor returns a personalised greeting with the user's real phone number.
// phone is the E.164 number from the webhook (e.g. "919812345678").
func GreetingFor(lang, phone string) string {
	// Format: +91 XXXXX XXXXX  (WhatsApp sends digits only, no "+")
	formatted := formatPhone(phone)

	switch lang {
	case "mr":
		return fmt.Sprintf(`👋 नमस्कार *%s*
अटलजनसेवा नागरिक सेवेशी जोडल्याबद्दल धन्यवाद.

आपल्या सोयीसाठी, कृपया आपली पसंतीची भाषा निवडा:`, formatted)
	case "hi":
		return fmt.Sprintf(`👋 नमस्ते *%s*
अटलजनसेवा नागरिक सेवा से जुड़ने के लिए धन्यवाद.

बेहतर अनुभव के लिए, कृपया अपनी पसंदीदा भाषा चुनें:`, formatted)
	default: // "en"
		return fmt.Sprintf(`👋 Hi *%s*
Thank you for connecting with *Ataljanseva Citizen Service*.

For a better experience, please select your preferred language:`, formatted)
	}
}

// formatPhone converts "919812345678" → "+91 98123 45678"
func formatPhone(raw string) string {
	// Strip leading "+" if present
	digits := raw
	if len(digits) > 0 && digits[0] == '+' {
		digits = digits[1:]
	}
	// Indian numbers arrive as 91XXXXXXXXXX (12 digits)
	if len(digits) == 12 && digits[:2] == "91" {
		local := digits[2:] // 10 digits
		return fmt.Sprintf("+91 %s %s", local[:5], local[5:])
	}
	// Fallback: just prefix "+"
	return "+" + digits
}

// I18n maps a language code to its Strings.
// NOTE: Greeting is intentionally left empty here — use GreetingFor() instead
// so the user's real number is injected at runtime.
var I18n = map[string]Strings{
	"en": {
		PinPrompt: `✅ Great! You've selected *English*.

──────────────────────
🔍 *Find Your Nagarsevak*
──────────────────────
Please enter your *6-digit PIN code* to locate the civic representatives in your area:

_Example: 411001, 400001, 440001, 421301_`,

		InvalidPin: `⚠️ *PIN Code Not Found*

The PIN code you entered doesn't match any records in our database.

Please try one of these valid demo PINs:
  📍 *411001* — Pune
  📍 *400001* — Mumbai City
  📍 *440001* — Nagpur
  📍 *421301* — Thane

Please type your *6-digit PIN code* again:`,

		WardPrompt: `📍 *Location Identified!*

┌───────────────────┐
  🏛  *State:*     %s
  🏙  *District:*  %s
└───────────────────┘

Your area has been found. Please select your *Ward* from the list below to continue 👇`,

		NagarsevakPrompt: `🏅 *Select Your Nagarsevak*

Here are the elected representatives for your ward.
Tap a name to connect with your Nagarsevak 👇`,

		Welcome: `🎉 *Connection Successful!*

Your Nagarsevak has been linked to your account.

━━━━━━━━━━━━━━━━━━━━━
*Ataljanseva Citizen Service*
Your bridge to local governance
━━━━━━━━━━━━━━━━━━━━━

Here's what you can do:
  🆘 Lodge emergency SOS complaints
  📝 File general civic complaints
  🔍 Track your complaint status
  📊 View ward reports & events

🌐 _ataljanseva.in_

How can we help you today? 👇`,

		LabelSOS:      "🆘 SOS Emergency",
		LabelRegister: "📝 File Complaint",
		LabelTrack:    "🔍 Track Complaint",
		LabelEnglish:  "🇮🇳 English",
		LabelMarathi:  "🇮🇳 मराठी",
		LabelHindi:    "🇮🇳 हिंदी",
	},

	"mr": {
		PinPrompt: `✅ उत्तम! आपण *मराठी* निवडली.

──────────────────────
🔍 *नगरसेवक शोधा*
──────────────────────
आपल्या परिसरातील नगरसेवक शोधण्यासाठी कृपया आपला *६-अंकी पिन कोड* टाइप करा:

_उदाहरण: ४११००१, ४००००१, ४४०००१, ४२१३०१_`,

		InvalidPin: `⚠️ *पिन कोड सापडला नाही*

आपण दिलेला पिन कोड आमच्या नोंदींमध्ये उपलब्ध नाही.

कृपया हे वैध डेमो पिन वापरा:
  📍 *411001* — पुणे
  📍 *400001* — मुंबई
  📍 *440001* — नागपूर
  📍 *421301* — ठाणे

कृपया पुन्हा *६-अंकी पिन कोड* टाइप करा:`,

		WardPrompt: `📍 *आपले स्थान सापडले!*

┌─────────────────────┐
  🏛  *राज्य:*    %s
  🏙  *जिल्हा:*   %s
└─────────────────────┘

आपला परिसर सापडला. पुढे जाण्यासाठी खालील यादीतून आपला *प्रभाग* निवडा 👇`,

		NagarsevakPrompt: `🏅 *आपला नगरसेवक निवडा*

आपल्या प्रभागातील निवडून आलेल्या प्रतिनिधींची यादी खाली आहे.
आपल्या नगरसेवकाशी जोडण्यासाठी नाव निवडा 👇`,

		Welcome: `🎉 *यशस्वीरीत्या जोडले गेले!*

आपला नगरसेवक आपल्या खात्याशी जोडला गेला आहे.

━━━━━━━━━━━━━━━━━━━━━━
*अटलजनसेवा नागरिक सेवा*
स्थानिक प्रशासनाशी आपला सेतू
━━━━━━━━━━━━━━━━━━━━━━

आपण येथे करू शकता:
  🆘 आपत्कालीन SOS तक्रार नोंदवा
  📝 साधारण नागरी तक्रार नोंदवा
  🔍 तक्रारीची स्थिती जाणून घ्या
  📊 प्रभाग अहवाल व कार्यक्रम पहा

🌐 _ataljanseva.in_

आज आपण काय करू इच्छिता? 👇`,

		LabelSOS:      "🆘 SOS आपत्कालीन",
		LabelRegister: "📝 तक्रार नोंदवा",
		LabelTrack:    "🔍 तक्रार ट्रॅक करा",
		LabelEnglish:  "🇮🇳 English",
		LabelMarathi:  "🇮🇳 मराठी",
		LabelHindi:    "🇮🇳 हिंदी",
	},

	"hi": {
		PinPrompt: `✅ बढ़िया! आपने *हिंदी* चुनी.

──────────────────────
🔍 *अपना नगरसेवक खोजें*
──────────────────────
अपने क्षेत्र के नागरिक प्रतिनिधि खोजने के लिए कृपया अपना *६-अंकीय पिन कोड* दर्ज करें:

_उदाहरण: ४११००१, ४००००१, ४४०००१, ४२१३०१_`,

		InvalidPin: `⚠️ *पिन कोड नहीं मिला*

आपका दर्ज किया गया पिन कोड हमारे रिकॉर्ड में उपलब्ध नहीं है.

कृपया इन मान्य डेमो पिन का उपयोग करें:
  📍 *411001* — पुणे
  📍 *400001* — मुंबई
  📍 *440001* — नागपुर
  📍 *421301* — ठाणे

कृपया फिर से *6 अंकों का पिन कोड* दर्ज करें:`,

		WardPrompt: `📍 *आपका स्थान मिल गया!*

┌─────────────────────┐
  🏛  *राज्य:*    %s
  🏙  *जिल्ला:*   %s
└─────────────────────┘

आपका क्षेत्र मिल गया. आगे बढ़ने के लिए नीचे सूची से अपना *वार्ड* चुनें 👇`,

		NagarsevakPrompt: `🏅 *अपना नगरसेवक चुनें*

आपके वार्ड के निर्वाचित प्रतिनिधियों की सूची नीचे दी गई है.
अपने नगरसेवक से जुड़ने के लिए नाम चुनें 👇`,

		Welcome: `🎉 *सफलतापूर्वक जुड़ गए!*

आपका नगरसेवक आपके खाते से जोड़ दिया गया है.

━━━━━━━━━━━━━━━━━━━━━━
*अटलजनसेवा नागरिक सेवा*
स्थानीय प्रशासन से आपका सेतु
━━━━━━━━━━━━━━━━━━━━━━

आप यहाँ कर सकते हैं:
  🆘 आपातकालीन SOS शिकायत दर्ज करें
  📝 सामान्य नागरिक शिकायत दर्ज करें
  🔍 अपनी शिकायत की स्थिति जानें
  📊 वार्ड रिपोर्ट और कार्यक्रम देखें

🌐 _ataljanseva.in_

आज आप क्या करना चाहते हैं? 👇`,

		LabelSOS:      "🆘 SOS आपातकालीन",
		LabelRegister: "📝 शिकायत दर्ज करें",
		LabelTrack:    "🔍 शिकायत ट्रैक करें",
		LabelEnglish:  "🇮🇳 English",
		LabelMarathi:  "🇮🇳 मराठी",
		LabelHindi:    "🇮🇳 हिंदी",
	},
}