package client

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/helloworlde/miwifi-exporter/internal/config"
	"github.com/helloworlde/miwifi-exporter/internal/errors"
	"github.com/helloworlde/miwifi-exporter/internal/logger"
	"github.com/helloworlde/miwifi-exporter/internal/models"
	httputil "github.com/helloworlde/miwifi-exporter/pkg/http"
)

type RouterClient interface {
	GetSystemStatus(ctx context.Context) (*models.SystemStatus, error)
	GetDeviceList(ctx context.Context) (*models.DeviceList, error)
	GetWanInfo(ctx context.Context) (*models.WanInfo, error)
	GetWifiDetails(ctx context.Context) (*models.WifiDetailAll, error)
	Authenticate(ctx context.Context) error
}

type MiWiFiClient struct {
	config     *config.Config
	httpClient *http.Client
	auth       *models.Auth
	retry      *errors.RetryHandler
}

func NewMiWiFiClient(cfg *config.Config) *MiWiFiClient {
	jar, _ := cookiejar.New(nil)
	
	// Create optimized HTTP client with connection pooling
	httpCfg := &httputil.Config{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		Timeout:             time.Duration(cfg.Router.Timeout) * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableKeepAlives:   false,
		MaxConnsPerHost:     30,
		DisableCompression:  false,
	}
	
	optimizedClient := httputil.NewOptimizedClient(httpCfg)
	optimizedClient.Jar = jar
	
	return &MiWiFiClient{
		config:     cfg,
		httpClient: optimizedClient,
		retry:      errors.NewRetryHandler(3, 30*time.Second, logger.Default),
	}
}

func (c *MiWiFiClient) Authenticate(ctx context.Context) error {
	return c.retry.WithRetry(func() error {
		return c.doAuthenticate(ctx)
	})
}

func (c *MiWiFiClient) doAuthenticate(ctx context.Context) error {
	router := &models.Router{
		IP:       c.config.Router.IP,
		Password: c.config.Router.Password,
		Headers: map[string]string{
			"Connection": "keep-alive",
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.72 Safari/537.36",
		},
	}

	if err := c.login(ctx, router); err != nil {
		return errors.NewAuthenticationError("router authentication failed", err)
	}

	c.auth = &models.Auth{
		URL:   router.Path,
		Token: router.Stok,
		Code:  200,
	}

	logger.Default.Info("Router authentication successful")
	return nil
}

func (c *MiWiFiClient) login(ctx context.Context, router *models.Router) error {
	// Get initial page to extract nonce and device ID
	if err := c.getInitialPage(ctx, router); err != nil {
		return err
	}

	// Get initialization info
	if err := c.getInitInfo(ctx, router); err != nil {
		return err
	}

	// Perform login
	return c.doLogin(ctx, router)
}

func (c *MiWiFiClient) getInitialPage(ctx context.Context, router *models.Router) error {
	webURL := fmt.Sprintf("http://%s/cgi-bin/luci/web", router.IP)
	
	req, err := http.NewRequestWithContext(ctx, "GET", webURL, nil)
	if err != nil {
		return err
	}

	for key, value := range router.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.NewNetworkError("failed to get initial page", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.NewInternalError("failed to read response body", err)
	}

	// Extract key and device ID
	key, deviceID, err := c.extractCredentials(string(body))
	if err != nil {
		return errors.NewInternalError("failed to extract credentials", err)
	}

	// Store credentials in router data
	router.Data = map[string]string{
		"key":       key,
		"device_id": deviceID,
	}

	return nil
}

func (c *MiWiFiClient) extractCredentials(body string) (string, string, error) {
	// Clean up the body
	body = strings.ReplaceAll(body, "\r", "")
	body = strings.ReplaceAll(body, "\n", "")
	body = strings.ReplaceAll(body, "\t", "")

	// Extract key
	keyRegex := regexp.MustCompile(`key:.*?'(.*?)',`)
	keyMatches := keyRegex.FindStringSubmatch(body)
	if len(keyMatches) < 2 {
		return "", "", fmt.Errorf("key not found in response")
	}

	// Extract device ID
	deviceIDRegex := regexp.MustCompile(`deviceId = '(.*?)';`)
	deviceIDMatches := deviceIDRegex.FindStringSubmatch(body)
	if len(deviceIDMatches) < 2 {
		return "", "", fmt.Errorf("device ID not found in response")
	}

	return keyMatches[1], deviceIDMatches[1], nil
}

func (c *MiWiFiClient) getInitInfo(ctx context.Context, router *models.Router) error {
	initInfoURL := fmt.Sprintf("http://%s/cgi-bin/luci/api/xqsystem/init_info", router.IP)
	
	req, err := http.NewRequestWithContext(ctx, "GET", initInfoURL, nil)
	if err != nil {
		return err
	}

	for key, value := range router.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.NewNetworkError("failed to get init info", err)
	}
	defer resp.Body.Close()

	var initInfo models.InitInfo
	if err := json.NewDecoder(resp.Body).Decode(&initInfo); err != nil {
		return errors.NewInternalError("failed to decode init info", err)
	}

	// Store init info
	router.Data["hardware"] = initInfo.Hardware
	router.Data["rom_version"] = initInfo.RomVersion
	router.Data["serial_number"] = initInfo.SerialNumber
	router.Data["router_name"] = initInfo.RouterName
	router.Data["new_encrypt_mode"] = strconv.Itoa(initInfo.NewEncryptMode)

	return nil
}

func (c *MiWiFiClient) doLogin(ctx context.Context, router *models.Router) error {
	pwd := router.Password
	key := router.Data["key"]
	deviceID := router.Data["device_id"]
	nonce := fmt.Sprintf("0_%s_%d_962", deviceID, time.Now().Unix())

	var password string
	if router.Data["new_encrypt_mode"] == "1" {
		a := c.hashSHA256(pwd + key)
		password = c.hashSHA256(nonce + a)
	} else {
		a := c.hashSHA1(pwd + key)
		password = c.hashSHA1(nonce + a)
	}

	loginURL := fmt.Sprintf("http://%s/cgi-bin/luci/api/xqsystem/login", router.IP)
	data := url.Values{}
	data.Set("username", "admin")
	data.Set("password", password)
	data.Set("logtype", "2")
	data.Set("nonce", nonce)

	req, err := http.NewRequestWithContext(ctx, "POST", loginURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	for key, value := range router.Headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.NewNetworkError("failed to login", err)
	}
	defer resp.Body.Close()

	var loginData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&loginData); err != nil {
		return errors.NewInternalError("failed to decode login response", err)
	}

	token, ok := loginData["token"].(string)
	if !ok {
		return errors.NewAuthenticationError("token not found in login response", nil)
	}

	path, ok := loginData["url"].(string)
	if !ok {
		return errors.NewAuthenticationError("URL not found in login response", nil)
	}

	router.Token = token
	router.Path = path

	// Extract stok from path
	stokRegex := regexp.MustCompile(`;stok=(.*?)/`)
	stokMatches := stokRegex.FindStringSubmatch(path)
	if len(stokMatches) < 2 {
		return errors.NewAuthenticationError("stok not found in path", nil)
	}
	router.Stok = stokMatches[1]

	return nil
}

func (c *MiWiFiClient) GetSystemStatus(ctx context.Context) (*models.SystemStatus, error) {
	if c.auth == nil {
		if err := c.Authenticate(ctx); err != nil {
			return nil, err
		}
	}

	var result *models.SystemStatus
	err := c.retry.WithRetry(func() error {
		status, err := c.getSystemStatus(ctx)
		if err != nil {
			return err
		}
		result = status
		return nil
	})
	
	return result, err
}

func (c *MiWiFiClient) getSystemStatus(ctx context.Context) (*models.SystemStatus, error) {
	url := fmt.Sprintf("http://%s/cgi-bin/luci/;stok=%s/api/misystem/status", 
		c.config.Router.IP, c.auth.Token)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.NewInternalError("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.NewNetworkError("failed to get system status", err)
	}
	defer resp.Body.Close()

	var status models.SystemStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		// If token is invalid, re-authenticate and retry
		if strings.Contains(err.Error(), "token") || status.Code != 0 {
			c.auth = nil
			return nil, errors.NewAuthenticationError("invalid token", err)
		}
		return nil, errors.NewInternalError("failed to decode system status", err)
	}

	return &status, nil
}

func (c *MiWiFiClient) GetDeviceList(ctx context.Context) (*models.DeviceList, error) {
	if c.auth == nil {
		if err := c.Authenticate(ctx); err != nil {
			return nil, err
		}
	}

	var result *models.DeviceList
	err := c.retry.WithRetry(func() error {
		devices, err := c.getDeviceList(ctx)
		if err != nil {
			return err
		}
		result = devices
		return nil
	})
	
	return result, err
}

func (c *MiWiFiClient) getDeviceList(ctx context.Context) (*models.DeviceList, error) {
	url := fmt.Sprintf("http://%s/cgi-bin/luci/;stok=%s/api/misystem/devicelist", 
		c.config.Router.IP, c.auth.Token)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.NewInternalError("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.NewNetworkError("failed to get device list", err)
	}
	defer resp.Body.Close()

	var deviceList models.DeviceList
	if err := json.NewDecoder(resp.Body).Decode(&deviceList); err != nil {
		// If token is invalid, re-authenticate and retry
		if strings.Contains(err.Error(), "token") || deviceList.Code != 0 {
			c.auth = nil
			return nil, errors.NewAuthenticationError("invalid token", err)
		}
		return nil, errors.NewInternalError("failed to decode device list", err)
	}

	return &deviceList, nil
}

func (c *MiWiFiClient) GetWanInfo(ctx context.Context) (*models.WanInfo, error) {
	if c.auth == nil {
		if err := c.Authenticate(ctx); err != nil {
			return nil, err
		}
	}

	var result *models.WanInfo
	err := c.retry.WithRetry(func() error {
		wan, err := c.getWanInfo(ctx)
		if err != nil {
			return err
		}
		result = wan
		return nil
	})
	
	return result, err
}

func (c *MiWiFiClient) getWanInfo(ctx context.Context) (*models.WanInfo, error) {
	url := fmt.Sprintf("http://%s/cgi-bin/luci/;stok=%s/api/xqnetwork/wan_info", 
		c.config.Router.IP, c.auth.Token)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.NewInternalError("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.NewNetworkError("failed to get WAN info", err)
	}
	defer resp.Body.Close()

	var wanInfo models.WanInfo
	if err := json.NewDecoder(resp.Body).Decode(&wanInfo); err != nil {
		// If token is invalid, re-authenticate and retry
		if strings.Contains(err.Error(), "token") || wanInfo.Code != 0 {
			c.auth = nil
			return nil, errors.NewAuthenticationError("invalid token", err)
		}
		return nil, errors.NewInternalError("failed to decode WAN info", err)
	}

	return &wanInfo, nil
}

func (c *MiWiFiClient) GetWifiDetails(ctx context.Context) (*models.WifiDetailAll, error) {
	if c.auth == nil {
		if err := c.Authenticate(ctx); err != nil {
			return nil, err
		}
	}

	var result *models.WifiDetailAll
	err := c.retry.WithRetry(func() error {
		wifi, err := c.getWifiDetails(ctx)
		if err != nil {
			return err
		}
		result = wifi
		return nil
	})
	
	return result, err
}

func (c *MiWiFiClient) getWifiDetails(ctx context.Context) (*models.WifiDetailAll, error) {
	url := fmt.Sprintf("http://%s/cgi-bin/luci/;stok=%s/api/xqnetwork/wifi_detail_all", 
		c.config.Router.IP, c.auth.Token)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.NewInternalError("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.NewNetworkError("failed to get WiFi details", err)
	}
	defer resp.Body.Close()

	var wifiDetails models.WifiDetailAll
	if err := json.NewDecoder(resp.Body).Decode(&wifiDetails); err != nil {
		// If token is invalid, re-authenticate and retry
		if strings.Contains(err.Error(), "token") || wifiDetails.Code != 0 {
			c.auth = nil
			return nil, errors.NewAuthenticationError("invalid token", err)
		}
		return nil, errors.NewInternalError("failed to decode WiFi details", err)
	}

	return &wifiDetails, nil
}

func (c *MiWiFiClient) hashSHA1(data string) string {
	h := sha1.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *MiWiFiClient) hashSHA256(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}