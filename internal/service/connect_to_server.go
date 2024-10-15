package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
)

// CreateSession создает сессию и сохраняет cookie в http.Client
func (sc *SessionClient) CreateSession() error {
	url := fmt.Sprintf("http://%s:51821/api/session", sc.IP)
	payload := map[string]string{"password": sc.Password}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := sc.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "connect.sid" {
			fmt.Println("Session Cookie Set:", cookie.Value)
		}
	}

	return nil
}

// NewSessionClient создает новый экземпляр SessionClient и устанавливает сессию
func NewSessionClient(ip, password string) (*SessionClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %v", err)
	}
	client := &SessionClient{
		IP:       ip,
		Password: password,
		Client: &http.Client{
			Jar: jar,
		},
	}

	if err := client.CreateSession(); err != nil {
		return nil, err
	}

	return client, nil
}

func (wg *Wireguard) PrepareAndSendGETRequest(address string) (*http.Response, error) {
	url := fmt.Sprintf("http://%s:51821/"+address, wg.session.IP)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := wg.session.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}
	return resp, nil
}

func (wg *Wireguard) PrepareAndSendPOSTRequest(address string, payload map[string]string) (*http.Response, error) {
	url := fmt.Sprintf("http://%s:51821/"+address, wg.session.IP)
	payloadBytes, err := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		wg.Logger.Error("failed to create request: ", "error", err)
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := wg.session.Client.Do(req)
	if err != nil {
		wg.Logger.Error("failed to send request: ", "error", err)
		return nil, fmt.Errorf("failed to send request: ", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		wg.Logger.Error("unexpected status code: ", "status code", resp.StatusCode, "body", body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, body)
	}

	return resp, nil
}
