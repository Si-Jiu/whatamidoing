package dev.whatamidoing.client

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

    var deviceId: String
        get() = prefs.getString("device_id", "phone") ?: "phone"
        set(v) = prefs.edit().putString("device_id", v).apply()

    var deviceName: String
        get() = prefs.getString("device_name", "") ?: ""
        set(v) = prefs.edit().putString("device_name", v).apply()

    var intervalSecs: Int
        get() = prefs.getInt("interval_secs", 5)
        set(v) = prefs.edit().putInt("interval_secs", v.coerceIn(1, 60)).apply()

    var enabled: Boolean
        get() = prefs.getBoolean("enabled", false)
        set(v) = prefs.edit().putBoolean("enabled", v).apply()
}
