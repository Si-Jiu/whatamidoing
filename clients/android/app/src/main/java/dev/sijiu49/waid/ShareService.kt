package dev.sijiu49.waid

import android.app.AlarmManager
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Intent
import android.os.IBinder
import android.os.PowerManager
import androidx.core.app.NotificationCompat

/** 前台服务：在后台持续上报前台应用状态（通知保活）。 */
class ShareService : Service() {
    private var reporter: Reporter? = null
    private var wakeLock: PowerManager.WakeLock? = null
    private val nm by lazy { getSystemService(NotificationManager::class.java) }

    override fun onCreate() {
        super.onCreate()
        // PARTIAL_WAKE_LOCK：息屏时也保持 CPU 唤醒，避免系统冻结后停止上报。
        val pm = getSystemService(PowerManager::class.java)
        wakeLock = pm.newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, "whatamidoing:report").apply {
            setReferenceCounted(false)
            acquire()
        }
        createChannel()
        reporter = Reporter(this).apply {
            // 前台应用变化时实时刷新通知内容
            onAppChanged = { pkg -> updateNotification(pkg) }
        }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        startForeground(NOTIFICATION_ID, buildNotification(null))
        reporter?.start(ConfigStore(this))
        return START_STICKY
    }

    override fun onDestroy() {
        reporter?.stop()
        wakeLock?.let { if (it.isHeld) it.release() }
        // 自愈：共享仍开启时，若服务被系统/厂商 ROM 回收，延时拉起自己。
        if (ConfigStore(this).enabled) {
            scheduleRestart()
        }
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun createChannel() {
        nm.createNotificationChannel(
            NotificationChannel(CHANNEL_ID, "共享状态", NotificationManager.IMPORTANCE_LOW)
        )
    }

    private fun updateNotification(appLabel: String?) {
        nm.notify(NOTIFICATION_ID, buildNotification(appLabel))
    }

    private fun buildNotification(appLabel: String?): Notification {
        val contentIntent = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE
        )
        val text = if (appLabel.isNullOrBlank()) "正在上报前台应用" else "正在上报：$appLabel"
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_stat_share)
            .setContentTitle("正在共享前台状态")
            .setContentText(text)
            .setContentIntent(contentIntent)
            .setOngoing(true)
            .build()
    }

    /** 延时重启：解决部分 ROM 杀死服务后 START_STICKY 不生效的问题。 */
    private fun scheduleRestart() {
        val am = getSystemService(AlarmManager::class.java)
        val pi = PendingIntent.getForegroundService(
            this,
            0,
            Intent(this, ShareService::class.java),
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
        )
        am.set(
            AlarmManager.RTC_WAKEUP,
            System.currentTimeMillis() + RESTART_DELAY_MS,
            pi
        )
    }

    companion object {
        private const val CHANNEL_ID = "share"
        private const val NOTIFICATION_ID = 1
        private const val RESTART_DELAY_MS = 3_000L
    }
}
