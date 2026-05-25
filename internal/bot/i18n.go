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
कृपया आपली पसंतीची भाषा निवडा:`, formatted)
	case "hi":
		return fmt.Sprintf(`👋 नमस्ते *%s*
कृपया अपनी पसंदीदा भाषा चुनें:`, formatted)
	default: // "en"
		return fmt.Sprintf(`👋 Hi *%s*
Please select your preferred language from below 👇!!!! 

कृपया तुमची पसंतीची भाषा निवडा 👇!!!! 

कृपया नीचे से अपनी पसंदीदा भाषा चुनें 👇!!!!`, formatted)
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

_Example: 401107, 401303_`,

		InvalidPin: `⚠️ *PIN Code Not Found*

The PIN code you entered doesn't match any records in our database.

Please try one of these valid demo PINs:
  📍 *401107* — Mumbai City
  📍 *401303* — Nagpur


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

		Welcome: `Thank you for staying connected with the Ataljanseva Citizen
Service.

You can:
- View work reports, events & meetings, and active programs
through the Ataljanseva Citizen Portal.

🌐 For more information, please visit: _ataljanseva.in_

How can we help you today? 👇`,

		LabelSOS:      "",
		LabelRegister: "",
		LabelTrack:    "",
		LabelEnglish:  "English",
		LabelMarathi:  "मराठी",
		LabelHindi:    "हिंदी",
	},

	"mr": {
		PinPrompt: `✅ उत्तम! आपण *मराठी* निवडली.

──────────────────────
🔍 *नगरसेवक शोधा*
──────────────────────
आपल्या परिसरातील नगरसेवक शोधण्यासाठी कृपया आपला *६-अंकी पिन कोड* टाइप करा:

_उदाहरण: ४०११०७, ४०१३०३_`,

		InvalidPin: `⚠️ *पिन कोड सापडला नाही*

आपण दिलेला पिन कोड आमच्या नोंदींमध्ये उपलब्ध नाही.

कृपया हे वैध डेमो पिन वापरा:
  📍 *४०११०७* — ठाणे
  📍 *४०१३०३* — पालघर

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

		Welcome: `अटलजनसेवा नागरि क सेवेशी जोडलेलेराहि ल्याबद्दल धन्यवाद।

*आपण येथे:*
-  अटलजनसेवा नागरि क पोर्टलद्वारेकामाचा अहवाल, कार्यक्रर्य म व बठै कांची माहि ती
तसेच सक्रि य योजनांची माहि ती पाहू शकता

🌐 अधि क माहि तीसाठी कृ पया भेट द्या: _ataljanseva.in_

कृपया खालील पर्या यांपकै ी एक नि वडा 👇`,

		LabelSOS:      "",
		LabelRegister: "",
		LabelTrack:    "",
		LabelEnglish:  "English",
		LabelMarathi:  "मराठी",
		LabelHindi:    "हिंदी",
	},

	"hi": {
		PinPrompt: `✅ बढ़िया! आपने *हिंदी* चुनी.

──────────────────────
🔍 *अपना नगरसेवक खोजें*
──────────────────────
अपने क्षेत्र के नागरिक प्रतिनिधि खोजने के लिए कृपया अपना *६-अंकीय पिन कोड* दर्ज करें:

_उदाहरण: ४०११०७, ४०१३०३_`,

		InvalidPin: `⚠️ *पिन कोड नहीं मिला*

आपका दर्ज किया गया पिन कोड हमारे रिकॉर्ड में उपलब्ध नहीं है.

कृपया इन मान्य डेमो पिन का उपयोग करें:
  📍 *४०११०७* — ठाणे
  📍 *४०१३०३* — पालघर

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

		Welcome: `अटलजनसेवा नागरि क सेवा सेजड़ु
ेरहनेके लि ए धन्यवाद।

*आप यहाँ:*
-अटलजनसेवा नागरि क पोर्टल के माध्यम सेकार्य रि पोर्ट, कार्यक्रर्य म एवं बठै कों तथा
सक्रि य योजनाओं की जानकारी देख सकते है

🌐 अधिक जानकारी के लिए कृपया विजिट करें: _ataljanseva.in_

आज आप क्या करना चाहते हैं? 👇`,

		LabelSOS:      "",
		LabelRegister: "",
		LabelTrack:    "",
		LabelEnglish:  "English",
		LabelMarathi:  "मराठी",
		LabelHindi:    "हिंदी",
	},
}