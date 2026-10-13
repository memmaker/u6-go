package game

// Full words for the topic list where the script's 4-letter keyword has no longer word in
// that NPC's script to spell it out (see topicWord). Picked from the game's prose and each
// NPC's reply; doubtful stubs are left out and listed in docs/topic-doubts.md.

// topicWords spells out a keyword for every NPC.
var topicWords = map[string]string{
	"advi": "advice", "allo": "allow", "aman": "amanda", "amul": "amulet", "anci": "ancient",
	"andr": "andreas", "arbe": "arbeth", "armo": "armor", "artu": "arturos", "avat": "avatar",
	"bedt": "bedtime", "beyv": "beyvin", "blac": "blackrock", "bles": "bless", "bloc": "block",
	"bole": "bolesh", "braw": "brawl", "brea": "bread", "brit": "british", "bumb": "bumberchutes",
	"bunc": "bunch", "buri": "buried", "capt": "captain", "cent": "center", "chil": "children",
	"civi": "civilization", "conv": "conversation", "cono": "conor", "cons": "conservatory",
	"crys": "crystal", "culh": "culham", "dang": "danger", "demo": "demons", "deza": "dezana",
	"dela": "delancy", "dori": "dorin", "drax": "draxinusom", "drin": "drink", "embr": "embroidery",
	"empi": "empire", "ench": "enchanted", "enli": "enlightenment", "erup": "eruption",
	"expe": "experiments", "fals": "false", "fami": "family", "firs": "first", "flam": "flame",
	"fung": "fungus", "garg": "gargoyles", "gert": "gertan", "gesh": "gesundheit", "ghos": "ghost",
	"gide": "gideon", "gord": "gordon", "grap": "grapes", "gris": "grison", "haun": "haunted",
	"hend": "hendle", "humb": "humble", "hora": "horance", "huma": "humans", "humi": "humility", "illu": "illusion",
	"impo": "important", "impr": "improvising", "inqu": "inquisitor", "jerr": "jerris",
	"kora": "koranada", "lear": "learn", "leav": "leave", "lodg": "lodging", "lyca": "lycaeum",
	"mamm": "mammoth", "mand": "mandrake", "mani": "manipulate", "manr": "manrel", "mant": "mantra",
	"mari": "marissa", "marn": "marney", "meri": "merideth", "mich": "michael", "mone": "money",
	"monu": "monuments", "mort": "mortude", "mous": "mouse", "murd": "murder", "mush": "mushroom",
	"naxa": "naxatilor", "netw": "network", "nico": "nicodemus", "noth": "nothing", "orde": "order",
	"peac": "peace", "plea": "please", "quak": "quake", "quen": "quenton", "ques": "question",
	"rang": "ranger", "rati": "rations", "rele": "release", "relt": "relthor", "resu": "resurrect",
	"ritu": "ritual", "robb": "robber", "roun": "round", "sacr": "sacrifice", "scho": "scholar",
	"segg": "seggallion", "shar": "share", "shee": "sheep", "shie": "shield", "shor": "shorty",
	"shri": "shrine", "silv": "silver", "smit": "smith", "snee": "sneeze", "spea": "speak",
	"spor": "spores", "stiv": "stivius", "swip": "swipe", "tabl": "tablet",
	"tele": "telescope", "timo": "timothy", "torc": "torch", "trad": "trade", "tran": "transcribe",
	"trav": "travel", "tren": "trenton", "tric": "tricks", "uber": "ubermon", "valk": "valkadesh",
	"veno": "venom", "volc": "volcano", "vort": "vortex", "wher": "where", "zolt": "zoltan",
	"ston": "stone", "airs": "airship",
}

// npcTopicWords overrides topicWords for one NPC (by script name); a keyword mapped to
// itself stays a stub.
var npcTopicWords = map[string]map[string]string{
	"Antonio":   {"star": "starfalcon"},
	"Marney":    {"lock": "locket"},
	"Yorl":      {"lock": "locket"},
	"Ybarra":    {"spar": "spare"},
	"Sin'Vraal": {"inte": "intelligence", "mari": "mari"},
	"Troy":      {"inte": "interconnected"},
	"Trenton":   {"humb": "humb"},
}

// scriptTopicWord is keyword spelled out from the tables for the NPC whose script is
// loaded, or "".
func (g *Game) scriptTopicWord(keyword string) string {
	const nameStart = 2
	var name []byte
	for i := nameStart; i < len(g.script) && g.script[i]&highBit == 0 && g.script[i] >= ' '; i++ {
		name = append(name, g.script[i])
	}
	if w, ok := npcTopicWords[string(name)][keyword]; ok {
		return w
	}
	return topicWords[keyword]
}
