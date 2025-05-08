package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

type Account struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AccountManager struct {
	XGOPath      string
	CookiesPath  string
	AccountsPath string
}

func NewAccountManager(xgoPath string) *AccountManager {
	return &AccountManager{
		XGOPath:      xgoPath,
		CookiesPath:  filepath.Join(xgoPath, "cookies"),
		AccountsPath: filepath.Join(xgoPath, "accounts.json"),
	}
}

func (am *AccountManager) LoadAccounts() ([]Account, error) {
	data, err := os.ReadFile(am.AccountsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read accounts file: %w", err)
	}

	var accounts []Account
	if err := json.Unmarshal(data, &accounts); err != nil {
		return nil, fmt.Errorf("failed to parse accounts file: %w", err)
	}

	return accounts, nil
}

func (am *AccountManager) SaveCookies(username string, cookies []*http.Cookie) error {
	if err := os.MkdirAll(am.CookiesPath, 0755); err != nil {
		return fmt.Errorf("failed to create cookies directory: %w", err)
	}

	cookieFile := filepath.Join(am.CookiesPath, username+".json")
	data, err := json.Marshal(cookies)
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}

	if err := os.WriteFile(cookieFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cookies file: %w", err)
	}

	return nil
}

func (am *AccountManager) LoadCookies(username string) ([]*http.Cookie, error) {
	cookieFile := filepath.Join(am.CookiesPath, username+".json")
	data, err := os.ReadFile(cookieFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cookies file: %w", err)
	}

	var cookies []*http.Cookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return nil, fmt.Errorf("failed to parse cookies file: %w", err)
	}

	return cookies, nil
}

func (am *AccountManager) CookiesExist(username string) bool {
	cookieFile := filepath.Join(am.CookiesPath, username+".json")
	_, err := os.Stat(cookieFile)
	return err == nil
}
