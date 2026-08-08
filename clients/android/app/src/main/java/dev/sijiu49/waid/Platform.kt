package dev.sijiu49.waid

import android.os.Build

/**
 * 判断设备是否为小米 HyperOS / MIUI（含红米、POCO 子品牌）。
 *
 * 优先看厂商名；再用 MIUI/HyperOS 特有的系统属性兜底确认，
 * 避免刷机/定制 ROM 上报错厂商的情况。
 */
fun isMiui(): Boolean {
    val manufacturer = Build.MANUFACTURER?.lowercase() ?: ""
    if (manufacturer.contains("xiaomi") ||
        manufacturer.contains("redmi") ||
        manufacturer.contains("poco")
    ) {
        return true
    }
    // HyperOS / MIUI 专属属性（如 ro.miui.ui.version.name="V816...", ro.mi.os.version.name="OS1.0.1..."）
    return !getSystemProperty("ro.miui.ui.version.name").isNullOrEmpty() ||
        !getSystemProperty("ro.mi.os.version.name").isNullOrEmpty()
}

private fun getSystemProperty(name: String): String? = try {
    val clazz = Class.forName("android.os.SystemProperties")
    val get = clazz.getMethod("get", String::class.java)
    get.invoke(null, name) as? String
} catch (_: Exception) {
    null
}
