package store

import (
	"testing"
	"time"
)

func TestUpsertAndList(t *testing.T) {
	s := New(30 * time.Second)
	s.now = func() time.Time { return time.Unix(1000, 0) }
	started := time.Unix(900, 0)
	s.Upsert(DeviceState{
		DeviceID: "a", DeviceName: "电脑", Platform: "linux",
		App: "终端", AppStartedAt: started,
	})

	list := s.List()
	if len(list) != 1 {
		t.Fatalf("want 1 device, got %d", len(list))
	}
	d := list[0]
	if !d.Online {
		t.Error("want online=true right after report")
	}
	if !d.AppStartedAt.Equal(started) {
		t.Errorf("AppStartedAt not preserved: %v", d.AppStartedAt)
	}
	if d.WindowTitle != "" {
		t.Error("want empty WindowTitle when unset")
	}
}

func TestIdleMarksOffline(t *testing.T) {
	s := New(30 * time.Second)
	s.now = func() time.Time { return time.Unix(1000, 0) }
	s.Upsert(DeviceState{DeviceID: "a", DeviceName: "电脑", Platform: "linux", App: "终端"})

	s.now = func() time.Time { return time.Unix(1031, 0) } // 31s later, no new report
	list := s.List()
	if list[0].Online {
		t.Error("want online=false after idle timeout")
	}
}

func TestListChangesFiresOnce(t *testing.T) {
	s := New(30 * time.Second)
	s.now = func() time.Time { return time.Unix(1000, 0) }
	s.Upsert(DeviceState{DeviceID: "a", DeviceName: "电脑", Platform: "linux", App: "终端"})

	// No flip immediately after report.
	if changes := s.ListChanges(); len(changes) != 0 {
		t.Fatalf("want 0 changes, got %d", len(changes))
	}

	// Offline transition fires exactly once.
	s.now = func() time.Time { return time.Unix(1031, 0) }
	changes := s.ListChanges()
	if len(changes) != 1 {
		t.Fatalf("want 1 change, got %d", len(changes))
	}
	if changes[0].Online {
		t.Error("want the change to report online=false")
	}
	if changes = s.ListChanges(); len(changes) != 0 {
		t.Fatalf("want 0 changes on second poll, got %d", len(changes))
	}
}

func TestSameIDOverwrites(t *testing.T) {
	s := New(30 * time.Second)
	s.now = func() time.Time { return time.Unix(1000, 0) }
	s.Upsert(DeviceState{DeviceID: "a", DeviceName: "电脑", Platform: "linux", App: "终端"})
	s.Upsert(DeviceState{DeviceID: "a", DeviceName: "电脑", Platform: "linux", App: "浏览器"})

	list := s.List()
	if len(list) != 1 {
		t.Fatalf("same device_id should overwrite, got %d devices", len(list))
	}
	if list[0].App != "浏览器" {
		t.Errorf("want latest app, got %q", list[0].App)
	}
}
