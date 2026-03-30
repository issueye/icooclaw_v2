<template>
    <ManagementPageLayout
        title="渠道管理"
        description="配置消息渠道，统一管理接入方式、凭证与回调信息。"
        :icon="WebhookIcon"
        content-class="overflow-y-auto pr-1"
    >
        <template #actions>
            <button
                @click="openAddChannel"
                class="btn btn-primary"
            >
                <PlusIcon :size="16" />
                添加渠道
            </button>
        </template>

        <template #metrics>
            <div class="metric-card bg-bg-secondary border border-border">
                <div class="flex items-center gap-3">
                    <div class="w-10 h-10 rounded-lg bg-accent/10 flex items-center justify-center">
                        <WebhookIcon :size="20" class="text-accent" />
                    </div>
                    <div>
                        <p class="text-2xl font-bold">{{ channels.length }}</p>
                        <p class="text-xs text-text-muted">渠道总数</p>
                    </div>
                </div>
            </div>
            <div class="metric-card bg-bg-secondary border border-border">
                <div class="flex items-center gap-3">
                    <div class="w-10 h-10 rounded-lg bg-green-500/10 flex items-center justify-center">
                        <CheckIcon :size="20" class="text-green-500" />
                    </div>
                    <div>
                        <p class="text-2xl font-bold">{{ enabledChannelCount }}</p>
                        <p class="text-xs text-text-muted">已启用渠道</p>
                    </div>
                </div>
            </div>
            <div class="metric-card bg-bg-secondary border border-border">
                <div class="flex items-center gap-3">
                    <div class="w-10 h-10 rounded-lg bg-sky-500/10 flex items-center justify-center">
                        <HashIcon :size="20" class="text-sky-500" />
                    </div>
                    <div>
                        <p class="text-2xl font-bold">{{ activeChannelTypeCount }}</p>
                        <p class="text-xs text-text-muted">渠道类型</p>
                    </div>
                </div>
            </div>
            <div class="metric-card bg-bg-secondary border border-border">
                <div class="flex items-center gap-3">
                    <div class="w-10 h-10 rounded-lg bg-amber-500/10 flex items-center justify-center">
                        <GlobeIcon :size="20" class="text-amber-500" />
                    </div>
                    <div>
                        <p class="text-sm font-semibold truncate max-w-[180px]">{{ topChannelTypeLabel }}</p>
                        <p class="text-xs text-text-muted">主要接入类型</p>
                    </div>
                </div>
            </div>
        </template>

        <template #filters>
            <div class="grid grid-cols-[minmax(0,1fr)_180px] gap-3 max-md:grid-cols-1">
                <div class="relative">
                    <SearchIcon :size="16" class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
                    <input
                        v-model="searchQuery"
                        type="text"
                        placeholder="搜索渠道名称、类型或标识..."
                        class="input-field w-full pl-10"
                    />
                </div>
                <AppSelect v-model="filterStatus" :options="statusOptions" />
            </div>
        </template>

        <div v-if="loadingChannels" class="flex items-center justify-center py-16 flex-1">
            <LoaderIcon :size="28" class="animate-spin text-accent" />
            <span class="ml-3 text-text-secondary">加载中...</span>
        </div>

        <!-- 渠道列表 -->
        <div v-else-if="filteredChannels.length > 0" class="space-y-3 overflow-y-auto pr-1 flex-1">
            <div
                v-for="ch in filteredChannels"
                :key="ch.id"
                class="bg-bg-secondary rounded-lg border border-border p-4 hover:border-accent/30 transition-all group"
            >
                <div class="flex items-start justify-between">
                    <div class="flex items-start gap-4">
                        <!-- 渠道图标 -->
                        <div :class="[
                                'w-11 h-11 rounded-lg flex items-center justify-center flex-shrink-0',
                            getChannelStyle(ch).bgClass
                        ]">
                            <component :is="getChannelIcon(ch)" :size="20"
                                :class="getChannelStyle(ch).iconClass" />
                        </div>

                        <div class="min-w-0">
                            <div class="flex items-center gap-2 flex-wrap">
                                <span class="font-semibold text-text-primary">{{ ch.name }}</span>
                                <span class="px-1.5 py-0.5 text-[10px] bg-bg-tertiary text-text-muted rounded font-medium uppercase">
                                    {{ getChannelTypeLabel(ch) }}
                                </span>
                                <span :class="[
                                    'text-[10px] px-1.5 py-0.5 rounded-full font-medium',
                                    ch.enabled
                                        ? 'bg-green-500/10 text-green-500'
                                        : 'bg-bg-tertiary text-text-muted'
                                ]">
                                    {{ ch.enabled ? '已启用' : '未启用' }}
                                </span>
                            </div>

                            <!-- 渠道详情 -->
                            <div class="mt-2 flex items-center gap-3 flex-wrap">
                                <span v-if="getChannelEndpoint(ch)" class="inline-flex items-center gap-1 text-xs text-text-muted">
                                    <GlobeIcon :size="10" />
                                    {{ getChannelEndpoint(ch) }}
                                </span>
                                <span v-if="getChannelId(ch)" class="inline-flex items-center gap-1 text-xs text-text-muted">
                                    <HashIcon :size="10" />
                                    {{ getChannelId(ch) }}
                                </span>
                            </div>
                        </div>
                    </div>

                    <!-- 操作 -->
                    <div class="flex items-center gap-2">
                        <!-- Toggle -->
                        <button
                            @click="toggleChannelEnabled(ch)"
                            :class="[
                                'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
                                ch.enabled ? 'bg-green-500' : 'bg-bg-tertiary'
                            ]"
                            :title="ch.enabled ? '点击禁用' : '点击启用'"
                        >
                            <span
                                :class="[
                                    'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
                                    ch.enabled ? 'translate-x-4' : 'translate-x-1'
                                ]"
                            />
                        </button>

                        <div class="flex items-center gap-1 opacity-60 group-hover:opacity-100 transition-opacity">
                            <button
                                @click="openEditChannel(ch)"
                                class="btn btn-ghost btn-icon text-text-secondary hover:text-accent"
                                title="编辑"
                            >
                                <EditIcon :size="15" />
                            </button>
                            <button
                                @click="handleDeleteChannel(ch)"
                                class="btn btn-ghost btn-icon text-text-secondary hover:text-red-500 hover:bg-red-500/10"
                                title="删除"
                            >
                                <TrashIcon :size="15" />
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- 空状态 -->
        <div v-else class="bg-bg-secondary rounded-lg border border-border p-10 text-center flex-1 flex flex-col items-center justify-center">
            <div class="w-14 h-14 mx-auto mb-4 rounded-2xl bg-bg-tertiary flex items-center justify-center">
                <WebhookIcon :size="26" class="text-text-muted" />
            </div>
            <h3 class="text-sm font-medium text-text-primary mb-1">{{ searchQuery || filterStatus ? "没有找到匹配的渠道" : "暂无渠道配置" }}</h3>
            <p class="text-xs text-text-secondary mb-4">{{ searchQuery || filterStatus ? "试试其他筛选条件" : "添加消息渠道来接收和发送消息" }}</p>
            <button
                @click="openAddChannel"
                class="btn btn-primary"
            >
                <PlusIcon :size="14" />
                添加第一个渠道
            </button>
        </div>

        <!-- 渠道编辑弹窗 -->
        <ModalDialog
            v-model:visible="channelDialogVisible"
            :title="editingChannel ? '编辑渠道' : '添加渠道'"
            size="lg"
            :scrollable="true"
            :loading="savingChannel"
            :confirm-disabled="!channelForm.name || channelErrors.length > 0"
            confirm-text="保存"
            loading-text="保存中..."
            @confirm="handleSaveChannel"
        >
            <div class="space-y-5">
                <!-- 基本信息 -->
                <div class="bg-bg-tertiary rounded-xl p-4 space-y-4">
                    <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">基本信息</h4>

                    <div>
                        <label class="block text-sm text-text-secondary mb-2">渠道类型</label>
                        <div class="grid grid-cols-5 gap-2">
                            <button
                                v-for="ct in channelTypes"
                                :key="ct.value"
                                @click="!editingChannel && (channelForm.type = ct.value)"
                                :class="[
                                    'p-2.5 rounded-lg border transition-all flex flex-col items-center gap-1.5',
                                    channelForm.type === ct.value
                                        ? 'border-accent bg-accent/10 text-accent'
                                        : 'border-border bg-bg-secondary hover:border-accent/50 text-text-secondary hover:text-text-primary',
                                    !!editingChannel ? 'opacity-50 cursor-not-allowed' : ''
                                ]"
                                :disabled="!!editingChannel"
                            >
                                <component :is="ct.icon" :size="18" />
                                <span class="text-[11px] font-medium">{{ ct.label }}</span>
                            </button>
                        </div>
                        <p v-if="editingChannel" class="text-[11px] text-text-muted mt-2 flex items-center gap-1">
                            <LockIcon :size="10" />
                            编辑模式下渠道类型不可修改
                        </p>
                    </div>

                    <div>
                        <label class="block text-sm text-text-secondary mb-2">渠道名称</label>
                        <input
                            v-model="channelForm.name"
                            type="text"
                            placeholder="如: 飞书客服、测试机器人"
                            class="w-full px-4 py-2.5 bg-bg-secondary border border-border rounded-lg focus:outline-none focus:border-accent/60 transition-colors"
                        />
                    </div>

                    <div class="flex items-center gap-3">
                        <input
                            v-model="channelForm.enabled"
                            type="checkbox"
                            id="channel-enabled"
                            class="w-4 h-4 rounded border-border bg-bg-secondary text-accent focus:ring-accent"
                        />
                        <label for="channel-enabled" class="text-sm text-text-secondary">
                            启用此渠道
                        </label>
                    </div>
                </div>

                <!-- 飞书配置 -->
                <template v-if="channelForm.type === '飞书'">
                    <div class="bg-bg-tertiary rounded-xl p-4 space-y-4">
                        <div class="flex items-center gap-2 mb-1">
                            <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">飞书配置</h4>
                        </div>

                        <div class="bg-blue-500/10 border border-blue-500/20 rounded-lg p-3">
                            <div class="text-xs text-blue-400 font-medium mb-2">📋 配置步骤</div>
                            <ol class="text-[11px] text-text-secondary space-y-1 list-decimal list-inside">
                                <li>前往 <a href="https://open.feishu.cn/app" target="_blank" class="text-accent hover:underline">飞书开放平台</a> 创建企业自建应用</li>
                                <li>在「凭证与基础信息」获取 App ID 和 App Secret</li>
                                <li>在「事件订阅」配置请求网址，并获取 Verification Token</li>
                                <li>在「权限管理」开通所需权限</li>
                            </ol>
                        </div>

                        <div class="grid grid-cols-2 gap-3">
                            <div>
                                <label class="block text-xs text-text-secondary mb-1">监听端口 <span class="text-red-400">*</span></label>
                                <input v-model.number="channelForm.config.port" type="number" placeholder="8082"
                                    class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                            </div>
                            <div>
                                <label class="block text-xs text-text-secondary mb-1">Webhook 路径</label>
                                <input v-model="channelForm.config.path" type="text" placeholder="/feishu/webhook"
                                    class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                            </div>
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">Webhook 回调地址</label>
                            <div class="flex items-center gap-2">
                                <code class="flex-1 bg-bg-secondary px-3 py-2 rounded-lg text-xs text-text-primary break-all">
                                    {{ getWebhookUrl() }}
                                </code>
                                <button @click="copyWebhookUrl" type="button"
                                    class="px-3 py-2 bg-bg-secondary border border-border rounded-lg text-xs hover:bg-bg-hover transition-colors flex items-center gap-1">
                                    <component :is="webhookUrlCopied ? CheckIcon : CopyIcon" :size="12" />
                                    {{ webhookUrlCopied ? '已复制' : '复制' }}
                                </button>
                            </div>
                            <p class="text-[11px] text-text-muted mt-1">将此地址配置到飞书事件订阅</p>
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">App ID <span class="text-red-400">*</span></label>
                            <input v-model="channelForm.config.app_id" type="text" placeholder="cli_xxxxxxxxxxxx"
                                class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">App Secret <span class="text-red-400">*</span></label>
                            <div class="relative">
                                <input v-model="channelForm.config.app_secret" :type="showAppSecret ? 'text' : 'password'"
                                    placeholder="应用密钥"
                                    class="w-full px-3 py-2 pr-9 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                                <button @click="showAppSecret = !showAppSecret" type="button"
                                    class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary transition-colors">
                                    <EyeIcon v-if="!showAppSecret" :size="14" />
                                    <EyeOffIcon v-else :size="14" />
                                </button>
                            </div>
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">Verification Token <span class="text-red-400">*</span></label>
                            <input v-model="channelForm.config.verification_token" type="text"
                                placeholder="事件订阅验证 Token"
                                class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">
                                Encrypt Key <span class="text-text-muted">(可选)</span>
                            </label>
                            <div class="relative">
                                <input v-model="channelForm.config.encrypt_key" :type="showEncryptKey ? 'text' : 'password'"
                                    placeholder="消息加密密钥"
                                    class="w-full px-3 py-2 pr-9 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                                <button @click="showEncryptKey = !showEncryptKey" type="button"
                                    class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary transition-colors">
                                    <EyeIcon v-if="!showEncryptKey" :size="14" />
                                    <EyeOffIcon v-else :size="14" />
                                </button>
                            </div>
                            <p class="text-[11px] text-text-muted mt-1">开启消息加密后需要配置，不加密可留空</p>
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">欢迎消息 <span class="text-text-muted">(可选)</span></label>
                            <textarea v-model="channelForm.config.welcome_message" rows="2"
                                placeholder="机器人被添加到群聊时发送的欢迎消息"
                                class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors resize-none" />
                        </div>

                        <div class="space-y-2">
                            <label class="flex items-center gap-3 cursor-pointer">
                                <input v-model="channelForm.config.enable_group_events" type="checkbox"
                                    class="w-4 h-4 rounded border-border bg-bg-secondary text-accent focus:ring-accent" />
                                <div>
                                    <span class="text-sm">处理群聊事件</span>
                                    <p class="text-[11px] text-text-muted">成员加入/退出、群解散等事件</p>
                                </div>
                            </label>
                            <label class="flex items-center gap-3 cursor-pointer">
                                <input v-model="channelForm.config.enable_card_message" type="checkbox"
                                    class="w-4 h-4 rounded border-border bg-bg-secondary text-accent focus:ring-accent" />
                                <div>
                                    <span class="text-sm">启用卡片消息</span>
                                    <p class="text-[11px] text-text-muted">支持发送交互式卡片消息</p>
                                </div>
                            </label>
                        </div>

                        <div class="bg-bg-secondary rounded-lg p-3">
                            <div class="text-xs text-text-muted font-medium mb-2">所需权限</div>
                            <div class="space-y-1">
                                <div class="flex items-center gap-2 text-[11px]">
                                    <span class="w-1.5 h-1.5 rounded-full bg-green-500"></span>
                                    <code class="text-text-secondary">im:message</code> — 接收消息
                                </div>
                                <div class="flex items-center gap-2 text-[11px]">
                                    <span class="w-1.5 h-1.5 rounded-full bg-green-500"></span>
                                    <code class="text-text-secondary">im:message:send_as_bot</code> — 以应用身份发消息
                                </div>
                                <div class="flex items-center gap-2 text-[11px]">
                                    <span class="w-1.5 h-1.5 rounded-full bg-yellow-500"></span>
                                    <code class="text-text-secondary">contact:user.base:readonly</code> — 获取用户信息（可选）
                                </div>
                                <div class="flex items-center gap-2 text-[11px]">
                                    <span class="w-1.5 h-1.5 rounded-full bg-yellow-500"></span>
                                    <code class="text-text-secondary">im:chat:readonly</code> — 获取群聊信息（可选）
                                </div>
                            </div>
                        </div>
                    </div>
                </template>

                <!-- 钉钉配置 -->
                <template v-else-if="channelForm.type === 'dingtalk'">
                    <div class="bg-bg-tertiary rounded-xl p-4 space-y-4">
                        <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">钉钉配置</h4>

                        <div class="bg-blue-500/10 border border-blue-500/20 rounded-lg p-3">
                            <div class="text-xs text-blue-400 font-medium mb-2">📋 配置步骤</div>
                            <ol class="text-[11px] text-text-secondary space-y-1 list-decimal list-inside">
                                <li>前往 <a href="https://open.dingtalk.com" target="_blank" class="text-accent hover:underline">钉钉开放平台</a> 创建企业内部应用</li>
                                <li>在「应用信息」获取 Client ID 和 Client Secret</li>
                                <li>在「开发管理」配置消息接收地址</li>
                                <li>在「权限管理」开通所需权限</li>
                            </ol>
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">Client ID <span class="text-red-400">*</span></label>
                            <input v-model="channelForm.config.client_id" type="text" placeholder="dingxxxxxxxxxxxxxxx"
                                class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">Client Secret <span class="text-red-400">*</span></label>
                            <div class="relative">
                                <input v-model="channelForm.config.client_secret" :type="showClientSecret ? 'text' : 'password'"
                                    placeholder="应用密钥"
                                    class="w-full px-3 py-2 pr-9 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                                <button @click="showClientSecret = !showClientSecret" type="button"
                                    class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary transition-colors">
                                    <EyeIcon v-if="!showClientSecret" :size="14" />
                                    <EyeOffIcon v-else :size="14" />
                                </button>
                            </div>
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">Agent ID <span class="text-text-muted">(可选)</span></label>
                            <input v-model.number="channelForm.config.agent_id" type="number" placeholder="应用 Agent ID"
                                class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                        </div>
                    </div>
                </template>

                <!-- Webhook 配置 -->
                <template v-else-if="channelForm.type === 'webhook'">
                    <div class="bg-bg-tertiary rounded-xl p-4 space-y-4">
                        <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">Webhook 配置</h4>

                        <div class="grid grid-cols-2 gap-3">
                            <div>
                                <label class="block text-xs text-text-secondary mb-1">监听端口</label>
                                <input v-model.number="channelForm.config.port" type="number" placeholder="8081"
                                    class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                            </div>
                            <div>
                                <label class="block text-xs text-text-secondary mb-1">Webhook 路径</label>
                                <input v-model="channelForm.config.path" type="text" placeholder="/webhook"
                                    class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                            </div>
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">目标 Webhook URL <span class="text-red-400">*</span></label>
                            <input v-model="channelForm.config.webhook_url" type="text" placeholder="https://example.com/webhook"
                                class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                            <p class="text-[11px] text-text-muted mt-1">接收消息的目标 Webhook 地址</p>
                        </div>
                    </div>
                </template>

                <!-- Telegram 配置 -->
                <template v-else-if="channelForm.type === 'telegram'">
                    <div class="bg-bg-tertiary rounded-xl p-4 space-y-4">
                        <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">Telegram 配置</h4>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">Bot Token <span class="text-red-400">*</span></label>
                            <div class="relative">
                                <input v-model="channelForm.config.bot_token" :type="showBotToken ? 'text' : 'password'"
                                    placeholder="123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
                                    class="w-full px-3 py-2 pr-9 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                                <button @click="showBotToken = !showBotToken" type="button"
                                    class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary transition-colors">
                                    <EyeIcon v-if="!showBotToken" :size="14" />
                                    <EyeOffIcon v-else :size="14" />
                                </button>
                            </div>
                            <p class="text-[11px] text-text-muted mt-1">从 @BotFather 获取，格式：数字:字母数字组合</p>
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">Webhook URL <span class="text-text-muted">(可选)</span></label>
                            <input v-model="channelForm.config.webhook_url" type="text"
                                placeholder="https://your-domain.com/api/telegram/webhook"
                                class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                            <p class="text-[11px] text-text-muted mt-1">接收 Telegram 消息的回调地址，需要公网可访问</p>
                        </div>

                        <div class="grid grid-cols-2 gap-3">
                            <div>
                                <label class="block text-xs text-text-secondary mb-1">监听端口</label>
                                <input v-model.number="channelForm.config.port" type="number" placeholder="8083"
                                    class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                            </div>
                            <div>
                                <label class="block text-xs text-text-secondary mb-1">Webhook 路径</label>
                                <input v-model="channelForm.config.path" type="text" placeholder="/telegram/webhook"
                                    class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                            </div>
                        </div>
                    </div>
                </template>

                <!-- QQ 配置 -->
                <template v-else-if="channelForm.type === 'qq'">
                    <div class="bg-bg-tertiary rounded-xl p-4 space-y-4">
                        <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">QQ 配置</h4>

                        <div class="bg-blue-500/10 border border-blue-500/20 rounded-lg p-3">
                            <div class="text-xs text-blue-400 font-medium mb-2">📋 配置步骤</div>
                            <ol class="text-[11px] text-text-secondary space-y-1 list-decimal list-inside">
                                <li>前往 <a href="https://q.qq.com/" target="_blank" class="text-accent hover:underline">QQ 开放平台</a> 创建 QQ 机器人</li>
                                <li>在「应用管理」获取 App ID 和 App Secret</li>
                                <li>配置机器人 Intent（消息权限）</li>
                            </ol>
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">App ID <span class="text-red-400">*</span></label>
                            <input v-model="channelForm.config.app_id" type="text" placeholder="1234567890"
                                class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">App Secret <span class="text-red-400">*</span></label>
                            <div class="relative">
                                <input v-model="channelForm.config.app_secret" :type="showQQSecret ? 'text' : 'password'"
                                    placeholder="应用密钥"
                                    class="w-full px-3 py-2 pr-9 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                                <button @click="showQQSecret = !showQQSecret" type="button"
                                    class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary transition-colors">
                                    <EyeIcon v-if="!showQQSecret" :size="14" />
                                    <EyeOffIcon v-else :size="14" />
                                </button>
                            </div>
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">允许的用户 <span class="text-text-muted">(可选)</span></label>
                            <input v-model="channelForm.config.allow_from" type="text" placeholder="多个用户用逗号分隔"
                                class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                            <p class="text-[11px] text-text-muted mt-1">留空则允许所有用户</p>
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">最大消息长度</label>
                            <input v-model.number="channelForm.config.max_message_length" type="number" placeholder="2000"
                                class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">群聊触发关键词 <span class="text-text-muted">(可选)</span></label>
                            <input v-model="channelForm.config.group_trigger" type="text" placeholder="@Bot"
                                class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                            <p class="text-[11px] text-text-muted mt-1">群聊中需要 @机器人 才能触发回复</p>
                        </div>

                        <label class="flex items-center gap-3 cursor-pointer">
                            <input v-model="channelForm.config.send_markdown" type="checkbox"
                                class="w-4 h-4 rounded border-border bg-bg-secondary text-accent focus:ring-accent" />
                            <div>
                                <span class="text-sm">发送 Markdown 消息</span>
                                <p class="text-[11px] text-text-muted">使用 Markdown 格式发送消息</p>
                            </div>
                        </label>
                    </div>
                </template>

                <!-- icoo_chat 配置 -->
                <template v-else-if="channelForm.type === 'icoo_chat'">
                    <div class="bg-bg-tertiary rounded-xl p-4 space-y-4">
                        <h4 class="text-xs font-semibold text-text-muted uppercase tracking-wider">icoo_chat 配置</h4>

                        <div class="bg-emerald-500/10 border border-emerald-500/20 rounded-lg p-3">
                            <div class="text-xs text-emerald-400 font-medium mb-2">🔌 接入说明</div>
                            <ol class="text-[11px] text-text-secondary space-y-1 list-decimal list-inside">
                                <li>先在 <code>icoo_chat_app_server</code> 创建 bot，拿到 <code>app_id</code> 和 <code>app_secret</code></li>
                                <li>将 Endpoint 指向 app server 的 <code>/ws/bot</code>，例如 <code>ws://127.0.0.1:18080/ws/bot</code></li>
                                <li>保存并启用后，<code>icoo_agent</code> 会作为 bot websocket 客户端接入</li>
                            </ol>
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">Bot WebSocket Endpoint <span class="text-red-400">*</span></label>
                            <input v-model="channelForm.config.endpoint" type="text" placeholder="ws://127.0.0.1:18080/ws/bot"
                                class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                        </div>

                        <div class="grid grid-cols-2 gap-3">
                            <div>
                                <label class="block text-xs text-text-secondary mb-1">App ID <span class="text-red-400">*</span></label>
                                <input v-model="channelForm.config.app_id" type="text" placeholder="app_xxxxx"
                                    class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                            </div>
                            <div>
                                <label class="block text-xs text-text-secondary mb-1">App Secret <span class="text-red-400">*</span></label>
                                <div class="relative">
                                    <input v-model="channelForm.config.app_secret" :type="showIcooChatSecret ? 'text' : 'password'"
                                        placeholder="bot app secret"
                                        class="w-full px-3 py-2 pr-9 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                                    <button @click="showIcooChatSecret = !showIcooChatSecret" type="button"
                                        class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary transition-colors">
                                        <EyeIcon v-if="!showIcooChatSecret" :size="14" />
                                        <EyeOffIcon v-else :size="14" />
                                    </button>
                                </div>
                            </div>
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-1">允许的用户 <span class="text-text-muted">(可选)</span></label>
                            <input v-model="channelForm.config.allow_from" type="text" placeholder="多个用户 ID 用逗号分隔"
                                class="w-full px-3 py-2 bg-bg-secondary border border-border rounded-lg text-sm focus:outline-none focus:border-accent/60 transition-colors" />
                            <p class="text-[11px] text-text-muted mt-1">留空则允许所有 app 用户消息进入该 bot 渠道</p>
                        </div>
                    </div>
                </template>

                <!-- 错误提示 -->
                <div v-if="channelErrors.length > 0" class="bg-red-500/10 border border-red-500/30 rounded-lg p-3">
                    <div class="text-sm text-red-400">
                        <div v-for="(error, index) in channelErrors" :key="index">• {{ error }}</div>
                    </div>
                </div>
            </div>
        </ModalDialog>
    </ManagementPageLayout>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from "vue";
import {
    Plus as PlusIcon,
    Edit as EditIcon,
    Trash as TrashIcon,
    Copy as CopyIcon,
    Check as CheckIcon,
    Webhook as WebhookIcon,
    Send as SendIcon,
    MessageSquare as MessageSquareIcon,
    Hash as HashIcon,
    Globe as GlobeIcon,
    Eye as EyeIcon,
    EyeOff as EyeOffIcon,
    Loader as LoaderIcon,
    Lock as LockIcon,
    MessageCircle as QQIcon,
    Bot as BotIcon,
    Search as SearchIcon,
} from "lucide-vue-next";
import api from "@/services/api";
import ModalDialog from "@/components/ModalDialog.vue";
import ManagementPageLayout from "@/components/layout/ManagementPageLayout.vue";
import AppSelect from "@/components/form/AppSelect.vue";
import { useToast } from "@/composables/useToast.js";
import { useConfirm } from "@/composables/useConfirm.js";

const { toast } = useToast();
const { confirm } = useConfirm();

const channelTypes = [
    { label: '飞书', value: '飞书', icon: SendIcon },
    { label: '钉钉', value: 'dingtalk', icon: SendIcon },
    { label: 'QQ', value: 'qq', icon: QQIcon },
    { label: 'icoo_chat', value: 'icoo_chat', icon: BotIcon },
    { label: 'Telegram', value: 'telegram', icon: SendIcon },
    { label: 'Webhook', value: 'webhook', icon: WebhookIcon },
];
const statusOptions = [
    { label: "全部状态", value: "" },
    { label: "已启用", value: "enabled" },
    { label: "未启用", value: "disabled" },
];

const channels = ref([]);
const loadingChannels = ref(false);
const searchQuery = ref("");
const filterStatus = ref("");
const showChannelDialog = ref(false);
const editingChannel = ref(null);
const savingChannel = ref(false);
const channelErrors = ref([]);

const showAppSecret = ref(false);
const showEncryptKey = ref(false);
const showClientSecret = ref(false);
const showBotToken = ref(false);
const showQQSecret = ref(false);
const showIcooChatSecret = ref(false);

const channelDialogVisible = computed({
    get: () => showChannelDialog.value || !!editingChannel.value,
    set: (val) => { if (!val) closeChannelDialog(); },
});

const channelForm = reactive({
    name: "",
    type: "飞书",
    enabled: true,
    config: {
        port: null,
        path: "",
        app_id: "",
        app_secret: "",
        verification_token: "",
        encrypt_key: "",
        client_id: "",
        client_secret: "",
        agent_id: null,
        bot_token: "",
        webhook_url: "",
        endpoint: "",
        welcome_message: "",
        enable_group_events: true,
        enable_card_message: true,
        allow_from: "",
        max_message_length: 2000,
        group_trigger: "",
        send_markdown: false,
    },
});

const webhookUrlCopied = ref(false);

const enabledChannelCount = computed(() => channels.value.filter((channel) => channel.enabled).length);
const activeChannelTypeCount = computed(() => new Set(channels.value.map((channel) => channel.type)).size);
const filteredChannels = computed(() => {
    let result = channels.value;
    if (searchQuery.value) {
        const keyword = searchQuery.value.toLowerCase();
        result = result.filter((channel) =>
            channel.name?.toLowerCase().includes(keyword) ||
            String(getChannelTypeLabel(channel)).toLowerCase().includes(keyword) ||
            String(getChannelId(channel)).toLowerCase().includes(keyword) ||
            String(getChannelEndpoint(channel)).toLowerCase().includes(keyword),
        );
    }
    if (filterStatus.value === "enabled") {
        result = result.filter((channel) => channel.enabled);
    } else if (filterStatus.value === "disabled") {
        result = result.filter((channel) => !channel.enabled);
    }
    return result;
});
const topChannelTypeLabel = computed(() => {
    if (channels.value.length === 0) return "未设置";
    const counter = new Map();
    for (const channel of channels.value) {
        const label = getChannelTypeLabel(channel);
        counter.set(label, (counter.get(label) || 0) + 1);
    }
    return [...counter.entries()].sort((a, b) => b[1] - a[1])[0]?.[0] || "未设置";
});

function getWebhookUrl() {
    const port = channelForm.config.port || 8082;
    const path = channelForm.config.path || "/feishu/webhook";
    return `http://<your-host>:${port}${path}`;
}

async function copyWebhookUrl() {
    try {
        await navigator.clipboard.writeText(getWebhookUrl());
        webhookUrlCopied.value = true;
        setTimeout(() => { webhookUrlCopied.value = false; }, 2000);
    } catch (err) { console.error("复制失败:", err); }
}

function getChannelTypeLabel(ch) {
    const typeMap = { feishu: "飞书", dingtalk: "钉钉", webhook: "Webhook", telegram: "Telegram", qq: "QQ", icoo_chat: "icoo_chat" };
    return typeMap[ch.type] || ch.type || "未知";
}

function getChannelIcon(ch) {
    const iconMap = { feishu: SendIcon, dingtalk: SendIcon, webhook: WebhookIcon, telegram: SendIcon, qq: QQIcon, icoo_chat: BotIcon };
    return iconMap[ch.type] || MessageSquareIcon;
}

function getChannelStyle(ch) {
    const styles = {
        feishu: { bgClass: "bg-blue-500/10", iconClass: "text-blue-500" },
        dingtalk: { bgClass: "bg-blue-600/10", iconClass: "text-blue-600" },
        webhook: { bgClass: "bg-purple-500/10", iconClass: "text-purple-500" },
        telegram: { bgClass: "bg-sky-500/10", iconClass: "text-sky-500" },
        qq: { bgClass: "bg-green-500/10", iconClass: "text-green-500" },
        icoo_chat: { bgClass: "bg-emerald-500/10", iconClass: "text-emerald-500" },
    };
    return styles[ch.type] || { bgClass: "bg-accent/10", iconClass: "text-accent" };
}

function getChannelEndpoint(ch) {
    try {
        const cfg = typeof ch.config === "string" ? JSON.parse(ch.config || "{}") : ch.config || {};
        if (cfg.endpoint) return cfg.endpoint.length > 36 ? cfg.endpoint.slice(0, 33) + "..." : cfg.endpoint;
        if (cfg.port && cfg.path) return `:${cfg.port}${cfg.path}`;
        if (cfg.port) return `:${cfg.port}`;
        if (cfg.webhook_url) return cfg.webhook_url.length > 25 ? cfg.webhook_url.slice(0, 22) + "..." : cfg.webhook_url;
        return "";
    } catch { return ""; }
}

function getChannelId(ch) {
    try {
        const cfg = typeof ch.config === "string" ? JSON.parse(ch.config || "{}") : ch.config || {};
        if (cfg.app_id) return cfg.app_id;
        if (cfg.client_id) return cfg.client_id;
        return "";
    } catch { return ""; }
}

function validateChannelConfig() {
    channelErrors.value = [];
    const { type, config } = channelForm;
    if (type === "飞书") {
        if (!config.app_id) channelErrors.value.push("App ID 不能为空");
        else if (!config.app_id.startsWith("cli_")) channelErrors.value.push("App ID 格式不正确，应以 cli_ 开头");
        if (!config.app_secret) channelErrors.value.push("App Secret 不能为空");
        if (!config.verification_token) channelErrors.value.push("Verification Token 不能为空");
    } else if (type === "dingtalk") {
        if (!config.client_id) channelErrors.value.push("Client ID 不能为空");
        if (!config.client_secret) channelErrors.value.push("Client Secret 不能为空");
    } else if (type === "telegram") {
        if (!config.bot_token) channelErrors.value.push("Bot Token 不能为空");
        else if (!/^\d+:[A-Za-z0-9_-]+$/.test(config.bot_token)) channelErrors.value.push("Bot Token 格式不正确，应为：数字:字母数字组合");
    } else if (type === "qq") {
        if (!config.app_id) channelErrors.value.push("App ID 不能为空");
        if (!config.app_secret) channelErrors.value.push("App Secret 不能为空");
    } else if (type === "icoo_chat") {
        if (!config.endpoint) channelErrors.value.push("Bot WebSocket Endpoint 不能为空");
        else if (!/^wss?:\/\//.test(config.endpoint)) channelErrors.value.push("Bot WebSocket Endpoint 必须以 ws:// 或 wss:// 开头");
        if (!config.app_id) channelErrors.value.push("App ID 不能为空");
        if (!config.app_secret) channelErrors.value.push("App Secret 不能为空");
    } else if (type === "webhook") {
        if (!config.webhook_url) channelErrors.value.push("Webhook URL 不能为空");
    }
    if (config.port && (config.port < 1 || config.port > 65535)) {
        channelErrors.value.push("端口号必须在 1-65535 之间");
    }
    return channelErrors.value.length === 0;
}

async function toggleChannelEnabled(ch) {
    const newEnabled = !ch.enabled;
    try {
        await api.updateChannel({ id: ch.id, name: ch.name, type: ch.type, enabled: newEnabled, config: ch.config });
        ch.enabled = newEnabled;
    } catch (error) {
        console.error("切换渠道状态失败:", error);
        toast("操作失败: " + error.message, "error");
    }
}

async function loadChannels() {
    loadingChannels.value = true;
    try {
        const response = await api.getChannels();
        channels.value = response.data || [];
    } catch (error) {
        console.error("获取渠道失败:", error);
        channels.value = [];
    }
    loadingChannels.value = false;
}

function resetChannelForm() {
    channelForm.name = "";
    channelForm.type = "feishu";
    channelForm.enabled = true;
    channelForm.config = {
        port: null, path: "",
        app_id: "", app_secret: "", verification_token: "", encrypt_key: "",
        client_id: "", client_secret: "", agent_id: null,
        bot_token: "", webhook_url: "",
        endpoint: "",
        welcome_message: "", enable_group_events: true, enable_card_message: true,
        allow_from: "", max_message_length: 2000, group_trigger: "", send_markdown: false,
    };
    channelErrors.value = [];
    showAppSecret.value = false;
    showEncryptKey.value = false;
    showClientSecret.value = false;
    showBotToken.value = false;
    showQQSecret.value = false;
    showIcooChatSecret.value = false;
}

function openAddChannel() {
    editingChannel.value = null;
    resetChannelForm();
    showChannelDialog.value = true;
}

function openEditChannel(ch) {
    editingChannel.value = ch;
    channelForm.name = ch.name;
    channelForm.type = ch.type || "飞书";
    channelForm.enabled = ch.enabled;
    channelErrors.value = [];
    try {
        const cfg = typeof ch.config === "string" ? JSON.parse(ch.config || "{}") : ch.config || {};
        Object.assign(channelForm.config, {
            port: cfg.port ?? null, path: cfg.path || "",
            app_id: cfg.app_id || "", app_secret: "", verification_token: cfg.verification_token || "", encrypt_key: cfg.encrypt_key || "",
            client_id: cfg.client_id || "", client_secret: "", agent_id: cfg.agent_id ?? null,
            bot_token: "", webhook_url: cfg.webhook_url || "", endpoint: cfg.endpoint || "",
            welcome_message: cfg.welcome_message || "", enable_group_events: cfg.enable_group_events !== false, enable_card_message: cfg.enable_card_message !== false,
            allow_from: cfg.allow_from || "", max_message_length: cfg.max_message_length || 2000, group_trigger: cfg.group_trigger || "", send_markdown: !!cfg.send_markdown,
        });
    } catch {
        resetChannelForm();
        channelForm.name = ch.name;
        channelForm.type = ch.type || "飞书";
        channelForm.enabled = ch.enabled;
    }
    showAppSecret.value = false;
    showEncryptKey.value = false;
    showClientSecret.value = false;
    showBotToken.value = false;
    showQQSecret.value = false;
    showIcooChatSecret.value = false;
    showChannelDialog.value = true;
}

function closeChannelDialog() {
    showChannelDialog.value = false;
    editingChannel.value = null;
    channelErrors.value = [];
}

async function handleSaveChannel() {
    if (!channelForm.name) return;
    if (!validateChannelConfig()) return;
    savingChannel.value = true;
    const data = {
        name: channelForm.name.trim(),
        type: channelForm.type,
        enabled: channelForm.enabled,
        config: JSON.stringify(channelForm.config),
    };
    try {
        if (editingChannel.value) {
            await api.updateChannel({ id: editingChannel.value.id, ...data });
        } else {
            await api.createChannel(data);
        }
        await loadChannels();
        closeChannelDialog();
    } catch (error) {
        console.error("保存渠道失败:", error);
        toast("保存渠道失败: " + error.message, "error");
    }
    savingChannel.value = false;
}

async function handleDeleteChannel(ch) {
    const ok = await confirm(`确定要删除渠道 "${ch.name}" 吗？`);
    if (!ok) return;
    try {
        await api.deleteChannel(ch.id);
        await loadChannels();
    } catch (error) {
        console.error("删除渠道失败:", error);
        toast("删除渠道失败: " + error.message, "error");
    }
}

onMounted(() => { loadChannels(); });
</script>
