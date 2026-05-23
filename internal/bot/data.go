package bot

// ─────────────────────────────────────────────
// Static reference data (mirrors the HTML JS)
// ─────────────────────────────────────────────

// PincodeData holds state / district / ward list for a PIN code.
type PincodeData struct {
	State    string
	District string
	Wards    []string
}

// Nagarsevak holds a single representative's details.
type Nagarsevak struct {
	Name     string
	Party    string
	Initials string
}

// Pincodes is the PIN → location lookup table.
var Pincodes = map[string]PincodeData{
	"411001": {
		State: "Maharashtra", District: "Pune",
		Wards: []string{
			"Ward 1 – Kasba Peth",
			"Ward 2 – Shivajinagar",
			"Ward 3 – Deccan Gymkhana",
			"Ward 4 – Kothrud",
		},
	},
	"400001": {
		State: "Maharashtra", District: "Mumbai City",
		Wards: []string{
			"Ward A – Colaba",
			"Ward B – Mazgaon",
			"Ward C – Girgaon",
			"Ward D – Worli",
		},
	},
	"440001": {
		State: "Maharashtra", District: "Nagpur",
		Wards: []string{
			"Ward 1 – Civil Lines",
			"Ward 2 – Dharampeth",
			"Ward 3 – Lakadganj",
			"Ward 4 – Mangalwari",
		},
	},
	"421301": {
		State: "Maharashtra", District: "Thane",
		Wards: []string{
			"Ward 1 – Kalyan East",
			"Ward 2 – Kalyan West",
			"Ward 3 – Dombivli East",
			"Ward 4 – Ulhas Nagar",
		},
	},
}

// NagarsevakDB maps ward name → list of nagarsevaks.
var NagarsevakDB = map[string][]Nagarsevak{
	"Ward 1 – Kasba Peth":      {{Name: "Suresh Patil", Party: "BJP", Initials: "SP"}, {Name: "Rekha Jadhav", Party: "NCP", Initials: "RJ"}},
	"Ward 2 – Shivajinagar":    {{Name: "Amol Kulkarni", Party: "INC", Initials: "AK"}, {Name: "Sunita More", Party: "BJP", Initials: "SM"}},
	"Ward 3 – Deccan Gymkhana": {{Name: "Priya Deshmukh", Party: "MNS", Initials: "PD"}, {Name: "Ravi Shinde", Party: "Shs", Initials: "RS"}},
	"Ward 4 – Kothrud":         {{Name: "Anjali Pawar", Party: "NCP", Initials: "AP"}, {Name: "Vijay Bhosale", Party: "BJP", Initials: "VB"}},
	"Ward A – Colaba":          {{Name: "Farhana Sheikh", Party: "INC", Initials: "FS"}, {Name: "Rajan Mehta", Party: "BJP", Initials: "RM"}},
	"Ward B – Mazgaon":         {{Name: "Abdul Siddiqui", Party: "AIMIM", Initials: "AS"}, {Name: "Priti Wagh", Party: "Shs", Initials: "PW"}},
	"Ward C – Girgaon":         {{Name: "Nilesh Rane", Party: "BJP", Initials: "NR"}, {Name: "Savita Naik", Party: "INC", Initials: "SN"}},
	"Ward D – Worli":           {{Name: "Aditya Thackeray", Party: "Shs", Initials: "AT"}, {Name: "Mangesh Dalvi", Party: "NCP", Initials: "MD"}},
	"Ward 1 – Civil Lines":     {{Name: "Sanjay Gawande", Party: "BJP", Initials: "SG"}, {Name: "Meena Tiwari", Party: "INC", Initials: "MT"}},
	"Ward 2 – Dharampeth":      {{Name: "Prakash Meshram", Party: "BSP", Initials: "PM"}, {Name: "Lata Dhakate", Party: "BJP", Initials: "LD"}},
	"Ward 3 – Lakadganj":       {{Name: "Rohit Nandurkar", Party: "INC", Initials: "RN"}, {Name: "Kavita Ingle", Party: "NCP", Initials: "KI"}},
	"Ward 4 – Mangalwari":      {{Name: "Deepak Kamble", Party: "BSP", Initials: "DK"}, {Name: "Sunita Raut", Party: "Shs", Initials: "SR"}},
	"Ward 1 – Kalyan East":     {{Name: "Mahesh Chaudhary", Party: "BJP", Initials: "MC"}, {Name: "Alka Patil", Party: "NCP", Initials: "AP"}},
	"Ward 2 – Kalyan West":     {{Name: "Ganesh Shinde", Party: "Shs", Initials: "GS"}, {Name: "Pushpa More", Party: "INC", Initials: "PM"}},
	"Ward 3 – Dombivli East":   {{Name: "Rahul Jadhav", Party: "MNS", Initials: "RJ"}, {Name: "Smita Kulkarni", Party: "BJP", Initials: "SK"}},
	"Ward 4 – Ulhas Nagar":     {{Name: "Sanjay Wagh", Party: "NCP", Initials: "SW"}, {Name: "Reena Bhoir", Party: "INC", Initials: "RB"}},
}
