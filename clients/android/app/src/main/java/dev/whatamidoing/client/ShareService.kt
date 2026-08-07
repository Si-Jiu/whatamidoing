package dev.whatamidoing.client

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Intent
import android.os.IBinder
import androidx.core.app.NotificationCompat

/** 前台服务：在后台持续上报前台应用状态。 */
class ShareService : Service() {
    private var reporter: Reporter? = null

    override fun onCreate() {
        super.onCreate()
        createChannel()
        reporter = Reporter(this)
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        startForeground(NOTIFICATION_ID, buildNotification())
        reporter?.start(ConfigStore(this))
        return START_STICKY
    }

    override fun onDestroy() {
        reporter?.stop()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun createChannel() {
        val nm = getSystemService(NotificationManager::class.java)
        nm.createNotificationChannel(
            NotificationChannel(CHANNEL_ID, "共享状态", NotificationManager.IMPORTANCE_LOW)
        )
    }

    private fun buildNotification(): Notification {
        val contentIntent = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE
        )
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_stat_share)
            .setContentTitle("正在共享前台状态")
            .setContentText("whatamidoing 正在上报当前应用")
            .setContentIntent(contentIntent)
            .setOngoing(true)
            .build()
    }

    companion object {
        private const val CHANNEL_ID = "share"
        private const val NOTIFICATION_ID = 1
    }
}
