package data

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLifecycleAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")

	s, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.IsAdminInitialized() {
		t.Error("新库不应已初始化")
	}

	if err := s.SetAdminPassword("hash123"); err != nil {
		t.Fatal(err)
	}
	if !s.IsAdminInitialized() {
		t.Error("设置密码后应已初始化")
	}
	if s.AdminPasswordHash() != "hash123" {
		t.Error("管理员 hash 不匹配")
	}

	dev, err := s.AddDevice("我的电脑", "linux", "archlinux")
	if err != nil {
		t.Fatal(err)
	}
	if dev.ID == "" || dev.Token == "" {
		t.Error("设备 ID/token 不应为空")
	}
	if !strings.HasPrefix(dev.Token, "tok_") {
		t.Errorf("token 前缀错误: %q", dev.Token)
	}
	if dev.Platform != "linux" {
		t.Errorf("平台不符: %q", dev.Platform)
	}
	if dev.Distro != "archlinux" {
		t.Errorf("发行版不符: %q", dev.Distro)
	}

	got, ok := s.DeviceByToken(dev.Token)
	if !ok || got.ID != dev.ID {
		t.Error("按 token 查找设备失败")
	}
	if _, ok := s.DeviceByToken("nope"); ok {
		t.Error("未知 token 不应命中")
	}

	// 重新加载验证持久化
	s2, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	if !s2.IsAdminInitialized() {
		t.Error("重载后应仍初始化")
	}
	devs := s2.Devices()
	if len(devs) != 1 || devs[0].Name != "我的电脑" || devs[0].Token != dev.Token || devs[0].Platform != "linux" || devs[0].Distro != "archlinux" {
		t.Errorf("重载后设备数据不符: %+v", devs)
	}

	// 删除设备
	if err := s2.RemoveDevice(dev.ID); err != nil {
		t.Fatal(err)
	}
	if len(s2.Devices()) != 0 {
		t.Error("删除后设备列表应为空")
	}
	if err := s2.RemoveDevice("nonexistent"); err == nil {
		t.Error("删除不存在设备应报错")
	}

	// 设置网页密码
	if err := s.SetViewerPasswordHash("viewer-hash"); err != nil {
		t.Fatal(err)
	}
	if s.ViewerPasswordHash() != "viewer-hash" {
		t.Error("网页密码 hash 不符")
	}
}
