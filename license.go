package main

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

const licensePrefix = "wawwd"

// expectedLicense 根据主板号计算正确授权码：大写 md5("wawwd-主板号")
func expectedLicense(machineID string) string {
	sum := md5.Sum([]byte(licensePrefix + "-" + machineID))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// licenseCachePath 授权缓存文件路径（用户目录下隐藏目录）
func licenseCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".action_composer")
	return filepath.Join(dir, "license"), nil
}

// loadCachedLicense 读取本机缓存的授权码
func loadCachedLicense() (string, bool) {
	path, err := licenseCachePath()
	if err != nil {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	code := strings.TrimSpace(string(data))
	return code, code != ""
}

// saveCachedLicense 永久缓存授权码到本机
func saveCachedLicense(code string) error {
	path, err := licenseCachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(code), 0o600)
}

// verifyLicense 校验授权码是否与主板号匹配
func verifyLicense(machineID, code string) bool {
	return strings.TrimSpace(code) != "" &&
		strings.EqualFold(strings.TrimSpace(code), expectedLicense(machineID))
}
