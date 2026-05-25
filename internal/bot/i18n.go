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
		PinPrompt: `This is the *Ataljanseva Citizen Service* Automatic Reply Chatbot for citizen support and assistance.
To continue further, please search for your respective Corporator.
Please enter your *6-digit PIN Code* and *Ward Number* in the following format:

📍 *PIN Code, Ward Number*
Example: *400601,21D*`,

		InvalidPin: `⚠️ The entered PIN Code and Ward Number *%s* could not be found or do not match our records.

Please re-enter the correct *6-digit PIN Code* and *Ward Number* to continue.

📍 *PIN Code, Ward Number*
Example: *400601,21D*`,

		WardPrompt: `✅ You entered PIN Code and Ward Number *%s and %s* successfully.

Please select your Corporator from the list below 👇`,


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

		LabelSOS:      "SOS Emergency Complaint",
		LabelRegister: "Register Complaint",
		LabelTrack:    "Track Your Complaint",
		LabelEnglish:  "English",
		LabelMarathi:  "मराठी",
		LabelHindi:    "हिंदी",
	},

	"mr": {
		PinPrompt: `ही नागरिक सहायता व सेवांसाठी *अटलजनसेवा नागरिक सेवेची* अधिकृत ऑटोमॅटिक रिप्लाय चॅटबॉट सेवा आहे.
आपल्या संबंधित नगरसेवकाचा शोध घेण्यासाठी, कृपया ६ अंकी पिन कोड आणि आपला प्रभाग क्रमांक प्रविष्ट करा.
📍 *पिन कोड, प्रभाग क्रमांक*
उदाहरण: *४००६०१,२१डी*`,

		InvalidPin: `⚠️ प्रविष्ट केलेला पिन कोड आणि प्रभाग क्रमांक *%s* आमच्या नोंदींशी जुळत नाही किंवा सापडला नाही.

कृपया पुढे जाण्यासाठी योग्य *६ अंकी पिन कोड* आणि *प्रभाग क्रमांक* पुन्हा प्रविष्ट करा.

📍 *पिन कोड, प्रभाग क्रमांक*
उदाहरण: *४००६०१,२१डी*`,

		WardPrompt: `✅ आपण पिन कोड आणि प्रभाग क्रमांक *%s आणि %s* यशस्वीरित्या प्रविष्ट केले आहेत.

कृपया खालील यादीतून आपल्या नगरसेवकाची निवड करा 👇`,


		NagarsevakPrompt: `🏅 *आपला नगरसेवक निवडा*

आपल्या प्रभागातील निवडून आलेल्या प्रतिनिधींची यादी खाली आहे.
आपल्या नगरसेवकाशी जोडण्यासाठी नाव निवडा 👇`,

		Welcome: `अटलजनसेवा नागरि क सेवेशी जोडलेलेराहि ल्याबद्दल धन्यवाद।

*आपण येथे:*
-  अटलजनसेवा नागरि क पोर्टलद्वारेकामाचा अहवाल, कार्यक्रर्य म व बठै कांची माहि ती
तसेच सक्रि य योजनांची माहि ती पाहू शकता

🌐 अधि क माहि तीसाठी कृ पया भेट द्या: _ataljanseva.in_

कृपया खालील पर्या यांपकै ी एक नि वडा 👇`,

		LabelSOS:      "SOS आपत्कालीि तक्रार",
		LabelRegister: "तक्रार िोंदव",
		LabelTrack:    "आपली तक्रार टरॅक करा",
		LabelEnglish:  "English",
		LabelMarathi:  "मराठी",
		LabelHindi:    "हिंदी",
	},

	"hi": {
		PinPrompt: `यह नागरिक सहायता और सेवाओं के लिए *अटलजनसेवा नागरिक सेवा* की अधिकृत ऑटोमेटिक रिप्लाय चैटबॉट सेवा है.
अपने संबंधित नगरसेवक को खोजने के लिए, कृपया ६ अंकों का पिन कोड और अपना प्रभाग क्रमांक दर्ज करें.
📍 *पिन कोड, प्रभाग क्रमांक*
उदाहरण: *४००६०१,२१डी*`,

		InvalidPin: `⚠️ आपके द्वारा दर्ज किया गया पिन कोड और प्रभाग क्रमांक *%s* गलत है या उपलब्ध नहीं है.

कृपया आगे बढ़ने के लिए सही *६ अंकों का पिन कोड* और *प्रभाग क्रमांक* पुनः दर्ज करें.

📍 *पिन कोड, प्रभाग क्रमांक*
उदाहरण: *४००६०१,२१डी*`,

		WardPrompt: `✅ आपने पिन कोड और प्रभाग क्रमांक *%s और %s* सफलतापूर्वक दर्ज किया है.

कृपया नीचे दी गई सूची में से अपने नगरसेवक का चयन करें 👇`,

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

		LabelSOS:      "SOS आपातकालीि नशकायत",
		LabelRegister: "नशकायत दजवकर",
		LabelTrack:    "अपिी नशकायत टरैक कर",
		LabelEnglish:  "English",
		LabelMarathi:  "मराठी",
		LabelHindi:    "हिंदी",
	},
}