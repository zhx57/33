<script setup lang="ts">
import { getPubDate } from '~/share/Time'
import { Notice, Request } from '~/share/Tools'

const store = useMainStore()
const pidNameKV = computed(() => store.pidNameKV)
const loading = computed(() => store.loading)

const tasksSwitch = ref<boolean>(false)
const limit = ref<number>(0)
const loadingList = ref<boolean>(false)

const settings = ref<{ global_limit: string; personal_limit: string; global_cooldown: string }>({ global_limit: '5', personal_limit: '', global_cooldown: '0' })

const tasksList = ref<
    {
        id: number
        uid: number
        pid: number
        fname: string
        tid: number
        last_floor: number
        last_replied_pid: number
        last_reply_time: number
        last_status: string
        last_error: string
        last_check_time: number
        log: string
        reply_content: string
        reply_content_list: string
        reply_interval: number
        reply_interval_min: number
        reply_interval_max: number
        reply_probability: number
        enabled: number
        retry_count: number
        trigger_mode: string
        reply_target: string
        allow_replied: number
        match_keywords: string
        max_count: number
        active_time_start: string
        active_time_end: string
        success_count: number
    }[]
>([])

const editMode = ref<boolean>(false)
const editingTaskId = ref<number>(0)

const taskToAdd = ref<{
    pid: number
    fname: string
    tid: string
    reply_content_list: string
    reply_interval_min: number
    reply_interval_max: number
    reply_probability: number
    trigger_mode: string
    reply_target: string
    allow_replied: boolean
    match_keywords: string
    enabled: boolean
    max_count: number
    active_time_start: string
    active_time_end: string
}>({
    pid: 0,
    fname: '',
    tid: '',
    reply_content_list: '',
    reply_interval_min: 300,
    reply_interval_max: 0,
    reply_probability: 100,
    trigger_mode: 'new_floor',
    reply_target: 'floor',
    allow_replied: false,
    match_keywords: '',
    enabled: true,
    max_count: 0,
    active_time_start: '',
    active_time_end: ''
})

const testForm = ref<{
    pid: number
    fname: string
    tid: string
    reply_content: string
    trigger_mode: string
    reply_target: string
    allow_replied: boolean
    match_keywords: string
}>({
    pid: 0,
    fname: '',
    tid: '',
    reply_content: '',
    trigger_mode: 'new_floor',
    reply_target: 'floor',
    allow_replied: false,
    match_keywords: ''
})

const testResult = ref<string>('')

const helpModalKey = ref<string>('')
const helpModalVisible = ref<boolean>(false)

const helpTexts: Record<string, string> = {
    interval: '回帖间隔\n两次自动回帖之间至少要等多久（单位：秒）。\n举例：填 300，就是回完一帖后，至少等 5 分钟才会回下一帖。\n建议：别设太短，贴吧有频率限制，太容易被封号。',
    probability: '回帖概率\n到了该回帖的时候，实际去回帖的几率。\n举例：填 50，就是每次有一半的可能回帖，一半的可能啥也不干。\n填 100 就是每次都回。想降低被发现的风险，可以调低一点。',
    trigger: '触发模式 — 决定机器人什么时候回帖\n新楼层即回复：监控指定帖子，有新楼层时自动回复，适合跟帖互动。\n关键词匹配回复：监控指定帖子，只有楼层内容命中关键词才回复，适合问答、提醒和定向互动。',
    keywords: '匹配关键词\n每行写一个关键词，命中任意一个就算匹配。\n用于匹配楼层内容，命中后可在回复内容中使用 {username} 代入楼层用户名。\n举例：\n求助\n怎么弄\n报错',
    reply_target: '回复目标\n主楼层回复：直接回在帖子里，大家都能看到。\n楼中楼回复：在别人的楼层里回复，只有那层楼里的人能看到。需要关键词模式。',
    allow_replied: '允许回复已回复过的楼层\n打开后，每次都会重新扫描所有楼层（包括以前回复过的），可能会重复回同一层。\n默认关闭，只回复新楼层。',
    content: '回复内容怎么写\n支持用花括号变量自动替换：\n{floor} — 当前回帖数\n{time} — 当前时间\n{date} — 当前日期\n{tid} — 帖子 ID\n{username} — 楼层用户名（只在关键词模式下有用）\n举例：第{floor}楼打卡！今天是{date}',
    fname: '贴吧名称\n就是目标贴吧的名字，不是网址。\n对的：天堂鸡汤\n错的：https://tieba.baidu.com/f?kw=天堂鸡汤\n打开贴吧页面，顶上看那个吧名就是。',
    tid: '帖子 ID\n帖子的纯数字编号，从浏览器地址栏就能看到。\n举例：链接是 https://tieba.baidu.com/p/12345678，那 ID 就是 12345678',
    pid: '发帖账号\n选一个你绑定了的百度贴吧账号，机器就用这个号去回帖。\n如果这里没得选，说明你还没绑定账号，去账号管理绑定一下。',
    reply_content_list: '回复内容（随机发送）\n每行写一条回复内容，系统会随机选择一条发送。\n写多条可以避免每次回复都一样，更自然。\n举例：\n打卡打卡\n今天也来签到了\n继续坚持',
    reply_interval_range: '回复间隔范围\n设置两次回复之间的等待时间范围。\n最小间隔和最大间隔可以不同，系统会在范围内随机取值。\n举例：最小60秒、最大300秒，意味着每次回复后等1-5分钟不等。\n如果最小和最大一样，就是固定间隔。',
    max_count: '最大执行次数\n设置这个任务最多回复几次，0表示无限次。\n举例：填10，就是回复10次后自动停止。\n适合只想回复固定次数的场景。',
    active_time: '活跃时间窗口\n设置每天允许自动回复的时间段。\n举例：08:00-22:00，就只在早8点到晚10点之间回复。\n留空表示全天24小时都可以回复。\n支持跨午夜，如 22:00-08:00。',
    global_cooldown:
        '全局回复冷却时间（秒）\n设置任意两次回复之间的最小间隔，跨所有任务生效。\n举例：填300，就是任何任务回复后，所有任务至少等5分钟才能再回复。\n0表示不启用全局冷却，每个任务按自己的间隔独立运行。\n适合多任务场景，避免短时间内多个任务连续回复。'
}

const showHelp = (key: string) => {
    helpModalKey.value = key
    helpModalVisible.value = true
}

const updateTasksSwitch = () => {
    Request(store.basePath + '/plugins/weltolk_autoreply/switch', {
        headers: {
            Authorization: store.authorization
        },
        method: 'POST'
    }).then((res) => {
        if (res.code !== 200) {
            Notice(res.message, 'error')
            return
        }
        // 切换后重新获取真实状态，确保与数据库同步
        Request(store.basePath + '/plugins/weltolk_autoreply/switch', {
            headers: {
                Authorization: store.authorization
            }
        }).then((res2) => {
            if (res2.code === 200) {
                tasksSwitch.value = res2.data
            }
        })
    })
}

const getTasksList = () => {
    loadingList.value = true
    Request(store.basePath + '/plugins/weltolk_autoreply/list', {
        headers: {
            Authorization: store.authorization
        }
    })
        .then((res) => {
            if (res.code !== 200) {
                Notice(res.message, 'error')
                return
            }
            tasksList.value = res.data?.tasks || []
            limit.value = res.data?.limit || 0
            if (limit.value < 0) {
                limit.value = 0
            }
        })
        .catch((e) => {
            console.error(e)
            Notice(e.toString(), 'error')
        })
        .finally(() => {
            loadingList.value = false
        })
}

const resetForm = () => {
    editMode.value = false
    editingTaskId.value = 0
    taskToAdd.value = {
        pid: 0,
        fname: '',
        tid: '',
        reply_content_list: '',
        reply_interval_min: 300,
        reply_interval_max: 0,
        reply_probability: 100,
        trigger_mode: 'new_floor',
        reply_target: 'floor',
        allow_replied: false,
        match_keywords: '',
        enabled: true,
        max_count: 0,
        active_time_start: '',
        active_time_end: ''
    }
}

const addTask = (enabledOverride?: number) => {
    if (!Object.keys(pidNameKV.value).includes(taskToAdd.value.pid.toString())) {
        Notice('请选择百度账号', 'error')
        return
    }
    const enabled = enabledOverride !== undefined ? enabledOverride : taskToAdd.value.enabled ? 1 : 0
    Request(store.basePath + '/plugins/weltolk_autoreply/list', {
        headers: {
            Authorization: store.authorization,
            'Content-Type': 'application/x-www-form-urlencoded'
        },
        method: 'PATCH',
        body: new URLSearchParams({
            pid: taskToAdd.value.pid.toString(),
            fname: taskToAdd.value.fname,
            tid: taskToAdd.value.tid,
            reply_content: taskToAdd.value.reply_content_list.split('\n')[0] || '',
            reply_content_list: taskToAdd.value.reply_content_list,
            reply_interval: taskToAdd.value.reply_interval_min.toString(),
            reply_interval_min: taskToAdd.value.reply_interval_min.toString(),
            reply_interval_max: taskToAdd.value.reply_interval_max.toString(),
            reply_probability: taskToAdd.value.reply_probability.toString(),
            trigger_mode: taskToAdd.value.trigger_mode,
            reply_target: taskToAdd.value.reply_target,
            allow_replied: taskToAdd.value.allow_replied ? '1' : '0',
            match_keywords: taskToAdd.value.match_keywords,
            enabled: enabled.toString(),
            max_count: taskToAdd.value.max_count.toString(),
            active_time_start: taskToAdd.value.active_time_start,
            active_time_end: taskToAdd.value.active_time_end
        }).toString()
    }).then((res) => {
        if (res.code !== 200 && res.code !== 201 && res.code !== 204) {
            Notice(res.message, 'error')
            return
        }
        Notice('添加成功', 'success')
        getTasksList()
        resetForm()
    })
}

const editTask = (task: (typeof tasksList.value)[0]) => {
    editMode.value = true
    editingTaskId.value = task.id
    taskToAdd.value = {
        pid: task.pid,
        fname: task.fname,
        tid: task.tid.toString(),
        reply_content_list: task.reply_content_list || task.reply_content || '',
        reply_interval_min: task.reply_interval_min || task.reply_interval || 300,
        reply_interval_max: task.reply_interval_max || 0,
        reply_probability: task.reply_probability,
        trigger_mode: task.trigger_mode || 'new_floor',
        reply_target: task.reply_target || 'floor',
        allow_replied: task.allow_replied === 1,
        match_keywords: task.match_keywords || '',
        enabled: task.enabled === 1,
        max_count: task.max_count || 0,
        active_time_start: task.active_time_start || '',
        active_time_end: task.active_time_end || ''
    }
}

const saveEditTask = () => {
    if (!Object.keys(pidNameKV.value).includes(taskToAdd.value.pid.toString())) {
        Notice('请选择百度账号', 'error')
        return
    }
    Request(store.basePath + '/plugins/weltolk_autoreply/list/' + editingTaskId.value, {
        headers: {
            Authorization: store.authorization,
            'Content-Type': 'application/x-www-form-urlencoded'
        },
        method: 'PUT',
        body: new URLSearchParams({
            pid: taskToAdd.value.pid.toString(),
            fname: taskToAdd.value.fname,
            tid: taskToAdd.value.tid,
            reply_content: taskToAdd.value.reply_content_list.split('\n')[0] || '',
            reply_content_list: taskToAdd.value.reply_content_list,
            reply_interval: taskToAdd.value.reply_interval_min.toString(),
            reply_interval_min: taskToAdd.value.reply_interval_min.toString(),
            reply_interval_max: taskToAdd.value.reply_interval_max.toString(),
            reply_probability: taskToAdd.value.reply_probability.toString(),
            trigger_mode: taskToAdd.value.trigger_mode,
            reply_target: taskToAdd.value.reply_target,
            allow_replied: taskToAdd.value.allow_replied ? '1' : '0',
            match_keywords: taskToAdd.value.match_keywords,
            enabled: taskToAdd.value.enabled ? '1' : '0',
            max_count: taskToAdd.value.max_count.toString(),
            active_time_start: taskToAdd.value.active_time_start,
            active_time_end: taskToAdd.value.active_time_end
        }).toString()
    }).then((res) => {
        if (res.code !== 200) {
            Notice(res.message, 'error')
            return
        }
        Notice('保存成功', 'success')
        getTasksList()
        resetForm()
    })
}

const deleteTask = (id: number) => {
    if (id <= 0) {
        return
    }
    Request(store.basePath + '/plugins/weltolk_autoreply/list/' + id, {
        headers: {
            Authorization: store.authorization
        },
        method: 'DELETE'
    }).then((res) => {
        if (res.code !== 200) {
            Notice(res.message, 'error')
            return
        }
        Notice('已删除任务: ' + id, 'success')
        tasksList.value = tasksList.value.filter((x) => x.id !== id)
    })
}

const toggleTaskEnabled = (task: (typeof tasksList.value)[0]) => {
    Request(store.basePath + '/plugins/weltolk_autoreply/list/' + task.id + '/toggle', {
        headers: {
            Authorization: store.authorization
        },
        method: 'POST'
    }).then((res) => {
        if (res.code !== 200) {
            Notice(res.message, 'error')
            return
        }
        task.enabled = task.enabled === 1 ? 0 : 1
        Notice(task.enabled === 1 ? '已启用' : '已暂停', 'success')
    })
}

const emptyAllTasks = () => {
    Request(store.basePath + '/plugins/weltolk_autoreply/list/empty', {
        headers: {
            Authorization: store.authorization
        },
        method: 'POST'
    }).then((res) => {
        if (res.code !== 200) {
            Notice(res.message, 'error')
            return
        }
        Notice('已清空全部任务', 'success')
        tasksList.value = []
    })
}

const runTest = () => {
    testResult.value = ''
    if (!Object.keys(pidNameKV.value).includes(testForm.value.pid.toString())) {
        Notice('请选择百度账号', 'error')
        return
    }
    Request(store.basePath + '/plugins/weltolk_autoreply/test', {
        headers: {
            Authorization: store.authorization,
            'Content-Type': 'application/x-www-form-urlencoded'
        },
        method: 'POST',
        body: new URLSearchParams({
            pid: testForm.value.pid.toString(),
            fname: testForm.value.fname,
            tid: testForm.value.tid,
            reply_content: testForm.value.reply_content,
            trigger_mode: testForm.value.trigger_mode,
            reply_target: testForm.value.reply_target,
            allow_replied: testForm.value.allow_replied ? '1' : '0',
            match_keywords: testForm.value.match_keywords
        }).toString()
    }).then((res) => {
        if (res.code !== 200) {
            Notice(res.message, 'error')
            testResult.value = res.message
            return
        }
        const data = res.data
        if (data && typeof data === 'object') {
            if (data.success) {
                testResult.value = '回帖成功'
            } else {
                testResult.value = '回帖失败：' + (data.error_msg || data.error_code || '未知错误')
            }
            if (data.need_vcode) {
                testResult.value += '（需要验证码）'
            }
        } else {
            testResult.value = res.message || '完成'
        }
        Notice(testResult.value, 'success')
    })
}

const saveSettings = () => {
    const params: Record<string, string> = {}
    if (settings.value.personal_limit !== '') {
        params.personal_limit = settings.value.personal_limit
    }
    if (settings.value.global_cooldown !== '') {
        params.global_cooldown = settings.value.global_cooldown
    }
    Request(store.basePath + '/plugins/weltolk_autoreply/settings', {
        headers: {
            Authorization: store.authorization,
            'Content-Type': 'application/x-www-form-urlencoded'
        },
        method: 'PUT',
        body: new URLSearchParams(params).toString()
    }).then((res) => {
        if (res.code !== 200) {
            Notice(res.message, 'error')
            return
        }
        Notice('设置已保存', 'success')
        if (res.data) {
            settings.value.global_limit = res.data.global_limit || '5'
            settings.value.personal_limit = res.data.personal_limit || ''
            settings.value.global_cooldown = res.data.global_cooldown || '0'
        }
    })
}

const parseLogs = (log_: string = '') => {
    if (!log_) {
        return []
    }
    return log_
        .split('<br>')
        .map((log) => {
            if (!log || log.length < 20) {
                return null
            }
            return {
                text: log
            }
        })
        .filter((x) => x)
        .slice(0, 200)
        .reverse()
}

const triggerModeText = (mode: string) => {
    switch (mode) {
        case 'new_floor':
            return '新楼层'
        case 'keyword':
            return '关键词'
        default:
            return mode
    }
}

const replyTargetText = (target: string) => {
    switch (target) {
        case 'floor':
            return '回复楼层'
        case 'subpost':
            return '楼中楼'
        default:
            return target
    }
}

const statusText = (status: string) => {
    switch (status) {
        case 'ok':
            return '成功'
        case 'skipped':
            return '跳过'
        case 'error':
            return '错误'
        case 'vcode':
            return '验证码'
        default:
            return status || '未执行'
    }
}

const taskStatusDisplay = (task: (typeof tasksList.value)[0]) => {
    if (!task.last_status && task.last_check_time === 0) {
        return { text: '尚未执行', class: 'text-gray-500' }
    }
    if (task.enabled === 1) {
        if (task.last_status === 'error') {
            return { text: '异常', class: 'text-yellow-500' }
        } else if (task.last_status === 'vcode') {
            return { text: '需验证码', class: 'text-yellow-500' }
        } else {
            return { text: '运行中', class: 'text-green-500' }
        }
    } else {
        if (task.last_status === 'error') {
            return { text: '失败已停用', class: 'text-red-500' }
        } else {
            return { text: '已暂停', class: 'text-gray-500' }
        }
    }
}

onMounted(() => {
    getTasksList()
    Request(store.basePath + '/plugins/weltolk_autoreply/switch', {
        headers: {
            Authorization: store.authorization
        }
    }).then((res) => {
        if (res.code !== 200) {
            Notice(res.message, 'error')
            return
        }
        tasksSwitch.value = res.data
    })
    Request(store.basePath + '/plugins/weltolk_autoreply/settings', {
        headers: {
            Authorization: store.authorization
        }
    }).then((res) => {
        if (res.code !== 200) {
            Notice(res.message, 'error')
            return
        }
        settings.value = res.data
    })
})
</script>

<template>
    <div class="px-3 py-2">
        <h3 class="text-2xl mb-4">设置</h3>

        <button :class="{ 'bg-sky-500': !tasksSwitch, 'bg-pink-500': tasksSwitch, 'rounded-lg': true, 'px-3': true, 'py-1': true, 'text-gray-100': true, 'transition-colors': true }" @click="updateTasksSwitch">
            {{ tasksSwitch ? '已开启自动回帖' : '已停止自动回帖' }}
        </button>

        <div class="my-5">
            <h4 class="my-2 text-xl">任务数量限额</h4>
            <div class="flex gap-3 items-center max-w-[48em]">
                <div class="grow">
                    <label>全局限额（仅管理员可改）</label>
                    <input type="number" v-model="settings.global_limit" class="bg-gray-100 dark:bg-gray-900 dark:text-gray-100 form-input w-full rounded-xl mt-1" disabled />
                </div>
                <div class="grow">
                    <label>个人限额（0 表示使用全局）</label>
                    <input type="number" v-model="settings.personal_limit" class="bg-gray-100 dark:bg-gray-900 dark:text-gray-100 form-input w-full rounded-xl mt-1" min="0" />
                </div>
            </div>
        </div>

        <div class="my-5">
            <h4 class="my-2 text-xl">全局回复冷却 <button @click="showHelp('global_cooldown')" class="inline-block text-xs bg-sky-500 text-white rounded px-1">?</button></h4>
            <div class="max-w-[48em]">
                <label>冷却时间（秒，0=不启用）</label>
                <input type="number" v-model="settings.global_cooldown" class="bg-gray-100 dark:bg-gray-900 dark:text-gray-100 form-input w-full rounded-xl mt-1" min="0" placeholder="0" />
                <p class="text-xs text-gray-500 mt-1">设置后，任意任务回复成功后，所有任务至少等待此时间才能再次回复。适合多任务场景，避免短时间连续回复。</p>
            </div>
        </div>

        <button class="bg-sky-500 hover:bg-sky-600 dark:hover:bg-sky-400 transition-colors rounded-lg px-3 py-1 text-gray-100" @click="saveSettings">保存设置</button>
    </div>

    <div class="px-3 py-2">
        <h4 class="text-xl">任务列表 ({{ tasksList.length }}/{{ limit }})</h4>
        <p v-show="tasksList.length >= limit" class="text-sm">注：任务数已达到或超出上限</p>

        <div class="my-5 grid grid-cols-6 gap-2 max-w-[48em]">
            <Modal class="col-span-6 sm:col-span-3 lg:col-span-1" :title="editMode ? '编辑自动回帖任务' : '添加自动回帖任务'" v-show="tasksList.length < limit || editMode">
                <template #default>
                    <button class="w-full rounded-2xl border-2 border-gray-300 hover:bg-gray-300 px-4 py-1 hover:text-black transition-colors">{{ editMode ? '编辑任务' : '添加任务' }}</button>
                </template>
                <template #container>
                    <!-- 分区 1：选择发帖账号 -->
                    <div class="border-b border-gray-400 dark:border-gray-600 pb-3 mb-3">
                        <h6 class="font-bold mb-2">1. 选择发帖账号</h6>
                        <div class="my-3">
                            <label class="flex items-center gap-1">发帖账号 <button @click="showHelp('pid')" class="inline-block text-xs bg-sky-500 text-white rounded px-1">?</button></label>
                            <select v-model.number="taskToAdd.pid" class="bg-gray-200 dark:bg-gray-900 dark:text-gray-100 form-select block w-full mt-1 rounded-xl">
                                <option :value="0">请选择</option>
                                <option v-for="(name, pid) in pidNameKV" :key="pid" :value="Number(pid)">{{ name }}</option>
                            </select>
                        </div>
                    </div>

                    <!-- 分区 2：目标帖子 -->
                    <div class="border-b border-gray-400 dark:border-gray-600 pb-3 mb-3">
                        <h6 class="font-bold mb-2">2. 目标帖子</h6>
                        <div class="my-3">
                            <label class="flex items-center gap-1">贴吧名称 <button @click="showHelp('fname')" class="inline-block text-xs bg-sky-500 text-white rounded px-1">?</button></label>
                            <input type="text" v-model="taskToAdd.fname" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-input" placeholder="贴吧名（不带末尾吧字）" />
                        </div>
                        <div class="my-3">
                            <label class="flex items-center gap-1">帖子ID <button @click="showHelp('tid')" class="inline-block text-xs bg-sky-500 text-white rounded px-1">?</button></label>
                            <input type="number" v-model="taskToAdd.tid" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-input" placeholder="帖子 tid" />
                        </div>
                    </div>

                    <!-- 分区 3：回复内容 -->
                    <div class="border-b border-gray-400 dark:border-gray-600 pb-3 mb-3">
                        <h6 class="font-bold mb-2">3. 回复内容</h6>
                        <div class="my-3">
                            <label class="flex items-center gap-1">回复内容（每行一条，随机发送） <button @click="showHelp('reply_content_list')" class="inline-block text-xs bg-sky-500 text-white rounded px-1">?</button></label>
                            <textarea v-model="taskToAdd.reply_content_list" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-textarea" rows="4" placeholder="每行一条回复内容，系统会随机选择一条发送"></textarea>
                        </div>
                    </div>

                    <!-- 分区 4：自动执行规则 -->
                    <div class="pb-3 mb-3">
                        <h6 class="font-bold mb-2">4. 自动执行规则</h6>
                        <div class="my-3 flex gap-3">
                            <div class="grow">
                                <label class="flex items-center gap-1">最小间隔(秒) <button @click="showHelp('reply_interval_range')" class="inline-block text-xs bg-sky-500 text-white rounded px-1">?</button></label>
                                <input type="number" v-model="taskToAdd.reply_interval_min" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-input" min="0" />
                            </div>
                            <div class="grow">
                                <label class="flex items-center gap-1">最大间隔(秒) <span class="text-xs text-gray-500">(0=与最小相同)</span></label>
                                <input type="number" v-model="taskToAdd.reply_interval_max" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-input" min="0" />
                            </div>
                        </div>
                        <div class="my-3 flex gap-3">
                            <div class="grow">
                                <label class="flex items-center gap-1">回复概率（%） <button @click="showHelp('probability')" class="inline-block text-xs bg-sky-500 text-white rounded px-1">?</button></label>
                                <input type="number" v-model="taskToAdd.reply_probability" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-input" min="1" max="100" />
                            </div>
                            <div class="grow">
                                <label class="flex items-center gap-1">最大执行次数 <button @click="showHelp('max_count')" class="inline-block text-xs bg-sky-500 text-white rounded px-1">?</button></label>
                                <input type="number" v-model="taskToAdd.max_count" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-input" min="0" placeholder="0=无限" />
                            </div>
                        </div>
                        <div class="my-3">
                            <label class="flex items-center gap-1">触发模式 <button @click="showHelp('trigger')" class="inline-block text-xs bg-sky-500 text-white rounded px-1">?</button></label>
                            <select v-model="taskToAdd.trigger_mode" class="bg-gray-200 dark:bg-gray-900 dark:text-gray-100 form-select block w-full mt-1 rounded-xl">
                                <option value="new_floor">新楼层</option>
                                <option value="keyword">关键词匹配</option>
                            </select>
                        </div>
                        <div class="my-3" v-show="taskToAdd.trigger_mode === 'keyword'">
                            <label class="flex items-center gap-1">匹配关键词 <button @click="showHelp('keywords')" class="inline-block text-xs bg-sky-500 text-white rounded px-1">?</button></label>
                            <textarea v-model="taskToAdd.match_keywords" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-textarea" rows="3" placeholder="每行一个关键词"></textarea>
                        </div>
                        <div class="my-3" v-show="taskToAdd.trigger_mode === 'keyword'">
                            <label class="flex items-center gap-1">回复目标 <button @click="showHelp('reply_target')" class="inline-block text-xs bg-sky-500 text-white rounded px-1">?</button></label>
                            <select v-model="taskToAdd.reply_target" class="bg-gray-200 dark:bg-gray-900 dark:text-gray-100 form-select block w-full mt-1 rounded-xl">
                                <option value="floor">回复楼层</option>
                                <option value="subpost">楼中楼</option>
                            </select>
                        </div>
                        <div class="my-3 flex items-center gap-2">
                            <input type="checkbox" v-model="taskToAdd.allow_replied" id="allow-replied-add" class="form-checkbox" />
                            <label for="allow-replied-add" class="flex items-center gap-1">允许重复回复已回复过的楼层 <button @click="showHelp('allow_replied')" class="inline-block text-xs bg-sky-500 text-white rounded px-1">?</button></label>
                        </div>
                        <div class="my-3 flex gap-3">
                            <div class="grow">
                                <label class="flex items-center gap-1">开始时间 <button @click="showHelp('active_time')" class="inline-block text-xs bg-sky-500 text-white rounded px-1">?</button></label>
                                <input type="time" v-model="taskToAdd.active_time_start" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-input" />
                            </div>
                            <div class="grow">
                                <label class="flex items-center gap-1">结束时间</label>
                                <input type="time" v-model="taskToAdd.active_time_end" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-input" />
                            </div>
                        </div>
                        <div class="my-3 flex items-center gap-2">
                            <input type="checkbox" v-model="taskToAdd.enabled" id="enabled-add" class="form-checkbox" />
                            <label for="enabled-add">是否启用</label>
                        </div>
                    </div>

                    <div class="flex gap-2">
                        <button v-if="editMode" class="px-3 py-1 rounded-lg my-2 bg-sky-500 hover:bg-sky-600 dark:hover:bg-sky-400 text-gray-100 transition-colors" @click="saveEditTask">保存修改</button>
                        <template v-else>
                            <button class="px-3 py-1 rounded-lg my-2 bg-sky-500 hover:bg-sky-600 dark:hover:bg-sky-400 text-gray-100 transition-colors" @click="addTask()">保存并启用</button>
                            <button class="px-3 py-1 rounded-lg my-2 bg-yellow-500 hover:bg-yellow-600 dark:hover:bg-yellow-400 text-gray-100 transition-colors" @click="addTask(0)">保存但暂停</button>
                        </template>
                        <button v-if="editMode" class="px-3 py-1 rounded-lg my-2 bg-gray-300 hover:bg-gray-400 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-900 dark:text-gray-100 transition-colors" @click="resetForm">取消编辑</button>
                    </div>
                </template>
            </Modal>

            <Modal class="col-span-6 sm:col-span-3 lg:col-span-1" title="确认清空全部任务？" v-show="tasksList.length > 0">
                <template #default>
                    <button class="w-full rounded-2xl border-2 border-pink-400 hover:bg-pink-400 hover:text-white px-4 py-1 transition-colors">清空全部</button>
                </template>
                <template #container>
                    <button class="bg-pink-500 hover:bg-pink-600 px-3 py-1 rounded-lg transition-colors text-gray-100 w-full text-lg" @click="emptyAllTasks">确认清空</button>
                </template>
            </Modal>
        </div>

        <div v-if="loadingList" class="w-full h-32 rounded-xl bg-gray-300 dark:bg-gray-700 animate-pulse"></div>
        <template v-else>
            <div class="border-4 border-gray-400 dark:border-gray-700 rounded-xl p-5 my-3" v-for="task in tasksList" :key="task.id">
                <ul class="marker:text-sky-500 list-disc list-inside">
                    <li>
                        <span class="font-bold">{{ task.fname }}吧 </span><NuxtLink class="font-mono hover:underline underline-offset-1" :to="'https://tieba.baidu.com/p/' + task.tid" target="blank">帖子{{ task.tid }}</NuxtLink>
                    </li>
                    <li>
                        <span class="font-bold">回复内容预览 : </span
                        ><span class="text-sm"
                            >{{ (task.reply_content_list || task.reply_content || '').split('\n')[0]
                            }}{{ (task.reply_content_list || '').split('\n').filter((l: string) => l.trim()).length > 1 ? '（随机' + (task.reply_content_list || '').split('\n').filter((l: string) => l.trim()).length + '条）' : '' }}</span
                        >
                    </li>
                    <li>
                        <span class="font-bold">间隔范围 : </span
                        ><span class="font-mono">{{ task.reply_interval_min === task.reply_interval_max || !task.reply_interval_max ? task.reply_interval_min + '秒' : task.reply_interval_min + '-' + task.reply_interval_max + '秒' }}</span>
                        <span class="ml-3 font-bold">概率 : </span><span class="font-mono">{{ task.reply_probability }}%</span>
                    </li>
                    <li>
                        <span class="font-bold">执行次数 : </span><span class="font-mono">{{ task.success_count || 0 }}/{{ task.max_count > 0 ? task.max_count : '无限' }}</span>
                    </li>
                    <li>
                        <span class="font-bold">活跃时间 : </span><span class="font-mono">{{ task.active_time_start && task.active_time_end ? task.active_time_start + '-' + task.active_time_end : '全天' }}</span>
                    </li>
                    <li>
                        <span class="font-bold">状态 : </span>
                        <span :class="taskStatusDisplay(task).class">{{ taskStatusDisplay(task).text }}</span>
                        <span v-if="task.last_check_time > 0" class="ml-2 text-sm text-gray-500 dark:text-gray-400">· {{ getPubDate(new Date(task.last_check_time * 1000)) }}</span>
                    </li>
                    <li v-if="task.last_error">
                        <span class="font-bold">跳过原因 : </span>
                        <span class="text-sm text-gray-500 dark:text-gray-400 truncate block" :title="task.last_error">{{ task.last_error }}</span>
                    </li>
                </ul>

                <hr class="border-gray-400 dark:border-gray-600 my-3" />

                <button
                    :class="task.enabled === 1 ? 'bg-green-500 hover:bg-green-600 dark:hover:bg-green-400' : 'bg-gray-400 hover:bg-gray-500 dark:bg-gray-600 dark:hover:bg-gray-500'"
                    class="rounded-lg px-3 py-1 text-gray-100 transition-colors mr-1"
                    @click="toggleTaskEnabled(task)"
                >
                    {{ task.enabled === 1 ? '已启用' : '已暂停' }}
                </button>
                <button class="bg-sky-500 hover:bg-sky-600 dark:hover:bg-sky-400 rounded-lg px-3 py-1 text-gray-100 transition-colors mr-1" @click="editTask(task)">编辑</button>
                <Modal class="mr-1 inline-block" :title="'确认删除任务 #' + task.id + ' ？'">
                    <template #default>
                        <button class="bg-pink-500 hover:bg-pink-600 dark:hover:bg-pink-400 rounded-lg px-3 py-1 text-gray-100 transition-colors">删除</button>
                    </template>
                    <template #container>
                        <button class="bg-pink-500 hover:bg-pink-600 px-3 py-1 rounded-lg transition-colors text-gray-100 w-full text-lg" @click="deleteTask(task.id)">确认删除</button>
                    </template>
                </Modal>
                <Modal class="mx-1 inline-block" :title="task.fname + ' 吧帖子 ' + task.tid + ' 执行日志'">
                    <template #default>
                        <button class="rounded-lg bg-gray-300 hover:bg-gray-400 dark:bg-gray-700 dark:hover:bg-gray-600 px-3 py-1 text-gray-900 dark:text-gray-100 transition-colors" title="日志">日志</button>
                    </template>
                    <template #container>
                        <div class="rounded-lg bg-gray-300 dark:bg-gray-800 px-5 py-3 mb-3" v-for="(log_, i) in parseLogs(task.log)" :key="task.id + '_' + i">
                            <span class="text-sm">{{ log_.text }}</span>
                        </div>
                        <div v-if="parseLogs(task.log).length === 0" class="text-gray-500 text-center py-3">暂无日志</div>
                    </template>
                </Modal>
            </div>
            <div v-if="tasksList.length === 0" class="text-gray-500 text-center py-8">暂无任务</div>
        </template>
    </div>

    <div class="px-3 py-2">
        <h3 class="text-2xl mb-4">测试回帖</h3>
        <div class="my-3 max-w-[48em]">
            <div class="my-3">
                <label>百度账号</label>
                <select v-model.number="testForm.pid" class="bg-gray-200 dark:bg-gray-900 dark:text-gray-100 form-select block w-full mt-1 rounded-xl">
                    <option :value="0">请选择</option>
                    <option v-for="(name, pid) in pidNameKV" :key="pid" :value="Number(pid)">{{ name }}</option>
                </select>
            </div>
            <div class="my-3">
                <label>贴吧名称</label>
                <input type="text" v-model="testForm.fname" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-input" placeholder="贴吧名" />
            </div>
            <div class="my-3">
                <label>帖子ID</label>
                <input type="number" v-model="testForm.tid" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-input" placeholder="帖子 tid" />
            </div>
            <div class="my-3">
                <label>回复内容</label>
                <textarea v-model="testForm.reply_content" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-textarea" rows="3" placeholder="支持变量：{floor} {time} {date} {tid} {username}"></textarea>
            </div>
            <div class="my-3">
                <label>触发模式</label>
                <select v-model="testForm.trigger_mode" class="bg-gray-200 dark:bg-gray-900 dark:text-gray-100 form-select block w-full mt-1 rounded-xl">
                    <option value="new_floor">新楼层</option>
                    <option value="keyword">关键词匹配</option>
                </select>
            </div>
            <div class="my-3" v-show="testForm.trigger_mode === 'keyword'">
                <label>匹配关键词（一行一个）</label>
                <textarea v-model="testForm.match_keywords" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-textarea" rows="3" placeholder="每行一个关键词"></textarea>
            </div>
            <div class="my-3">
                <label>回复目标</label>
                <select v-model="testForm.reply_target" class="bg-gray-200 dark:bg-gray-900 dark:text-gray-100 form-select block w-full mt-1 rounded-xl">
                    <option value="floor">回复楼层</option>
                    <option value="subpost">楼中楼</option>
                </select>
            </div>
            <div class="my-3 flex items-center gap-2">
                <input type="checkbox" v-model="testForm.allow_replied" id="allow-replied-test" class="form-checkbox" />
                <label for="allow-replied-test">允许重复回复</label>
            </div>
            <button class="px-3 py-1 rounded-lg my-2 bg-sky-500 hover:bg-sky-600 dark:hover:bg-sky-400 text-gray-100 transition-colors" @click="runTest">测试</button>
            <div v-if="testResult" class="my-3 p-3 rounded-xl bg-gray-100 dark:bg-gray-800">
                <span class="font-bold">结果：</span><span>{{ testResult }}</span>
            </div>
        </div>
    </div>

    <!-- 帮助弹窗 -->
    <Modal
        title="帮助"
        v-show="true"
        :active="helpModalVisible"
        @active-callback="
            (v: boolean) => {
                if (!v) helpModalVisible = false
            }
        "
    >
        <template #default></template>
        <template #container>
            <div class="whitespace-pre-line text-sm">{{ helpTexts[helpModalKey] || '暂无帮助信息。' }}</div>
        </template>
    </Modal>

    <SyncModule :loading="loading" :callback="getTasksList" />
</template>

<style scoped></style>
