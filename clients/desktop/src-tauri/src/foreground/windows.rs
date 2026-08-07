//! Windows foreground detection via Win32: `GetForegroundWindow` + `GetWindowTextW`.
//! App name comes from the owning process's executable name.

use windows_sys::Win32::Foundation::HWND;
use windows_sys::Win32::System::Diagnostics::ToolHelp::{
    CreateToolhelp32Snapshot, Process32FirstW, Process32NextW, PROCESSENTRY32W, TH32CS_SNAPPROCESS,
};
use windows_sys::Win32::System::Threading::CloseHandle;
use windows_sys::Win32::UI::WindowsAndMessaging::{
    GetForegroundWindow, GetWindowTextLengthW, GetWindowTextW,
};

use super::info::ForegroundInfo;

pub fn current() -> Option<ForegroundInfo> {
    unsafe {
        let hwnd: HWND = GetForegroundWindow();
        if hwnd.is_null() {
            return None;
        }
        let window_title = window_text(hwnd);

        let mut pid: u32 = 0;
        windows_sys::Win32::UI::WindowsAndMessaging::GetWindowThreadProcessId(hwnd, &mut pid);
        let app = process_name(pid).unwrap_or_else(|| window_title.clone());

        if app.is_empty() && window_title.is_empty() {
            return None;
        }
        Some(ForegroundInfo { app, window_title })
    }
}

unsafe fn window_text(hwnd: HWND) -> String {
    let len = GetWindowTextLengthW(hwnd);
    if len <= 0 {
        return String::new();
    }
    let mut buf = vec![0u16; (len + 1) as usize];
    GetWindowTextW(hwnd, buf.as_mut_ptr(), len + 1);
    String::from_utf16_lossy(&buf[..len as usize])
}

/// Resolve the exe name of `pid` via a process snapshot.
unsafe fn process_name(pid: u32) -> Option<String> {
    let snapshot = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0);
    if snapshot.is_invalid() {
        return None;
    }
    let mut entry: PROCESSENTRY32W = std::mem::zeroed();
    entry.dwSize = std::mem::size_of::<PROCESSENTRY32W>() as u32;

    let mut result = None;
    if Process32FirstW(snapshot, &mut entry) != 0 {
        loop {
            if entry.th32ProcessID == pid {
                let name = String::from_utf16_lossy(&entry.szExeFile);
                result = Some(name.trim_end_matches(".exe").to_string());
                break;
            }
            if Process32NextW(snapshot, &mut entry) == 0 {
                break;
            }
        }
    }
    CloseHandle(snapshot);
    result
}
