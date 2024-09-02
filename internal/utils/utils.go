package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/rand"
	"strings"
)

// GenerateStringID returns a random string containing only the following
// letters: 0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_
func GenerateStringID(length int) string {
	var (
		l  = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_"
		nl = len(l)
		id = ""
	)
	for i := 0; i < length; i++ {
		id = id + string(l[rand.Intn(nl)])
	}
	return id
}

// ValidMAC reports whether messageMAC64 (base64) is a valid HMAC tag for
// message.
func ValidMAC(message, messageMAC64, key string) (bool, error) {
	mac := hmac.New(sha256.New, []byte(key))
	if _, err := io.WriteString(mac, message); err != nil {
		return false, err
	}
	expectedMAC := mac.Sum(nil)
	mhmac, err := base64.URLEncoding.DecodeString(messageMAC64)
	if err != nil {
		return false, err
	}
	return hmac.Equal(mhmac, expectedMAC), nil
}

func NewHMAC(message, key string) string {
	hash := hmac.New(sha256.New, []byte(key))
	io.WriteString(hash, message)
	return base64.URLEncoding.EncodeToString(hash.Sum(nil))
}

func GenerateSenetence(wc int) string {
	words := []string{
		"bizarre",
		"honorable",
		"excellent",
		"phobic",
		"hop",
		"desire",
		"helpful",
		"machine",
		"medical",
		"object",
		"wound",
		"press",
		"butter",
		"belong",
		"vulgar",
		"quicksand",
		"glow",
		"death",
		"release",
		"hobbies",
		"even",
		"detail",
		"womanly",
		"premium",
		"crayon",
		"soft",
		"thought",
		"obey",
		"distinct",
		"dress",
		"tick",
		"crate",
		"coach",
		"opposite",
		"earth",
		"overwrought",
		"quixotic",
		"wheel",
		"gifted",
		"temper",
		"joke",
		"wish",
		"fairies",
		"royal",
		"stomach",
		"deafening",
		"dislike",
		"tramp",
		"branch",
		"marked",
		"handy",
		"meaty",
		"mindless",
		"representative",
		"puzzling",
		"kettle",
		"mass",
		"hollow",
		"look",
		"depressed",
		"tax",
		"voracious",
		"fretful",
		"risk",
		"arrogant",
		"crawl",
		"wonder",
		"magic",
		"bite",
		"mother",
		"hands",
		"ladybug",
		"damage",
		"disagreeable",
		"hole",
		"twist",
		"elbow",
		"bit",
		"believe",
		"maid",
		"discussion",
		"assorted",
		"mint",
		"handsomely",
		"yard",
		"current",
		"underwear",
		"necessary",
		"show",
		"develop",
		"square",
		"stupid",
		"canvas",
		"deranged",
		"drop",
		"messy",
		"poison",
		"uneven",
		"follow",
		"volcano",
	}
	s := strings.Title(words[rand.Int()%len(words)])
	for i := 0; i < wc; i++ {
		s += " " + words[rand.Int()%len(words)]
	}
	s += "."
	return s
}

func GenerateText() string {
	nsLow, nsHigh := 1, 7
	wcLow, wcHigh := 4, 20
	ns := nsLow + (rand.Int() % (nsHigh - nsLow))
	s := ""
	for i := 0; i < ns; i++ {
		if i != 0 {
			s += " "
		}
		wc := wcLow + (rand.Int() % (wcHigh - wcLow))
		s += GenerateSenetence(wc)
	}
	return s
}

// TruncateUnicodeString truncates s to a max length of length. It also
// guarantees that the returned string is of valid utf-8.
func TruncateUnicodeString(s string, length int) string {
	s = strings.ToValidUTF8(s, "")
	runes := []rune(s)
	if len(runes) > length {
		return string(runes[:length])
	}
	return s
}

// ExtractStringsFromMap returns a new map with all the string values from m. If
// trim is true, the string values of the returned map are space trimmed.
func ExtractStringsFromMap(m map[string]any, trim bool) map[string]string {
	strMap := make(map[string]string)
	for key, val := range m {
		if strVal, ok := val.(string); ok {
			if trim {
				strVal = strings.TrimSpace(strVal)
			}
			strMap[key] = strVal
		}
	}
	return strMap
}

// BreakUpOnCapitals breaks up a string s into words based on capital letters.
// For example, "HelloWorld" becomes "Hello World".
func BreakUpOnCapitals(s string) string {
	var words []string
	var word string
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			if word != "" {
				words = append(words, word)
			}
			word = string(r)
		} else {
			word += string(r)
		}
	}
	if word != "" {
		words = append(words, word)
	}
	return strings.Join(words, " ")
}

// CalculateBatchSize calculates the size of a batch of objects in bytes.
func CalculateBatchSize(objects []map[string]interface{}) (int, error) {
	var size int
	for _, obj := range objects {
		data, err := json.Marshal(obj)
		if err != nil {
			return 0, err
		}
		size += len(data)
	}
	return size, nil
}

// ConvertToMapSlice converts a slice of objects to a slice of maps.
func ConvertToMapSlice(objects []interface{}) ([]map[string]interface{}, error) {
	var documents []map[string]interface{}
	for _, obj := range objects {
		data, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		var doc map[string]interface{}
		err = json.Unmarshal(data, &doc)
		if err != nil {
			return nil, err
		}
		documents = append(documents, doc)
	}
	return documents, nil
}
