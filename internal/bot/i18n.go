package bot

import "fmt"

// Strings holds all localised UI text for one language.
type Strings struct {
	Greeting         string
	PinPrompt        string
	InvalidPin       string
	WardPrompt       string
	Welcome          string
	SOSHeader        string
	ComplaintHeader  string
	TrackHeader      string

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
	
	return fmt.Sprintf(`👋 Hi *%s*
Please select your preferred language from below 👇!!

कृपया तुमची पसंतीची भाषा निवडा 👇!! 

कृपया नीचे से अपनी पसंदीदा भाषा चुनें 👇!!`, formatted)

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


		PinPrompt: `This is the *Ataljanseva Citizen Service* Automatic Reply Chatbot.

To continue further, please enter your *6-digit PIN Code* and *Ward Number* in the following format 👇!!.

📍 *PIN Code, Ward Number*
Example: *400601,21D*`,



		InvalidPin: `The entered PIN Code and Ward Number *%s* could not be found or do not match our records.

Please re-enter the correct *6-digit PIN Code* and *Ward Number* in the following format to continue 👇

📍 *PIN Code, Ward Number*
Example: *400601,21D*`,



		WardPrompt: `You entered *%s and %s* successfully.

Please select your Corporator from the list below 👇!!`,



		Welcome: `Thank you for staying connected with the Ataljanseva Citizen Service*!!

You can view Work reports, Events & Meetings, Atal Local Employment and Active programs through the Ataljanseva Citizen Portal.

🌐 For more information, please visit: _ataljanseva.in_

Please select one of the services below 👇!!`,


		SOSHeader: `Raise Emergency SOS requests`,

		ComplaintHeader: `Register complaints`,

		TrackHeader: `Track complaint status`,

		LabelSOS:      "SOS Emergency Complaint",
		LabelRegister: "Register Complaint",
		LabelTrack:    "Track Your Complaint",
		LabelEnglish:  "English",
		LabelMarathi:  "मराठी",
		LabelHindi:    "हिंदी",
	},

	"mr": {
		PinPrompt: `हे अटलजनसेवा नागरिक सेवेची ऑटोमॅटिक रिप्लाय चॅटबॉट आहे.

पुढे जाण्यासाठी कृपया आपला ६ अंकी पिन कोड आणि प्रभाग क्रमांक खालील स्वरूपात प्रविष्ट करा 👇!!

📍 *पिन कोड, प्रभाग क्रमांक*
उदाहरण: *४००६०१,२१डी*`,

		InvalidPin: `प्रविष्ट केलेला पिन कोड आणि प्रभाग क्रमांक *%s* आमच्या नोंदींशी जुळत नाही किंवा सापडला नाही.

पुढे जाण्यासाठी कृपया आपला योग्य *६ अंकी पिन कोड* आणि *प्रभाग क्रमांक* खालील स्वरूपात पुन्हा प्रविष्ट करा 👇!!

📍 *पिन कोड, प्रभाग क्रमांक*
उदाहरण: *४००६०१,२१डी*`,

		WardPrompt: `आपण *%s आणि %s* यशस्वीरित्या प्रविष्ट केले आहेत.

कृपया खालील यादीतून आपल्या नगरसेवकाची निवड करा 👇`,


		Welcome: `*अटलजनसेवा नागरिक सेवेशी* जोडलेले राहिल्याबद्दल धन्यवाद !!

आपण येथे अटलजनसेवा नागरिक पोर्टलद्वारे कामाचा अहवाल, कार्यक्रम व बैठकांची माहिती, अटल स्थानिक रोजगार सेवा तसेच सक्रिय योजनांची माहिती पाहू शकता.

🌐 अधिक माहितीसाठी कृपया भेट द्या: _ataljanseva.in_

कृपया खालील पर्यायांपैकी एक निवडा 👇!!`,


		SOSHeader: `आपत्कालीन SOS मदत विनतंी करा`,

		ComplaintHeader: `तक्रार नोंदव करा`,

		TrackHeader: `तक्रारीची स्थिती ट्रॅक करा`,

		LabelSOS:      "SOS आपत्कालीन तक्रार",
		LabelRegister: "तक्रार नोंदवा",
		LabelTrack:    "आपली तक्रार ट्रॅक करा",
		LabelEnglish:  "English",
		LabelMarathi:  "मराठी",
		LabelHindi:    "हिंदी",
	},

	"hi": {
		PinPrompt: `*अटलजनसेवा नागरिक सेवा* ऑटोमेटिक रिप्लाई चैटबॉट है।

आगे बढ़ने के लिए कृपया अपना ६ अंकों का पिन कोड और प्रभाग नंबर नीचे दिए गए प्रारूप में दर्ज करें 👇!!

📍 *पिन कोड, प्रभाग नंबर*
उदाहरण: *४००६०१,२१डी*`,


		InvalidPin: `आपके द्वारा दर्ज की गई पिन कोड और प्रभाग नंबर *%s* गलत है या उपलब्ध नहीं है।

आगे बढ़ने के लिए कृपया अपना सही ६ अंकों का पिन कोड और प्रभाग नंबर नीचे दिए गए प्रारूप में पुनः दर्ज करें 👇

📍 *पिन कोड, प्रभाग नंबर*
उदाहरण: *४००६०१,२१डी*`,



		WardPrompt: `आपने पिन कोड और प्रभाग नंबर *%s और %s* सफलतापूर्वक दर्ज की है।

कृपया नीचे दी गई सूची में से अपने नगरसेवक का चयन करें 👇!!`,



		Welcome: `अटलजनसेवा नागरिक सेवा से जुड़े रहने के लिए धन्यवाद !!

आप यहाँ अटलजनसेवा नागरिक पोर्टल के माध्यम से कार्य रिपोर्ट, अटल स्थानीय रोजगार सेवा, कार्यक्रम एवं बैठकों तथा सक्रिय योजनाओं की जानकारी देख सकते हैं।

🌐 अधिक जानकारी के लिए कृपया विजिट करें: _ataljanseva.in_

कृपया नीचे दिए गए विकल्पों में से एक का चयन करें 👇!!`,




		SOSHeader: `आपातकालीन SOS सहायता अनुरोध`,

		ComplaintHeader: `शिकायत दर्ज करें`,

		TrackHeader: `शिकायत की स्थिति ट्रैक करें`,

		LabelSOS:      "SOS आपातकालीन शिकायत",
		LabelRegister: "शिकायत दर्ज करें",
		LabelTrack:    "अपनी शिकायत ट्रैक करें",
		LabelEnglish:  "English",
		LabelMarathi:  "मराठी",
		LabelHindi:    "हिंदी",
	},
}