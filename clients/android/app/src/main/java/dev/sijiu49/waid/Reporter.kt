package dev.sijiu49.waid

import android.app.usage.UsageEvents
import android.app.usage.UsageStatsManager
import android.content.Context
import android.content.pm.PackageManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.concurrent.TimeUnit

/**
 * 轮询前台应用并上报到服务端。
 *
 * Android 只能拿到前台应用的包名（经 UsageStatsManager 的「使用情况访问」权限），
 * 无法读取窗口标题——`window_title` 恒为空（protocol 已注明）。
 */
class Reporter(private val context: Context) {
    private val client = OkHttpClient.Builder()
        .connectTimeout(5, TimeUnit.SECONDS)
        .readTimeout(5, TimeUnit.SECONDS)
        .build()

    private val scope = CoroutineScope(Dispatchers.IO)
    private var job: Job? = null

    @Volatile
    private var lastApp: String? = null
    private var appStartedAt: Long = System.currentTimeMillis()

    /** 前台应用变化回调（用于刷新常驻通知内容）。 */
    @Volatile
    var onAppChanged: ((String) -> Unit)? = null

    fun start(cfg: ConfigStore) {
        stop()
        job = scope.launch {
            while (isActive) {
                if (cfg.enabled && cfg.serverUrl.isNotBlank() && cfg.token.isNotBlank()) {
                    runCatching {
                        foregroundPackage()?.let { pkg ->
                            if (pkg != lastApp) {
                                lastApp = pkg
                                appStartedAt = System.currentTimeMillis()
                                onAppChanged?.invoke(pkg)
                            }
                            // app_id = 包名，window_title = 应用显示名（Android 无窗口标题）
                            report(cfg, pkg, appLabel(pkg))
                        }
                    }
                }
                delay((cfg.intervalSecs.coerceAtLeast(1) * 1000L))
            }
        }
    }

    fun stop() {
        job?.cancel()
        job = null
    }

    /** 最近一次 ACTIVITY_RESUMED 的包名即当前前台应用。 */
    private fun foregroundPackage(): String? {
        val usm = context.getSystemService(Context.USAGE_STATS_SERVICE) as UsageStatsManager
        val now = System.currentTimeMillis()
        val events = usm.queryEvents(now - 60_000, now)
        val event = UsageEvents.Event()
        var pkg: String? = null
        while (events.hasNextEvent()) {
            events.getNextEvent(event)
            if (event.eventType == UsageEvents.Event.ACTIVITY_RESUMED ||
                event.eventType == UsageEvents.Event.MOVE_TO_FOREGROUND
            ) {
                pkg = event.packageName
            }
        }
        return pkg
    }

    private fun appLabel(pkg: String): String = try {
        val pm = context.packageManager
        pm.getApplicationLabel(pm.getApplicationInfo(pkg, 0)).toString()
    } catch (_: PackageManager.NameNotFoundException) {
        pkg
    }

    private fun report(cfg: ConfigStore, pkg: String, windowTitle: String) {
        val url = cfg.serverUrl.trimEnd('/') + "/api/v1/report"
        // 设备身份由服务端按 token 确定，无需上报 device_id/device_name
        // app_id = 包名；window_title = 应用名称（Android 读不到窗口标题，用应用名填充）
        val body = JSONObject()
            .put("platform", "android")
            .put("app_id", pkg)
            .put("window_title", windowTitle)
            .put("app_started_at", iso(appStartedAt))
            .toString()

        val request = Request.Builder()
            .url(url)
            .addHeader("Authorization", "Bearer ${cfg.token}")
            .post(body.toRequestBody("application/json; charset=utf-8".toMediaType()))
            .build()

        client.newCall(request).execute().use { resp ->
            if (!resp.isSuccessful) throw RuntimeException("HTTP ${resp.code}")
        }
    }

    private fun iso(ts: Long): String {
        val fmt = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US)
        fmt.timeZone = java.util.TimeZone.getTimeZone("UTC")
        return fmt.format(Date(ts))
    }
}
