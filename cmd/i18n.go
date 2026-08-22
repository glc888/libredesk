package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/knadh/go-i18n"
	"github.com/knadh/stuffbin"
	"github.com/zerodha/fastglue"
)

const (
	defLang = "en-US"
)

// handleGetI18nLang returns the JSON language pack for the given language code.
func handleGetI18nLang(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		lang = r.RequestCtx.UserValue("lang").(string)
	)
	i, err := loadI18nLang(lang, app.fs)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// return r.SendBytes(http.StatusOK, "application/json", i.JSON())

	// 拿到完整JSON字节
	fullJSON := i.JSON()

	// 反序列化为map
	var fullDict map[string]string
	if err := json.Unmarshal([]byte(fullJSON), &fullDict); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// 黑名单前缀，剔除后台管理key
	blockPrefixes := []string{
	"actions.",
	"activityLog.",
	"admin.",
	"agent.",
	"auth.",
	"automation.",
	"businessHour.",
	"confirm.",
	"contextLink.",
	"copilot.",
	"customAttribute.",
	"importer.",
	"macro.",
	"notification.",
	"oidc.",
	"report.",
	"replyBox.",
	"role.",
	"sla.",
	"status.",
	"tag.",
	"team.",
	"template.",
	"user.",
	"view.",
	"webhook.",
	"setup.",
	"shortcuts.",
	"navigation.",
	}


	filtered := make(map[string]string)
outer:
	for k, v := range fullDict {
		for _, prefix := range blockPrefixes {
			if strings.HasPrefix(k, prefix) {
				continue outer
			}
		}
		filtered[k] = v
	}

	// 序列化为压缩无缩进JSON
	outBytes, err := json.Marshal(filtered)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// ✅修复：fastglue 用 r.SetHeader() 设置响应头，不要碰 ResponseWriter
	r.SetHeader("Cache‑Control", "public, max‑age=86400")
	r.SetHeader("Content‑Type", "application/json; charset=utf‑8")

	return r.SendBytes(http.StatusOK, "application/json", outBytes)
}

// handleGetAvailableLanguages returns the list of available languages
// by reading all JSON files from the /i18n/ directory in the embedded filesystem.
func handleGetAvailableLanguages(r *fastglue.Request) error {
	app := r.Context.(*App)

	files, err := app.fs.Glob("/i18n/*.json")
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, "error listing language files", nil))
	}

	type langInfo struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}

	var langs []langInfo
	for _, f := range files {
		code := strings.TrimSuffix(filepath.Base(f), ".json")
		b, err := app.fs.Read(f)
		if err != nil {
			continue
		}

		var meta map[string]string
		if err := json.Unmarshal(b, &meta); err != nil {
			continue
		}

		name := meta["_.name"]
		if name == "" {
			name = code
		}
		langs = append(langs, langInfo{Code: code, Name: name})
	}

	sort.Slice(langs, func(i, j int) bool {
		return langs[i].Name < langs[j].Name
	})

	return r.SendEnvelope(langs)
}

// loadI18nLang loads the i18n language pack for the given language code.
func loadI18nLang(lang string, fs stuffbin.FileSystem) (*i18n.I18n, error) {
	// Helper function to read and initialize i18n language.
	readLang := func(lang string) ([]byte, error) {
		return fs.Read(fmt.Sprintf("/i18n/%s.json", lang))
	}

	// Read default language.
	b, err := readLang(defLang)
	if err != nil {
		return nil, envelope.NewError(envelope.GeneralError, "error reading default language", nil)
	}

	// Initialize with the default language.
	i, err := i18n.New(b)
	if err != nil {
		return nil, envelope.NewError(envelope.GeneralError, "error unmarshalling i18n language", nil)
	}

	// Load the selected language on top of it.
	if b, err = readLang(lang); err == nil {
		if err := i.Load(b); err != nil {
			return i, envelope.NewError(envelope.GeneralError, "error loading i18n language file", nil)
		}
	}

	return i, nil
}
