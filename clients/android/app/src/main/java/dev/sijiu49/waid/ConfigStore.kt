package dev.sijiu49.waid

import android.content.Context
import android.content.SharedPreferences

/** 客户端配置，存于 SharedPreferences。 */
class ConfigStore(context: Context) {
    private val prefs: SharedPreferences =
        context.getSharedPreferences("config", Context.MODE_PRIVATE)

    var serverUrl: String
        get() = prefs.getString("server_url", "") ?: ""
        set(v) = prefs.edit().putString("server_url", v).apply()

    var token: String
        get() = prefs.getString("token", "") ?: ""
        set(v) = prefs.edit().putString("token", v).apply()

    var intervalSecs: Int
        get() = prefs.getInt("interval_secs", 5)
        set(v) = prefs.edit().putInt("interval_secs", v.coerceIn(1, 60)).apply()

    var enabled: Boolean
        get() = prefs.getBoolean("enabled", false)
        set(v) = prefs.edit().putBoolean("enabled", v).apply()

    /** 无感后台：从最近任务/后台预览隐藏（App 不退出，仅不可见）。 */
    var stealthBackground: Boolean
        get() = prefs.getBoolean("stealth_background", false)
        set(v) = prefs.edit().putBoolean("stealth_background", v).apply()
}
