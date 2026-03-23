package qqwry

import (
	"os"
	"testing"
)

func TestResolveDownloadURL(t *testing.T) {
	if resolveDownloadURL("") != DefaultQQWryURL {
		t.Fatalf("should use default qqwry download url when empty")
	}
	customURL := "https://example.com/qqwry.dat"
	if resolveDownloadURL(customURL) != customURL {
		t.Fatalf("should use configured qqwry download url")
	}
}

// TestDownload 测试qqwry.dat文件下载功能
func TestDownload(t *testing.T) {
	// 检查文件是否已存在
	info, err := os.Stat("qqwry.dat")
	if err == nil {
		// 文件已存在，检查文件大小是否合理（至少1MB）
		if info.Size() < 10*1024*1024 {
			t.Errorf("qqwry.dat file too small: %d bytes", info.Size())
			return
		}
		t.Logf("qqwry.dat already exists, skip download. File size: %d bytes", info.Size())
		return
	}

	// 文件不存在，执行下载
	err = download("")
	if err != nil {
		t.Errorf("download failed: %v", err)
		return
	}

	// 检查文件是否存在
	info, err = os.Stat("qqwry.dat")
	if err != nil {
		t.Errorf("qqwry.dat file not found: %v", err)
		return
	}

	// 检查文件大小是否合理（至少1MB）
	if info.Size() < 1024*1024 {
		t.Errorf("qqwry.dat file too small: %d bytes", info.Size())
		return
	}

	t.Logf("Download successful! File size: %d bytes", info.Size())
}

// TestQQwryIntegration 测试qqwry整体功能集成
func TestQQwryIntegration(t *testing.T) {
	// 检查文件是否已存在
	_, err := os.Stat("qqwry.dat")
	if err != nil {
		// 文件不存在，执行下载
		err = download("")
		if err != nil {
			t.Errorf("download failed: %v", err)
			return
		}
		t.Log("qqwry.dat downloaded successfully")
	} else {
		t.Log("qqwry.dat already exists, skip download")
	}

	// 初始化QQwry数据库
	db := New("")
	if db == nil {
		t.Error("New() returned nil")
		return
	}

	// 测试IP查询功能
	testIPs := []string{
		"114.114.114.114", // 国内DNS服务器
		"8.8.8.8",         // 谷歌DNS
		"127.0.0.1",       // 本地地址
	}

	for _, ip := range testIPs {
		area := db.Area(ip)
		t.Logf("IP: %s, Area: %s", ip, area)
		// 本地地址可能返回空，但其他IP应该返回非空结果
		if ip != "127.0.0.1" && area == "" {
			t.Errorf("Area lookup failed for IP: %s", ip)
		}
	}
}
