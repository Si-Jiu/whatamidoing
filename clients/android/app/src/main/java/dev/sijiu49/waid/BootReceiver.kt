package dev.sijiu49.waid

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent

/**
 * 开机自启：系统完成启动后，若共享已开启则自动重启前台服务，
 * 防止重启手机后上报中断（配合厂商自启动白名单效果最佳）。
 */
class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (!ConfigStore(context).enabled) return
        when (intent.action) {
            Intent.ACTION_BOOT_COMPLETED,
            Intent.ACTION_MY_PACKAGE_REPLACED,
            "android.intent.action.QUICKBOOT_POWERON", // 部分厂商（小米）的软重启广播
            -> context.startForegroundService(Intent(context, ShareService::class.java))
        }
    }
}
