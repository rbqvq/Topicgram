package captcha

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"

	botapi "github.com/OvyFlash/telegram-bot-api"
	"gitlab.com/go-extension/rand"
)

func generateMathCallbackData(secret string, challangeId, value uint64) string {
	hash := hmac.New(md5.New, []byte(secret))
	binary.Write(hash, binary.LittleEndian, challangeId)
	binary.Write(hash, binary.LittleEndian, value)

	return hex.EncodeToString(hash.Sum(nil))
}

func CheckMath(secret string, challangeId uint64, callbackData string) bool {
	signed := generateMathCallbackData(secret, challangeId, 0)
	return subtle.ConstantTimeCompare([]byte(signed), []byte(callbackData)) == 1
}

func NewMath(secret string, challangeId uint64) (string, botapi.InlineKeyboardMarkup) {
	add := rand.Crypto.IntN(10) < 5
	number1 := rand.Crypto.IntN(100)
	number2 := rand.Crypto.IntN(100)

	if number1 < number2 {
		number1, number2 = number2, number1
	}

	var problem string
	var answer int
	if add {
		problem = fmt.Sprintf("%d + %d = ?", number1, number2)
		answer = number1 + number2
	} else {
		problem = fmt.Sprintf("%d - %d = ?", number1, number2)
		answer = number1 - number2
	}

	buttons := make([]botapi.InlineKeyboardButton, 0, 4)
	buttons = append(buttons, botapi.NewInlineKeyboardButtonData(strconv.Itoa(answer), generateMathCallbackData(secret, challangeId, 0)))

	for len(buttons) < cap(buttons) {
		number := rand.Crypto.IntN(answer + 100)
		text := strconv.Itoa(number)
		if slices.ContainsFunc(buttons, func(button botapi.InlineKeyboardButton) bool {
			return button.Text == text
		}) {
			continue
		}

		number++ // make sure it not zero
		buttons = append(buttons, botapi.NewInlineKeyboardButtonData(text, generateMathCallbackData(secret, challangeId, uint64(number))))
	}

	rand.Crypto.Shuffle(len(buttons), func(i, j int) {
		buttons[i], buttons[j] = buttons[j], buttons[i]
	})
	return problem, botapi.NewInlineKeyboardMarkup(buttons)
}
