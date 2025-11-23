package locale

import (
	"embed"
	"encoding/json"
	"fmt"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed *.json
var localeFS embed.FS

var localizer *goi18n.Localizer

// Init initializes the localization system.
func Init(lang string) error {
	bundle := goi18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load en.json
	_, err := bundle.LoadMessageFileFS(localeFS, "en.json")
	if err != nil {
		return fmt.Errorf("failed to load en.json: %w", err)
	}

	// Load ko.json
	_, err = bundle.LoadMessageFileFS(localeFS, "ko.json")
	if err != nil {
		return fmt.Errorf("failed to load ko.json: %w", err)
	}

	localizer = goi18n.NewLocalizer(bundle, lang, "en")
	return nil
}

// T translates a messageID.
func T(messageID string) string {
	if localizer == nil {
		return messageID
	}

	msg, err := localizer.Localize(&goi18n.LocalizeConfig{
		MessageID: messageID,
	})
	if err != nil {
		return messageID
	}
	return msg
}

// TWithData translates a messageID with template data.
func TWithData(messageID string, data map[string]interface{}) string {
	if localizer == nil {
		return messageID
	}

	msg, err := localizer.Localize(&goi18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
	})
	if err != nil {
		return messageID
	}
	return msg
}
