package dev.sijiu49.waid

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import top.yukonga.miuix.kmp.basic.Button
import top.yukonga.miuix.kmp.basic.Card
import top.yukonga.miuix.kmp.basic.Scaffold
import top.yukonga.miuix.kmp.basic.SmallTopAppBar
import top.yukonga.miuix.kmp.basic.Switch
import top.yukonga.miuix.kmp.basic.Text
import top.yukonga.miuix.kmp.basic.TextField

/** HyperOS / MIUI 专用设置页：使用 Miuix 组件呈现 MIUI 原生视觉。 */
@Composable
fun MiuixSettingsScreen(
    cfg: ConfigStore,
    batteryOptExempt: Boolean,
    onPromptBatteryOpt: () -> Unit,
    onEnabledToggle: (Boolean) -> Unit
) {
    var serverUrl by rememberSaveable { mutableStateOf(cfg.serverUrl) }
    var token by rememberSaveable { mutableStateOf(cfg.token) }
    var interval by rememberSaveable { mutableStateOf(cfg.intervalSecs.toString()) }
    var enabled by rememberSaveable { mutableStateOf(cfg.enabled) }
    var saved by remember { mutableStateOf(false) }

    Scaffold(
        topBar = { SmallTopAppBar(title = "whatamidoing") }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(
                    top = padding.calculateTopPadding(),
                    start = 26.dp,
                    end = 26.dp,
                    bottom = 26.dp
                )
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Card {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(16.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    Text("共享前台状态")
                    Switch(
                        checked = enabled,
                        onCheckedChange = {
                            enabled = it
                            onEnabledToggle(it)
                        }
                    )
                }
            }

            // 防后台限制：引导用户豁免电池优化（MIUI 上尤为重要）
            Card {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Text("后台运行保护")
                    if (batteryOptExempt) {
                        Text("已允许后台运行 ✓")
                    } else {
                        Text("开启后系统不会在息屏/后台时限制上报")
                        Button(
                            onClick = onPromptBatteryOpt,
                            modifier = Modifier.fillMaxWidth()
                        ) { Text("允许后台运行") }
                    }
                }
            }

            TextField(
                value = serverUrl,
                onValueChange = { serverUrl = it },
                label = "服务端地址",
                modifier = Modifier.fillMaxWidth()
            )
            TextField(
                value = token,
                onValueChange = { token = it },
                label = "设备 Token（管理面板添加设备后复制）",
                modifier = Modifier.fillMaxWidth()
            )
            TextField(
                value = interval,
                onValueChange = { interval = it },
                label = "上报间隔（秒）",
                modifier = Modifier.fillMaxWidth()
            )

            Button(
                onClick = {
                    cfg.serverUrl = serverUrl.trim()
                    cfg.token = token.trim()
                    cfg.intervalSecs = interval.toIntOrNull() ?: 5
                    saved = true
                    onEnabledToggle(cfg.enabled)
                },
                modifier = Modifier.fillMaxWidth()
            ) {
                Text(if (saved) "已保存 ✓" else "保存")
            }
        }
    }
}
