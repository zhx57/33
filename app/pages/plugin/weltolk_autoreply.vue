<script setup lang="ts">
import { Notice, Request } from '~/share/Tools'
import { getPubDate } from '~/share/Time'

const store = useMainStore()
const pidNameKV = computed(() => store.pidNameKV)
const loading = computed(() => store.loading)

const tasksSwitch = ref<boolean>(false)
const limit = ref<number>(0)
const loadingList = ref<boolean>(false)

const settings = ref<{ global_limit: number; personal_limit: number }>({ global_limit: 5, personal_limit: 0 })

const editMode = ref<boolean>(false)
const editingTaskId = ref<number>(0)

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
        reply_interval: number
        reply_probability: number
        enabled: number
        retry_count: number
        trigger_mode: string
        reply_target: string
        allow_replied: number
        match_keywords: string
    }[]
>([])

const defaultTaskForm = () => ({
    pid: 0 as number,
    fname: '',
    tid: '',
    reply_content: '',
    reply_interval: 300,
    reply_probability: 100,
    trigger_mode: 'new_floor',
    reply_target: 'floor',
    allow_replied: false,
    match_keywords: '',
    enabled: true
})

const taskForm = ref(defaultTaskForm())

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

const helpKey = ref<string>('')
const helpModalActive = ref<boolean>(false)

const deleteTarget = ref<{ id: number; fname: string; tid: number; content: string }>({ id: 0, fname: '', tid: 0, content: '' })
const deleteModalActive = ref<boolean>(false)

const logModalActive = ref<boolean>(false)
const logModalTask = ref<typeof tasksList.value[number] | null>(null)

const helpTexts: Record<string, string> = {
    interval: '回帖间隔\n\n两次自动回帖之间最少等待的秒数。\n\n示例：设为 300 表示回帖后至少等 5 分钟才可能再次回帖。\n\n建议：不低于 60 秒，避免触发贴吧频率限制。',
    probability: '回帖概率\n\n检测到新回复时实际执行回帖的概率。\n\n示例：设为 50 表示每次检测到新楼层，有一半的机率会回帖，一半跳过。\n\n建议：设 100 则每次都回，设低值可降低封号风险。',
    trigger: '触发模式\n\n新楼层即回复：只要帖子有新楼层就触发回帖（需满足间隔和概率条件）。\n\n关键词匹配回复：帖子有新楼层时，先检查楼层内容是否包含你设置的关键词，匹配成功才回帖，且会自动在回帖内容前加上 @楼层用户名 前缀。',
    keywords: '匹配关键词\n\n仅在触发模式为"关键词匹配回复"时生效。\n\n格式：一行一个关键词。\n\n示例：\n求助\n怎么解决\n报错\n\n当最新楼层内容包含"求助"、"怎么解决"或"报错"任一词语时触发回帖。\n\n提示：不区分大小写，回复内容可用 {username} 变量引用楼层用户名。',
    reply_target: '回复目标\n\n主楼层回复：直接回复帖子的楼层\n\n楼中楼回复：回复楼层中的楼中楼评论（需要关键词模式）',
    allow_replied: '开启后，每次执行都会重新扫描所有楼层（包括已回复过的），不推进水位。默认关闭，只回复新楼层。',
    content: '回帖内容变量\n\n{floor} — 当前楼层数（回复数）\n{time} — 当前时间，如 18:30:00\n{date} — 当前日期，如 2026-06-16\n{tid} — 帖子ID\n{username} — 楼层用户名（仅关键词模式）\n\n示例：第{floor}楼打卡！{date} → 第100楼打卡！2026-06-16',
    fname: '贴吧名称\n\n目标贴吧的名称，不是网址。\n\n正确：天堂鸡汤\n\n错误：https://tieba.baidu.com/f?kw=天堂鸡汤',
    tid: '帖子 ID\n\n帖子的纯数字 ID，从帖子网址中获取。\n\n示例：帖子链接为 https://tieba.baidu.com/p/12345678，则 ID 为 12345678',
    pid: '发帖账号\n\n选择用于自动回帖的百度贴吧账号。必须已在云签到中绑定。\n\n提示：如无可用账号，请先前往贴吧管理绑定。'
}

const showHelp = (key: string) => {
    helpKey.value = key
    helpModalActive.value = true
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
        tasksSwitch.value = res.data
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
            tasksList.value = res.data?.list || []
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
    taskForm.value = defaultTaskForm()
    editMode.value = false
    editingTaskId.value = 0
}

const submitTask = (action: 'save' | 'save_pause' | 'test') => {
    if (action === 'test') {
        runTest()
        return
    }

    if (!Object.keys(pidNameKV.value).includes(taskForm.value.pid.toString())) {
        Notice('请选择百度账号', 'error')
        return
    }
    if (!taskForm.value.fname || !taskForm.value.tid || !taskForm.value.reply_content) {
        Notice('请填写所有必填字段（贴吧名称、帖子ID、回帖内容）', 'error')
        return
    }

    const enabled = action === 'save_pause' ? '0' : '1'
    const body = new URLSearchParams({
        pid: Number(taskForm.value.pid).toString(),
        fname: taskForm.value.fname,
        tid: taskForm.value.tid,
        reply_content: taskForm.value.reply_content,
        reply_interval: taskForm.value.reply_interval.toString(),
        reply_probability: taskForm.value.reply_probability.toString(),
        trigger_mode: taskForm.value.trigger_mode,
        reply_target: taskForm.value.reply_target,
        allow_replied: taskForm.value.allow_replied ? '1' : '0',
        match_keywords: taskForm.value.match_keywords,
        enabled
    }).toString()

    const url = editMode.value
        ? store.basePath + '/plugins/weltolk_autoreply/list/' + editingTaskId.value
        : store.basePath + '/plugins/weltolk_autoreply/list'
    const method = editMode.value ? 'PUT' : 'PATCH'

    Request(url, {
        headers: {
            Authorization: store.authorization,
            'Content-Type': 'application/x-www-form-urlencoded'
        },
        method,
        body
    }).then((res) => {
        if (res.code !== 200 && res.code !== 201 && res.code !== 204) {
            Notice(res.message, 'error')
            return
        }
        Notice(editMode.value ? '保存修改成功' : '添加成功', 'success')
        getTasksList()
        resetForm()
    })
}

const editTask = (task: typeof tasksList.value[number]) => {
    editMode.value = true
    editingTaskId.value = task.id
    taskForm.value = {
        pid: Number(task.pid),
        fname: task.fname,
        tid: String(task.tid),
        reply_content: task.reply_content,
        reply_interval: task.reply_interval,
        reply_probability: task.reply_probability,
        trigger_mode: task.trigger_mode || 'new_floor',
        reply_target: task.reply_target || 'floor',
        allow_replied: task.allow_replied === 1,
        match_keywords: task.match_keywords || '',
        enabled: task.enabled === 1
    }
    window.scrollTo({ top: 0, behavior: 'smooth' })
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
        deleteModalActive.value = false
    })
}

const confirmDelete = (task: typeof tasksList.value[number]) => {
    deleteTarget.value = {
        id: task.id,
        fname: task.fname,
        tid: task.tid,
        content: task.reply_content.length > 30 ? task.reply_content.substring(0, 30) + '...' : task.reply_content
    }
    deleteModalActive.value = true
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
            pid: Number(testForm.value.pid).toString(),
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
    const params: Record<string, string> = {
        personal_limit: String(settings.value.personal_limit || 0)
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
            settings.value.global_limit = res.data.global_limit || 5
            settings.value.personal_limit = res.data.personal_limit || 0
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

const getStatusInfo = (task: typeof tasksList.value[number]) => {
    if (task.enabled === 1) {
        if (task.last_status === 'error') {
            return { text: '异常', class: 'bg-yellow-500 text-white' }
        } else if (task.last_status === 'vcode') {
            return { text: '需验证码', class: 'bg-yellow-500 text-white' }
        } else {
            return { text: '运行中', class: 'bg-green-500 text-white' }
        }
    } else {
        if (task.last_status === 'error') {
            return { text: '失败已停用', class: 'bg-red-500 text-white' }
        } else {
            return { text: '已暂停', class: 'bg-gray-400 text-white dark:bg-gray-600' }
        }
    }
}

const contentPreview = (content: string, maxLen: number = 20) => {
    if (content.length <= maxLen) return content
    return content.substring(0, maxLen) + '...'
}

const formatCheckTime = (timestamp: number) => {
    if (!timestamp || timestamp === 0) return '-'
    const d = new Date(timestamp * 1000)
    const month = (d.getMonth() + 1).toString().padStart(2, '0')
    const day = d.getDate().toString().padStart(2, '0')
    const hour = d.getHours().toString().padStart(2, '0')
    const min = d.getMinutes().toString().padStart(2, '0')
    return `${month}-${day} ${hour}:${min}`
}

const errorPreview = (error: string, maxLen: number = 30) => {
    if (!error) return ''
    if (error.length <= maxLen) return error
    return error.substring(0, maxLen) + '...'
}

const showLogModal = (task: typeof tasksList.value[number]) => {
    logModalTask.value = task
    logModalActive.value = true
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
        <h2 class="text-2xl mb-4 font-bold">自动回帖</h2>

        <!-- 设置区域 -->
        <div class="mb-6">
            <button
                :class="{
                    'rounded-lg px-4 py-2 text-gray-100 transition-colors font-bold': true,
                    'bg-sky-500 hover:bg-sky-600 dark:hover:bg-sky-400': !tasksSwitch,
                    'bg-pink-500 hover:bg-pink-600 dark:hover:bg-pink-400': tasksSwitch
                }"
                @click="updateTasksSwitch"
            >
                {{ tasksSwitch ? '已开启自动回帖' : '已停止自动回帖' }}
            </button>

            <div class="my-5">
                <h4 class="my-2 text-xl font-bold">任务数量限额</h4>
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

            <button class="bg-sky-500 hover:bg-sky-600 dark:hover:bg-sky-400 transition-colors rounded-lg px-3 py-1 text-gray-100" @click="saveSettings">保存设置</button>
        </div>

        <!-- 任务表单 -->
        <div class="mb-6">
            <h3 class="text-xl mb-3 font-bold">{{ editMode ? '编辑任务 #' + editingTaskId : '添加任务' }}</h3>

            <!-- 分区 1：选择发帖账号 -->
            <div class="border border-gray-300 dark:border-gray-600 rounded-xl mb-3">
                <div class="bg-gray-100 dark:bg-gray-800 px-4 py-2 rounded-t-xl font-bold">1. 选择发帖账号</div>
                <div class="px-4 py-3">
                    <label class="flex items-center gap-1">
                        发帖账号
                        <button class="inline-flex items-center justify-center w-4 h-4 text-[10px] rounded-full bg-sky-500 text-white font-bold leading-none" @click="showHelp('pid')">?</button>
                    </label>
                    <select v-model.number="taskForm.pid" class="bg-gray-200 dark:bg-gray-900 dark:text-gray-100 form-select block w-full mt-1 rounded-xl">
                        <option :value="0">-- 请选择百度账号 --</option>
                        <option v-for="(name, pid) in pidNameKV" :key="pid" :value="Number(pid)">{{ name }}</option>
                    </select>
                </div>
            </div>

            <!-- 分区 2：目标帖子 -->
            <div class="border border-gray-300 dark:border-gray-600 rounded-xl mb-3">
                <div class="bg-gray-100 dark:bg-gray-800 px-4 py-2 rounded-t-xl font-bold">2. 目标帖子</div>
                <div class="px-4 py-3">
                    <div class="mb-3">
                        <label class="flex items-center gap-1">
                            贴吧名称
                            <button class="inline-flex items-center justify-center w-4 h-4 text-[10px] rounded-full bg-sky-500 text-white font-bold leading-none" @click="showHelp('fname')">?</button>
                        </label>
                        <input type="text" v-model="taskForm.fname" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-input" placeholder="例如：天堂鸡汤" />
                    </div>
                    <div>
                        <label class="flex items-center gap-1">
                            帖子 ID
                            <button class="inline-flex items-center justify-center w-4 h-4 text-[10px] rounded-full bg-sky-500 text-white font-bold leading-none" @click="showHelp('tid')">?</button>
                        </label>
                        <input type="number" v-model="taskForm.tid" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-input" placeholder="帖子链接中 p/ 后面的数字" />
                    </div>
                </div>
            </div>

            <!-- 分区 3：回复内容 -->
            <div class="border border-gray-300 dark:border-gray-600 rounded-xl mb-3">
                <div class="bg-gray-100 dark:bg-gray-800 px-4 py-2 rounded-t-xl font-bold">3. 回复内容</div>
                <div class="px-4 py-3">
                    <label class="flex items-center gap-1">
                        回帖内容
                        <button class="inline-flex items-center justify-center w-4 h-4 text-[10px] rounded-full bg-sky-500 text-white font-bold leading-none" @click="showHelp('content')">?</button>
                    </label>
                    <textarea v-model="taskForm.reply_content" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-textarea" rows="4" placeholder="支持变量：{floor} 当前楼层, {time} 当前时间, {date} 日期, {tid} 帖子ID, {username} 楼层用户名"></textarea>
                    <span class="text-xs text-gray-500 dark:text-gray-400 mt-1 block">
                        可用变量：<code class="bg-gray-200 dark:bg-gray-700 px-1 rounded">{floor}</code> 当前楼层（回复数）
                        <code class="bg-gray-200 dark:bg-gray-700 px-1 rounded">{time}</code> 当前时间
                        <code class="bg-gray-200 dark:bg-gray-700 px-1 rounded">{date}</code> 日期
                        <code class="bg-gray-200 dark:bg-gray-700 px-1 rounded">{tid}</code> 帖子ID
                        <code class="bg-gray-200 dark:bg-gray-700 px-1 rounded">{username}</code> 楼层用户名
                    </span>
                </div>
            </div>

            <!-- 分区 4：自动执行规则 -->
            <div class="border border-gray-300 dark:border-gray-600 rounded-xl mb-3">
                <div class="bg-gray-100 dark:bg-gray-800 px-4 py-2 rounded-t-xl font-bold">4. 自动执行规则</div>
                <div class="px-4 py-3">
                    <div class="flex gap-3">
                        <div class="grow">
                            <label class="flex items-center gap-1">
                                回帖间隔（秒）
                                <button class="inline-flex items-center justify-center w-4 h-4 text-[10px] rounded-full bg-sky-500 text-white font-bold leading-none" @click="showHelp('interval')">?</button>
                            </label>
                            <input type="number" v-model="taskForm.reply_interval" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-input" min="10" placeholder="默认 300" />
                        </div>
                        <div class="grow">
                            <label class="flex items-center gap-1">
                                回帖概率（%）
                                <button class="inline-flex items-center justify-center w-4 h-4 text-[10px] rounded-full bg-sky-500 text-white font-bold leading-none" @click="showHelp('probability')">?</button>
                            </label>
                            <input type="number" v-model="taskForm.reply_probability" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-input" min="1" max="100" placeholder="1-100，默认 100" />
                        </div>
                    </div>

                    <div class="mt-3">
                        <label class="flex items-center gap-1">
                            触发模式
                            <button class="inline-flex items-center justify-center w-4 h-4 text-[10px] rounded-full bg-sky-500 text-white font-bold leading-none" @click="showHelp('trigger')">?</button>
                        </label>
                        <div class="flex gap-4 mt-1">
                            <label class="inline-flex items-center gap-1 cursor-pointer">
                                <input type="radio" v-model="taskForm.trigger_mode" value="new_floor" class="form-radio" />
                                <span>新楼层即回复</span>
                            </label>
                            <label class="inline-flex items-center gap-1 cursor-pointer">
                                <input type="radio" v-model="taskForm.trigger_mode" value="keyword" class="form-radio" />
                                <span>关键词匹配回复</span>
                            </label>
                        </div>
                    </div>

                    <div v-show="taskForm.trigger_mode === 'keyword'" class="mt-3">
                        <label class="flex items-center gap-1">
                            匹配关键词
                            <button class="inline-flex items-center justify-center w-4 h-4 text-[10px] rounded-full bg-sky-500 text-white font-bold leading-none" @click="showHelp('keywords')">?</button>
                        </label>
                        <textarea v-model="taskForm.match_keywords" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-textarea" rows="3" placeholder="一行一个关键词，楼层内容包含任一即触发回复"></textarea>
                    </div>

                    <div v-show="taskForm.trigger_mode === 'keyword'" class="mt-3">
                        <label class="flex items-center gap-1">
                            回复目标
                            <button class="inline-flex items-center justify-center w-4 h-4 text-[10px] rounded-full bg-sky-500 text-white font-bold leading-none" @click="showHelp('reply_target')">?</button>
                        </label>
                        <div class="flex gap-4 mt-1">
                            <label class="inline-flex items-center gap-1 cursor-pointer">
                                <input type="radio" v-model="taskForm.reply_target" value="floor" class="form-radio" />
                                <span>主楼层回复</span>
                            </label>
                            <label class="inline-flex items-center gap-1 cursor-pointer">
                                <input type="radio" v-model="taskForm.reply_target" value="subpost" class="form-radio" />
                                <span>楼中楼回复</span>
                            </label>
                        </div>
                    </div>

                    <div class="mt-3 flex items-center gap-2">
                        <input type="checkbox" v-model="taskForm.allow_replied" id="allow-replied-form" class="form-checkbox" />
                        <label for="allow-replied-form" class="flex items-center gap-1 cursor-pointer">
                            允许回复已回复过的楼层
                            <button class="inline-flex items-center justify-center w-4 h-4 text-[10px] rounded-full bg-sky-500 text-white font-bold leading-none" @click="showHelp('allow_replied')">?</button>
                        </label>
                    </div>

                    <div class="mt-3 flex items-center gap-2">
                        <input type="checkbox" v-model="taskForm.enabled" id="enabled-form" class="form-checkbox" />
                        <label for="enabled-form" class="cursor-pointer">是否启用</label>
                    </div>
                </div>
            </div>

            <!-- 操作按钮 -->
            <div class="flex flex-wrap gap-2">
                <button class="px-3 py-1.5 rounded-lg bg-sky-400 hover:bg-sky-500 dark:hover:bg-sky-300 text-white transition-colors font-bold" @click="submitTask('test')">测试发一条</button>
                <button class="px-3 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-700 dark:hover:bg-sky-500 text-white transition-colors font-bold" @click="submitTask('save')">{{ editMode ? '保存修改' : '保存并启用' }}</button>
                <button v-if="!editMode" class="px-3 py-1.5 rounded-lg bg-yellow-500 hover:bg-yellow-600 dark:hover:bg-yellow-400 text-white transition-colors font-bold" @click="submitTask('save_pause')">保存但暂停</button>
                <button v-if="editMode" class="px-3 py-1.5 rounded-lg bg-gray-400 hover:bg-gray-500 dark:bg-gray-600 dark:hover:bg-gray-500 text-white transition-colors font-bold" @click="resetForm">取消编辑</button>
            </div>
        </div>

        <!-- 任务列表 -->
        <div class="mb-6">
            <div class="flex items-center justify-between mb-3">
                <h3 class="text-xl font-bold">已添加的任务 ({{ tasksList.length }}/{{ limit }})</h3>
                <Modal class="inline-block" title="确认清空全部任务？" v-show="tasksList.length > 0">
                    <template #default>
                        <button class="rounded-lg border-2 border-pink-400 hover:bg-pink-400 hover:text-white px-3 py-1 transition-colors text-sm">清空全部</button>
                    </template>
                    <template #container>
                        <button class="bg-pink-500 hover:bg-pink-600 px-3 py-1 rounded-lg transition-colors text-gray-100 w-full text-lg" @click="emptyAllTasks">确认清空</button>
                    </template>
                </Modal>
            </div>
            <p v-show="tasksList.length >= limit && limit > 0" class="text-sm text-yellow-600 dark:text-yellow-400 mb-2">注：任务数已达到或超出上限</p>

            <div v-if="loadingList" class="w-full h-32 rounded-xl bg-gray-300 dark:bg-gray-700 animate-pulse"></div>
            <template v-else>
                <div v-if="tasksList.length > 0" class="overflow-x-auto">
                    <table class="w-full text-sm border-collapse">
                        <thead>
                            <tr class="bg-gray-100 dark:bg-gray-800">
                                <th class="border border-gray-300 dark:border-gray-600 px-3 py-2 text-left">贴吧</th>
                                <th class="border border-gray-300 dark:border-gray-600 px-3 py-2 text-left">帖子ID</th>
                                <th class="border border-gray-300 dark:border-gray-600 px-3 py-2 text-left">回复内容</th>
                                <th class="border border-gray-300 dark:border-gray-600 px-3 py-2 text-left">间隔</th>
                                <th class="border border-gray-300 dark:border-gray-600 px-3 py-2 text-left">概率</th>
                                <th class="border border-gray-300 dark:border-gray-600 px-3 py-2 text-left">状态</th>
                                <th class="border border-gray-300 dark:border-gray-600 px-3 py-2 text-left">上次检测</th>
                                <th class="border border-gray-300 dark:border-gray-600 px-3 py-2 text-left">上次错误</th>
                                <th class="border border-gray-300 dark:border-gray-600 px-3 py-2 text-left">操作</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="task in tasksList" :key="task.id" class="hover:bg-gray-50 dark:hover:bg-gray-900">
                                <td class="border border-gray-300 dark:border-gray-600 px-3 py-2">{{ task.fname }}</td>
                                <td class="border border-gray-300 dark:border-gray-600 px-3 py-2">
                                    <a :href="'https://tieba.baidu.com/p/' + task.tid" target="_blank" class="text-sky-500 hover:underline">{{ task.tid }}</a>
                                </td>
                                <td class="border border-gray-300 dark:border-gray-600 px-3 py-2" :title="task.reply_content">{{ contentPreview(task.reply_content) }}</td>
                                <td class="border border-gray-300 dark:border-gray-600 px-3 py-2">{{ task.reply_interval }}s</td>
                                <td class="border border-gray-300 dark:border-gray-600 px-3 py-2">{{ task.reply_probability }}%</td>
                                <td class="border border-gray-300 dark:border-gray-600 px-3 py-2">
                                    <span :class="getStatusInfo(task).class + ' px-2 py-0.5 rounded text-xs font-bold'">{{ getStatusInfo(task).text }}</span>
                                </td>
                                <td class="border border-gray-300 dark:border-gray-600 px-3 py-2">{{ formatCheckTime(task.last_check_time) }}</td>
                                <td class="border border-gray-300 dark:border-gray-600 px-3 py-2" :title="task.last_error || ''">{{ errorPreview(task.last_error) || '-' }}</td>
                                <td class="border border-gray-300 dark:border-gray-600 px-3 py-2">
                                    <div class="flex gap-1">
                                        <button class="px-2 py-1 rounded bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-xs transition-colors" @click="editTask(task)" title="编辑">编辑</button>
                                        <button class="px-2 py-1 rounded bg-sky-100 dark:bg-sky-900 hover:bg-sky-200 dark:hover:bg-sky-800 text-xs transition-colors text-sky-700 dark:text-sky-300" @click="showLogModal(task)" title="日志">日志</button>
                                        <button class="px-2 py-1 rounded bg-pink-100 dark:bg-pink-900 hover:bg-pink-200 dark:hover:bg-pink-800 text-xs transition-colors text-pink-700 dark:text-pink-300" @click="confirmDelete(task)" title="删除">删除</button>
                                    </div>
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
                <div v-else class="text-gray-500 text-center py-8">暂无任务</div>
            </template>
        </div>

        <!-- 测试回帖 -->
        <div class="mb-6">
            <h3 class="text-2xl mb-4 font-bold">测试回帖</h3>
            <div class="max-w-[48em]">
                <div class="my-3">
                    <label>百度账号</label>
                    <select v-model.number="testForm.pid" class="bg-gray-200 dark:bg-gray-900 dark:text-gray-100 form-select block w-full mt-1 rounded-xl">
                        <option :value="0">-- 请选择百度账号 --</option>
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
                    <div class="flex gap-4 mt-1">
                        <label class="inline-flex items-center gap-1 cursor-pointer">
                            <input type="radio" v-model="testForm.trigger_mode" value="new_floor" class="form-radio" />
                            <span>新楼层即回复</span>
                        </label>
                        <label class="inline-flex items-center gap-1 cursor-pointer">
                            <input type="radio" v-model="testForm.trigger_mode" value="keyword" class="form-radio" />
                            <span>关键词匹配回复</span>
                        </label>
                    </div>
                </div>
                <div class="my-3" v-show="testForm.trigger_mode === 'keyword'">
                    <label>匹配关键词（一行一个）</label>
                    <textarea v-model="testForm.match_keywords" class="bg-gray-200 dark:bg-gray-900 w-full rounded-xl mt-1 form-textarea" rows="3" placeholder="每行一个关键词"></textarea>
                </div>
                <div class="my-3" v-show="testForm.trigger_mode === 'keyword'">
                    <label>回复目标</label>
                    <div class="flex gap-4 mt-1">
                        <label class="inline-flex items-center gap-1 cursor-pointer">
                            <input type="radio" v-model="testForm.reply_target" value="floor" class="form-radio" />
                            <span>主楼层回复</span>
                        </label>
                        <label class="inline-flex items-center gap-1 cursor-pointer">
                            <input type="radio" v-model="testForm.reply_target" value="subpost" class="form-radio" />
                            <span>楼中楼回复</span>
                        </label>
                    </div>
                </div>
                <div class="my-3 flex items-center gap-2">
                    <input type="checkbox" v-model="testForm.allow_replied" id="allow-replied-test" class="form-checkbox" />
                    <label for="allow-replied-test" class="cursor-pointer">允许回复已回复过的楼层</label>
                </div>
                <button class="px-3 py-1 rounded-lg my-2 bg-sky-500 hover:bg-sky-600 dark:hover:bg-sky-400 text-gray-100 transition-colors" @click="runTest">测试</button>
                <div v-if="testResult" class="my-3 p-3 rounded-xl bg-gray-100 dark:bg-gray-800">
                    <span class="font-bold">结果：</span><span>{{ testResult }}</span>
                </div>
            </div>
        </div>
    </div>

    <!-- 帮助弹窗 -->
    <Modal :title="'帮助'" :active="helpModalActive" @active-callback="(v: boolean) => helpModalActive = v" nested-modal>
        <template #container>
            <div class="whitespace-pre-wrap text-sm leading-relaxed">{{ helpTexts[helpKey] || '暂无帮助信息。' }}</div>
        </template>
    </Modal>

    <!-- 日志弹窗 -->
    <Modal :title="'日志详情'" :active="logModalActive" @active-callback="(v: boolean) => logModalActive = v" nested-modal>
        <template #container>
            <div v-if="logModalTask">
                <div class="rounded-lg bg-gray-100 dark:bg-gray-800 px-4 py-2 mb-2" v-for="(log_, i) in parseLogs(logModalTask.log)" :key="i">
                    <span class="text-sm">{{ log_.text }}</span>
                </div>
                <div v-if="parseLogs(logModalTask.log).length === 0" class="text-gray-500 text-center py-3">暂无日志</div>
            </div>
        </template>
    </Modal>

    <!-- 删除确认弹窗 -->
    <Modal :title="'确认删除'" :active="deleteModalActive" @active-callback="(v: boolean) => deleteModalActive = v" nested-modal>
        <template #container>
            <p class="mb-3">确定要删除这个任务吗？</p>
            <table class="w-full text-sm border-collapse mb-3">
                <tr>
                    <td class="border border-gray-300 dark:border-gray-600 px-3 py-1.5 font-bold w-20">贴吧</td>
                    <td class="border border-gray-300 dark:border-gray-600 px-3 py-1.5">{{ deleteTarget.fname }}</td>
                </tr>
                <tr>
                    <td class="border border-gray-300 dark:border-gray-600 px-3 py-1.5 font-bold">帖子ID</td>
                    <td class="border border-gray-300 dark:border-gray-600 px-3 py-1.5">{{ deleteTarget.tid }}</td>
                </tr>
                <tr>
                    <td class="border border-gray-300 dark:border-gray-600 px-3 py-1.5 font-bold">回复内容</td>
                    <td class="border border-gray-300 dark:border-gray-600 px-3 py-1.5">{{ deleteTarget.content }}</td>
                </tr>
            </table>
            <p class="text-red-500 font-bold mb-3">此操作不可撤销。</p>
            <div class="flex gap-2">
                <button class="px-3 py-1.5 rounded-lg bg-gray-300 dark:bg-gray-600 hover:bg-gray-400 dark:hover:bg-gray-500 transition-colors" @click="deleteModalActive = false">取消</button>
                <button class="px-3 py-1.5 rounded-lg bg-pink-500 hover:bg-pink-600 text-white transition-colors" @click="deleteTask(deleteTarget.id)">确认删除</button>
            </div>
        </template>
    </Modal>

    <SyncModule :loading="loading" :callback="getTasksList" />
</template>

<style scoped></style>
