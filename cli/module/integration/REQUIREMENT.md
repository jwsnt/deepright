### 第一性原则
+ 仅可以新增/更新/删除当前需求文档（REQUIREMENT.md）同目录及其子目录下的文件和文件夹

### 技术规范
+ 严格遵守整体设计文档：../DESIGN.md

### 需求介绍
> 新增自 ./iteration/20260517-1/REQUIREMENT.md
+ 收口Proxy需求：
    + 接口/api/plugins/meta不缓存，每次都实时读取
    + 接口/api/plugins/meta读取每个插件的scope命令，获取容器可配置项列表
    + 接口/api/plugins/meta获取每个插件的name、param、scope命令需要并发执行，使用最短时间方案
+ 将所有模块整合为一个完整的HTTP服务

### 整合模块
+ cli-get模块：
    + cli-get介绍：cli-get/REQUIREMENT.md
    + cli-get手册：cli-get/USER_GUIDE.md
    + cli-get迭代：cli-get/iteration/日期/REQUIREMENT.md
+ agent模块：
    + agent介绍：agent/REQUIREMENT.md
    + agent手册：agent/USER_GUIDE.md
    + agent迭代：agent/iteration/日期/REQUIREMENT.md
+ cron模块：
    + cron介绍：cron/REQUIREMENT.md
    + cron手册：cron/USER_GUIDE.md
    + cron迭代：cron/iteration/日期/REQUIREMENT.md
+ proxy模块：
    + proxy介绍：proxy/REQUIREMENT.md
    + proxy手册：proxy/USER_GUIDE.md
    + proxy迭代：proxy/iteration/日期/REQUIREMENT.md
+ static模块：
    + static介绍：static/REQUIREMENT.md
    + static手册：static/USER_GUIDE.md
    + static迭代：static/iteration/日期/REQUIREMENT.md
+ connect模块：
    + connect介绍：connect/REQUIREMENT.md
    + connect手册：connect/USER_GUIDE.md
    + connect迭代：connect/iteration/日期/REQUIREMENT.md
    + plugins模块：
        + 浏览器插件：
            + 浏览器插件介绍：connect/browser/REQUIREMENT.md
            + 浏览器插件手册：connect/browser/USER_GUIDE.md
            + 浏览器插件迭代：connect/browser/iteration/日期/REQUIREMENT.md
        + 飞书插件：
            + 飞书插件介绍：connect/feishu/REQUIREMENT.md
            + 飞书插件手册：connect/feishu/USER_GUIDE.md
            + 飞书插件迭代：connect/feishu/iteration/日期/REQUIREMENT.md
        + 邮件插件：
            + 邮件插件介绍：connect/email/REQUIREMENT.md
            + 邮件插件手册：connect/email/USER_GUIDE.md
            + 邮件插件迭代：connect/email/iteration/日期/REQUIREMENT.md
        + 其他插件：
            + 其他插件需要遵守connect/PLUGIN.md定义的统一插件规范
            + 其他插件介绍：connect/插件名/REQUIREMENT.md
            + 其他插件手册：connect/插件名/USER_GUIDE.md
            + 其他插件迭代：connect/插件名/iteration/日期/REQUIREMENT.md
+ knowledge模块：
    + knowledge介绍：knowledge/REQUIREMENT.md
    + knowledge手册：knowledge/USER_GUIDE.md
    + knowledge迭代：knowledge/iteration/日期/REQUIREMENT.md
+ 包括以上模块关联的模块

### 二进制收口原则
+ 最终交付给用户的主程序必须是`integration`一个二进制文件（不包含插件plugins目录下文件）
+ `integration`必须同时具备以下能力：
    + 作为`proxy`的HTTP服务
    + 作为`connect`的HTTP服务
    + 作为`connect`的CLI代理
    + 作为`site`的静态站点服务
+ 最终使用者只应感知`integration`，不应要求用户额外理解、安装、启动或调用`connect`
> 新增自 iteration/20260509-1/REQUIREMENT.md
+ 转发/v1/chat/completions及cli/get和cli/pub提交metadata的Agent元数据中加入plugins信息
> 新增自 iteration/20260510-1/REQUIREMENT.md
+ 转发/v1/chat/completions及cli/get和cli/pub提交metadata的Agent元数据中加入knowledge信息
> 新增自 iteration/20260511-2/REQUIREMENT.md
+ 转发/v1/chat/completions及cli/get和cli/pub提交metadata的Agent元数据中`agents[].skills`每次都需要实时遍历指定目录及其子孙目录后提取文件内容，不要缓存

### CLI收口要求
+ 任何模块（如`connect`模块）对外暴露的CLI能力，必须在`integration`顶层直接可用（不包含插件plugins目录下文件）
+ 示例：
```bash
./integration connect meta-create
./integration connect meta-update
./integration connect meta-delete
./integration connect meta-get
./integration connect meta-list
./integration connect add-request
./integration connect request-list
./integration connect add-response
./integration connect response-list
```

### Knowledge能力
> 新增自 iteration/20260510-1/REQUIREMENT.md
+ 收口Knowledge模块能力：
    + 支持`./integration knowledge update-time [时间戳]`
    + 共享knowledge目录和最后更新时间元数据到统一Agent metadata
> 新增自 iteration/20260511-1/REQUIREMENT.md
+ 通过集成后的Proxy能力对外提供知识库接口：
    + HTTP GET `/knowledge`
    + HTTP GET `/knowledge/<相对路径>`
    + HTTP GET `/knowledge_lastUpdate`

> 新增自 ./iteration/20260515-2/REQUIREMENT.md
+ 收口Proxy Knowledge更新锁/interval逻辑：
    + 转发/v1/chat/completions前检查knowledge.lastUpdate是否需要更新
    + Proxy需求：../proxy/iteration/20260515-3/REQUIREMENT.md
> 新增自 ./iteration/20260516-1/REQUIREMENT.md
+ 收口Proxy knowledge_commit功能：
    + 支持metadata中的knowledge_commit:true请求
    + 包含knowledge_commit的请求在SSE响应结束后更新知识库最后更新时间
    + Proxy需求：../proxy/iteration/20260516-1/REQUIREMENT.md
> 新增自 ./iteration/20260516-2/REQUIREMENT.md
+ 收口Proxy知识库接口扩展：
    + HTTP GET `/knowledge_path`：获取知识库真实文件系统绝对路径
    + Proxy需求：../proxy/iteration/20260516-2/REQUIREMENT.md
    + Proxy需求：../proxy/iteration/20260511-2/REQUIREMENT.md
> 新增自 ./iteration/20260515-2/REQUIREMENT.md
+ 收口Proxy Knowledge更新锁/interval逻辑：
    + 转发/v1/chat/completions前检查knowledge.lastUpdate是否需要更新
    + Proxy需求：../proxy/iteration/20260515-3/REQUIREMENT.md
> 新增自 ./iteration/20260516-1/REQUIREMENT.md
+ 收口Proxy knowledge_commit功能：
    + 支持metadata中的knowledge_commit:true请求
    + 包含knowledge_commit的请求在SSE响应结束后更新知识库最后更新时间
    + Proxy需求：../proxy/iteration/20260516-1/REQUIREMENT.md
> 新增自 ./iteration/20260516-2/REQUIREMENT.md
+ 收口Proxy知识库接口扩展：
    + HTTP GET `/knowledge_path`：获取知识库真实文件系统绝对路径
    + Proxy需求：../proxy/iteration/20260516-2/REQUIREMENT.md


### Cron CLI命令
> 新增自 iteration/20260503-1/REQUIREMENT.md
+ HTTP服务与CLI命令在统一命令上兼容，既可启动HTTP也可执行CLI操作
+ 创建时校验：Agent必须存在、模型必须在Proxy已注册Token、参数符合格式要求
+ 模型Token由调用时从Sqlite动态获取
+ 非Cron模块必填参数从主应用 `config/config.json` 获取

#### 创建任务
> 新增自 iteration/20260503-1/REQUIREMENT.md
+ `./integration cron create`：创建备忘录（定时任务）元数据
    + 必填：--content（任务内容）、--model（模型）、--rawTime（首次开始时间）、--cycle（执行周期）、--agent（AgentId）
    + 可选：--thinking（深度思考）、--chatId（会话ID）
    + cycle：0=仅一次/1=工作日/2=自然日/3=每小时/4=每15分钟/5=每30分钟
+ `./integration cron create-cron`：创建自定义Cron任务元数据
    + 必填：--content、--model、--cron（Cron表达式）、--agent
    + 可选：--thinking、--chatId
> 补充自 iteration/20260503-2/REQUIREMENT.md
+ `cron create`与`cron create-cron`在未启动HTTP服务时也须先完成共享sqlite初始化并复用与HTTP一致的校验逻辑

#### 查询任务
> 新增自 iteration/20260503-3/REQUIREMENT.md
+ `./integration cron find-meta`：查询任务元数据，支持条件：AgentId、Chat、模型、开始执行时间范围、执行周期，未指定则全匹配
+ `./integration cron find-detail`：查询任务明细，支持条件：元数据Id、AgentId、Chat、模型、执行周期、执行时间范围，未指定时间仅查当前时间之后

#### 删除任务
> 新增自 iteration/20260503-4/REQUIREMENT.md
+ `./integration cron delete-meta --id`：删除任务元数据（同时删除关联明细）
+ `./integration cron delete-detail --metaId/--detailId`：删除任务明细

### 启动关闭
+ 使用start命令启动，并指定启动参数（或默认值），启动后进入后台进程（并创建pid文件）
    > 新增自 iteration/20260509-2/REQUIREMENT.md
    + 每次start前和stop后清理应用启动目录下的*.pid文件,防止意外关闭导致数据污染
    > 新增自 iteration/20260509-3/REQUIREMENT.md
    + start时端口占用或其他异常需输出明确信息到控制台和日志
    + 命令默认参数：
        + --agent-dir 当前应用启动目录的agent目录，如果不存在则自动创建
            + 如果agent-dir指定目录为空（包括自动创建），则为该目录自动创建以下2个目录：
                + DEF_AGENT、DEF_AGENT/skills
        + --site 当前应用启动目录的site目录
        + --host https://www.deepright.cn
+ 使用stop命令关闭，关闭前释放应用开启的所有资源（定时任务、连接池、子进程等）后安全关闭，避免僵尸进程
    > 新增自 iteration/20260508-1/REQUIREMENT.md
    + stop命令关闭时,不能因内部错误（子进程、插件等）导致无法关闭,仅在控制台和日志提示,正常关闭进程和端口
    + 确保在关闭服务时也能正确关闭插件拉起的所有子进程，不留下残留后台进程
    + 包括关闭所有已启动插件：通过meta-list获取后进行stop操作
        + Connect模块：../connect/iteration/20260504-2/REQUIREMENT.md
        + 例如飞书插件：
        ```
        plugin/feishu stop
        ```
    > 新增自 iteration/20260509-3/REQUIREMENT.md
    + stop无进程可关闭时默认成功
+ 使用restart命令重启，先执行关闭然后执行启动
    + 使用restart必须默认继承上一次start的有效启动参数且仅在HTTP服务真实ready后才返回成功，避免参数丢失和假启动成功导致的重启后异常
+ 使用start、stop、restart的执行日志记录在同目录的integration.log，不要在命令（控制台）中返回

###### 默认启动参数
> 补充自 iteration/20260503-5/REQUIREMENT.md
+ --agent-dir默认为当前应用目录下的agent，不存在则自动创建
+ --site默认为当前应用目录下的site

###### 启动后自动打开浏览器
> 补充自 iteration/20260503-5/REQUIREMENT.md
+ HTTP服务启动成功后自动打开系统默认浏览器，访问`http://127.0.0.1:8080/site/#app`（端口随--port参数变化）

### 启动参数持久化
> 新增自 iteration/20260503-2/REQUIREMENT.md
+ 启动HTTP服务时将所有启动参数保存至应用启动目录下的 `config/config.json`，每次启动覆盖更新
+ 案例：
```
./integration --agent-dir /agent/ --site ../site
```
保存内容：
```
{
    "agent-dir": "/agent/",
    "site": "../site"
}
```

### 数据库路径规范
> 新增自 iteration/20260503-6/REQUIREMENT.md
+ 所有使用数据库路径统一为应用启动路径下的data目录
    + 如应用路径为/home/integration，数据库路径使用绝对路径/home/integration/data
+ 将数据库加载绝对路径写入 `config/config.json`
+ 修正当前所有不符合的数据库路径规范

### Skills解析检查收口
> 新增自 iteration/20260511-3/REQUIREMENT.md
+ 收口Skills模块定期解析检查和Proxy `/skills_warning` 接口
+ Proxy需求：../proxy/iteration/20260511-3/REQUIREMENT.md
+ Skills需求：../skills/iteration/20260512-1/REQUIREMENT.md

### Git实时路径与安装检查收口
> 新增自 iteration/20260512-1/REQUIREMENT.md
+ 收口Agent git实时查询和Proxy `/install_app` 接口
+ Proxy需求：../proxy/iteration/20260512-2/REQUIREMENT.md
+ Agent需求：../agent/iteration/20260512-1/REQUIREMENT.md

> 新增自 ./iteration/20260614-2/REQUIREMENT.md
+ 技术收口：
    + 修改接口/install_app，从主应用的config.json中的"install_app"读取数据，需要区分操作系统：
    ```
    {
        ...
        "install_app": {
            "linux": [...],
            "wsl": [...],
            "mac": [...],
        }
    }
    ```
        + 其中Linux系统使用linux，Mac系统使用mac，Windows（WSL）使用wsl
        + 数据结构不变，依旧是string array
        + --install_app参数不变，如果存在所有操作系统结构都要追加
    + 所有install_app的元素表示一个本地应用名称，需要检查是否已安装，已安装则从返回列表中删除
        + 不同操作系统判断方式不同
        + 接口缓存5分钟
    + Proxy需求：../proxy/iteration/20260614-1/REQUIREMENT.md

### 日志记录与查询收口
> 新增自 iteration/20260513-1/REQUIREMENT.md
+ 收口Proxy和CLI/GET的日志写入、读取能力
+ Proxy写入日志需求：../proxy/iteration/20260513-1/REQUIREMENT.md
+ Proxy读取日志需求：../proxy/iteration/20260513-2/REQUIREMENT.md
+ CLI/GET写入日志需求：../cli-get/iteration/20260513-1/REQUIREMENT.md

### Metadata透传收口
> 新增自 iteration/20260514-1/REQUIREMENT.md
+ 收口Proxy metadata透传能力
+ Proxy需求：../proxy/iteration/20260516-2/REQUIREMENT.md

### Compatibility数组兼容收口
> 新增自 iteration/20260515-1/REQUIREMENT.md
+ 收口Skills、Agent、Proxy三模块的compatibility数组兼容能力
+ Proxy需求：../proxy/iteration/20260515-2/REQUIREMENT.md
+ Agent需求：../agent/iteration/20260515-1/REQUIREMENT.md
+ Skill需求：../skills/iteration/20260515-1/REQUIREMENT.md

### Skills Warning能力
> 新增自 ./iteration/20260512-1/REQUIREMENT.md
+ 收口 Proxy Skills Warning 能力，通过集成后的 Proxy 对外提供 `/skills_warning` 接口
+ Skills需求：../skills/iteration/20260512-1/REQUIREMENT.md
+ Proxy需求：../proxy/iteration/20260512-1/REQUIREMENT.md

### Git路径实时获取
> 新增自 ../proxy/iteration/20260512-2/REQUIREMENT.md
+ 转发 `/v1/chat/completions`、`/cli/get`、`/cli/pub` 提交 metadata 的 git 路径每次实时获取，不要缓存
+ 收口 `/install_app` 能力：通过集成后的 Proxy 对外提供 `/install_app` 接口
+ Agent需求：../agent/iteration/20260512-1/REQUIREMENT.md
+ Proxy需求：../proxy/iteration/20260512-2/REQUIREMENT.md

### 日志记录与查询
> 新增自 ./iteration/20260513-1/REQUIREMENT.md
+ 收口 Proxy 日志记录能力到 Integration，统一日志存储与查询接口
+ 支持 `/log_skill`、`/log_skill_status` 等日志查询导出接口
+ Proxy日志需求：../proxy/iteration/20260513-1/REQUIREMENT.md
+ Proxy日志查询需求：../proxy/iteration/20260513-2/REQUIREMENT.md
+ Proxy日志状态需求：../proxy/iteration/20260513-3/REQUIREMENT.md

### Metadata透传
> 新增自 ./iteration/20260516-2/REQUIREMENT.md
+ 收口 Proxy `/v1/chat/completions` metadata 透传能力
+ Proxy需求：../proxy/iteration/20260516-2/REQUIREMENT.md

### Skills兼容数组compatibility
> 新增自 ./iteration/20260515-1/REQUIREMENT.md
+ Agent 的 skill 属性需要兼容数组形式的 compatibility 属性
+ Agent需求：../agent/iteration/20260515-1/REQUIREMENT.md
+ Proxy需求：../proxy/iteration/20260515-2/REQUIREMENT.md

### 文件或目录最后时间
> 新增自 ./iteration/20260516-1/REQUIREMENT.md
+ 收口Proxy文件或目录最后时间能力：
    + 支持`/file/lastUpdate`接口查询文件/目录最后更新时间距当前的毫秒数
    + Proxy需求：../proxy/iteration/20260515-4/REQUIREMENT.md

### 文件或目录最后时间
> 新增自 ./iteration/20260516-1/REQUIREMENT.md
+ 收口Proxy文件或目录最后时间能力：
    + 支持`/file/lastUpdate`接口查询文件/目录最后更新时间距当前的毫秒数
    + Proxy需求：../proxy/iteration/20260515-4/REQUIREMENT.md

> 新增自 ./iteration/20260524-1/REQUIREMENT.md
+ 技术规范的指导下收口：
    + 虚拟文件系统在任意当前目录下点击"新建文件或目录"时，输入合法名称必须创建到当前浏览目录，不得因为路径中含有`/`而误报name contains invalid characters
    + Proxy需求：../proxy/iteration/20260524-1/REQUIREMENT.md
    + Site需求：../site/iteration/20260524-1/REQUIREMENT.md

### 插件文件类型识别收口
> 新增自 ./iteration/20260603-1/REQUIREMENT.md
+ 技术规范的指导下收口：
    + 接口/api/plugins/meta判断plugins目录文件是否为插件的条件：
        + 没有后缀名的应用程序，后缀名为py、js或go的脚本文件
        + 跳过目录
    + 如果判断出现错误，不要报错或崩溃，仅跳过该文件并输出日志
        + Proxy需求：../proxy/iteration/20260603-1/REQUIREMENT.md

### 服务启动自动打开浏览器
> 新增自 ./iteration/20260605-1/REQUIREMENT.md
+ integration在服务启动后的浏览器打开应自动打开的地址应为http://localhost:<port>/site/#app的页面，其中port为--port指定端口，或默认值
+ 打开浏览器的动作应在服务启动后延迟约200ms触发异步执行，不阻塞integration主服务启动流程
+ 自动打开浏览器时，应优先以"最大化窗口"方式启动，如Chrome的--start-maximized
+ 在MacOS上，应优先按以下顺序查找并启动浏览器：Google Chrome、Google Chrome for Testing、Microsoft Edge、Brave Browser、Chromium
+ 在Linux上，应优先按以下顺序查找并启动浏览器：google-chrome、google-chrome-stable、chromium-browser、chromium、microsoft-edge、microsoft-edge-stable、brave-browser
+ 在Windows上（含WSL），应优先从常见安装目录查找并启动以下浏览器：Chrome、Edge、Chromium；若未命中，再从 PATH 中查找 chrome、msedge、chromium、brave
+ 当命中上述优先浏览器时，应直接启动该浏览器并附带最大化参数打开目标URL
+ 当未命中优先浏览器时，应回退到系统默认浏览器：MacOS使用open，Linux使用xdg-open，Windows使用 cmd /c start /max
+ 若当前操作系统不在支持范围内，应返回"unsupported OS"错误
+ 若自动打开浏览器失败，不应影响integration服务继续运行，系统只记录失败日志
+ 自动打开成功后，应记录成功日志，包含实际打开的 URL

### 插件远程执行接口收口
> 新增自 ./iteration/20260606-1/REQUIREMENT.md
+ 技术规范的指导下收口：
    + 新增/api/plugins/exec?key=x&command=y&param1=value1&param2=value2&...执行指定插件的指定命令
        + key：插件标识、command：该插件的命令
        + param1=value1&param2=value2&...：可以有任意组，表示插件参数
        + Proxy需求：../proxy/iteration/20260606-1/REQUIREMENT.md
    + /api/plugins/exec等待插件执行命令超时等待改为由integration/proxy参数--plugin_exec_timeout决定的毫秒数，默认600秒，如果超时或启动失败需要在integration.log保留日志
    + 如果插件完成了则立即返回

### 沙盒模式收口
> 新增自 ./iteration/20260608-1/REQUIREMENT.md
+ 技术规范的指导下收口：
    + 新增/api/sandbox=枚举值?agentId=x&chatId=y，为指定AgentID+ChatID开启或关闭沙盒模式，关闭则删除对应Agent+Chat的数据
    + 新增/api/sandbox_status?agentId=x&chatId=y，获取指定AgentID+ChatID的沙盒模式
    + 修改/api/cmd也需要参与沙盒执行判断
    + Proxy需求：../proxy/iteration/20260608-1/REQUIREMENT.md

> 新增自 ./iteration/20260618-1/REQUIREMENT.md
+ 区分系统（MAC或是Windows/WSL）调用不同沙盒方案（包括目录选择）
+ 严格隔离MAC系统的实现路径，完全保持原样
+ WSL沙盒需求：../cli-get/sandbox/wsl/REQUIREMENT.md
+ Proxy需求：../proxy/iteration/20260618-1/REQUIREMENT.md

### 延迟关机
> 新增自 ./iteration/20260608-2/REQUIREMENT.md
+ 新增/api/shutdown，启动延迟进程，5秒后关闭自身主进程，回收释放所有由主进程启动的资源（包括插件）
+ 插件关闭同integration stop，由插件自身stop命令关闭

### Agent元数据扩展收口
> 新增自 ./iteration/20260610-1/REQUIREMENT.md
+ 转发/v1/chat/completions时不再额外带顶层metadata.agent，当前Agent信息统一通过metadata.agentId + metadata.agents[]获取
+ Proxy需求：../proxy/iteration/20260610-1/REQUIREMENT.md

### 插件参数结构调整收口
> 新增自 ./iteration/20260610-2/REQUIREMENT.md
+ 修改/api/plugins/meta中每个插件的param的参数结构为[{"key":"val"},{"key":"val"},...]
+ Proxy需求：../proxy/iteration/20260610-2/REQUIREMENT.md

### 技能动态注入收口
> 新增自 ./iteration/20260610-3/REQUIREMENT.md
+ 修改/api/skills?agentId=xxx，在返回结果中加入：__internal_cron（无论如何都增加）
+ 如果开启了browser插件则增加：__internal_browser
+ 如果开启了remote插件则增加：__internal_remote
+ Proxy需求：../proxy/iteration/20260610-3/REQUIREMENT.md

> 新增自 ./iteration/20260614-4/REQUIREMENT.md
+ 技术收口：
    + 修改/api/skills?agentId=xxx：
        + 如果开启了browser插件（需要监测是否开启状态）则增加：__internal_browser
        + 如果开启了remote插件（需要监测是否开启状态）则增加：__internal_remote
        + 同时从主应用的config.json的将skills数据（string array）追加到结果
        + 原本__internal_cron的会改为从config.json读取
    ```
    {
        ...
        "skills": [
            "__internal_cron",
            ...
        ]
    }
    ```
    + 原需求：./iteration/20260610-3/REQUIREMENT.md
    + 不需要兼容，改为最新
    + Proxy需求：../proxy/iteration/20260614-3/REQUIREMENT.md

### 蜂群Agent查询收口
> 新增自 ./iteration/20260611-1/REQUIREMENT.md
+ 新增/api/swarm_agent，获取当前启动了蜂群的Agent名称（router_disable=false）
+ Agent名单中不能包含当前Agent
+ Proxy需求：../proxy/iteration/20260611-1/REQUIREMENT.md

### 运行时修改远程主机
> 新增自 ./iteration/20260611-2/REQUIREMENT.md
+ 新增/api/host，运行时修改--host或config.json或默认值指定的远程主机地址
+ 仅在运行时有效，重启后恢复原配置，不持久化

### 单机模式
> 新增自 ./iteration/20260613-1/REQUIREMENT.md
+ 新增/api/standalone=true/false，开启后所有api接口仅允许本机localhost/127.0.0.1调用
+ 非本机请求直接断开连接，不返回任何响应
+ --port端口下所有服务都需要禁止，包括静态页面

### 远程访问安全限制
> 新增自 ./iteration/20260613-2/REQUIREMENT.md
+ 非localhost/127.0.0.1访问时：
    + 模型与密钥数据中的密钥在接口读取时全部用10个*替代，禁止新增模型
    + 禁止使用插件参数读取/更改、重启插件、关闭插件接口，仅可读取日志

### Agent导入导出收口
> 新增自 ./iteration/20260614-1/REQUIREMENT.md
+ 增加/api/agent/export?agent_id=xxx，用于导出指定Agent目录（文件、子孙目录）
    + 文件过滤规则（一级目录）：
        + 去掉chrome开头的目录
        + 去掉data目录
        + 去掉tmp目录
    + 导出结构为.zip包
+ 增加/api/agent/import，用于导入Agent目录（输入数据为export zip或目录）
    + 导入前检查是否已经存在重名目录，如果存在需要拒绝导入，并提示先删除同名Agent
    + 需要支持export导出的zip结构，并解压后删除zip
    + 需要支持直接导入目录结构

### HTTP转发超时控制
> 新增自 ./iteration/20260615-1/REQUIREMENT.md
+ 转发下游（--host指定）服务的HTTP超时调整如下
    + --http_connect_timeout：默认15000（15秒）
    + --http_socket_timeout：默认45000（45秒）
    + --http_timeout：默认45000（45秒）
    + 所有HTTP请求强制使用HTTP/1.1
+ 先从主应用的config.json中的http读取以上配置，如果不存在则使用默认值：
```
{
    ...
    "http": {
        "http_connect_timeout":
        "http_socket_timeout":
        "http_timeout":
    }
}
```
+ 如果主应用的config.json中http.debug=true，则记录cli/get和cli/pub的详细日志，默认为false
```
{
    ...
    "http": {
        ...
        "debug": true
    }
}
```
    + 明细日志包括：
        + cli/get的发起时间
        + cli/get的响应：
            + 如果超时，记录超时的时间和原因
            + 如果有待执行renew，记录原始报文和时间
        + cli/pub：
            + 开始执行的时间
            + 完成执行的状态、结果、时间

### 插件MD5更新校验
> 新增自 ./iteration/20260615-3/REQUIREMENT.md
+ 点击应用时（integration）：
    + 检查安装包下每个插件的MD5和复制目标目录下插件的MD5是否一致，只有不一致时才允许更新该插件
    + 检查对象固定为：安装包内plugins目录下的插件二进制 与 运行时复制目标目录下的插件二进制
    + 当检测到有插件需要更新时：
        + 如果当前已启动（端口已占用）则弹窗提醒：有插件需要更新，请重启应用
        + 当前已启动时，禁止直接覆盖、截断、重写运行时目录中的插件二进制文件，避免运行中的可执行文件被修改
        + 如果当前未启动，则先完成所有需要更新插件的复制，再继续启动
    + 插件二进制复制/更新的技术收口：
        + MD5一致时禁止重复写入目标插件文件
        + MD5不一致时，必须使用临时文件 + rename 的`原子替换`方式更新插件二进制，禁止直接对目标文件执行覆盖写入
        + 单个插件更新失败时，必须保留旧文件，禁止留下半写入文件、0字节文件或临时文件残留
        + 多个插件需要更新时，需要逐个插件独立完成原子替换，不能先删除全部旧文件后再统一写入
+ 无论是否已启动，都需要在最后打开http://localhost:8080/site/#app
    + 如果是MAC OS，在打开http://localhost:8080/site/#app前判断系统Chrome是否已经打开了，如果是就激活这个Tab，如果不是则打开新页面

### Windows WSL默认路径规范
> 新增自 ./iteration/20260615-4/REQUIREMENT.md
+ 如果当前操作系统是Windows WSL则
    + --agent-dir默认为~/deepright/agent
    + 插件plugins目录为~/deepright/plugins
    + 知识库knowledge目录为~/deepright/knowledge
    + integration.log和integration.pid存放在~/deepright
+ 统一收口$HOME/deepright/
+ 如果目录不存在则创建

### 编写代码
+ 以Golang编写以上代码，要求：
    + 整合所有模块为一个独立、完整的HTTP服务，唯一端口号由命令行参数--port指定（默认8080）
    + 插件加载固定使用启动目录下的plugins目录，禁止任何候选回退
    + proxy和static模块均使用统一端口、不同路径
    + cli-get模块以HTTP服务的后台线程形式启动
    + 相同名称的命令行参数共享
    + 代码简洁，包体积越小越好
    + 能用开源包的就用开源包

### 帮助命令
+ 增加help为所有Integration支持、代理和集成的CLI命令提供使用手册和案例（User Guide/Usage）
    + 参考帮助需求：./iteration/20260503-4/REQUIREMENT.md

### 构建交付
+ 构建并编译代码，放在../release目录下：
    + 目录../site下index.html、icon.png、sw.js需要放在`../release/site`下
    + 目录/../connect下的插件二进制程序需要放在`../release/plugins`下
        + 新增插件/connect/PLUGIN.md自动扫描connect目录, 编译后放进release/plugins，并把插件的release资源一并复制过去
        + 已知插件为browser、feishu、email，以及其他符合connect/PLUGIN.md规范的插件
    + 独立运行的integration二进制程序放在../release
+ 编写最新的构建脚本build.sh

### 集成手册
+ 编写INTEGRATION.md记录和追加集成日志
+ 检查所有模块迭代逻辑，保证最新需求无集成遗漏

### 验证测试
+ 整合后的模块均可通过自身`REQUIREMENT.md`要求的验证测试
> 新增自 iteration/20260517-2/REQUIREMENT.md
+ add-request命令新增可选参数--schema（Json String）
> 新增自 iteration/20260517-3/REQUIREMENT.md
+ 定时器执行明细时有response_schema则转发时附加metadata.response_schema
+ 通过META_ID找到add-request后用插件send回复前对SSE响应JSON标准化
> 新增自 iteration/20260519-1/REQUIREMENT.md
+ 插件日志统一路径release/plugins/插件名.log，通过/api/plugins/log读取
> 新增自 iteration/20260519-2/REQUIREMENT.md
+ /api/token增加__url、__model、__model_fast、__model_thinking、__model_multi_input、__model_multi_output
+ token命令返回增加以上属性
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 转发时如模型配置了__url、__model等属性则加入metadata
> 新增自 iteration/20260520-2/REQUIREMENT.md
+ /api/config支持删除指定模型
> 新增自 iteration/20260521-1/REQUIREMENT.md
+ 技术规范的指导下收口：
    + 修改每30秒扫描一次add-request的消息，立即转换为任务明细（task_detail）
    + 修改将add-request待处理消息的桥接逻辑修改为每30秒扫描一次，且对命中的可处理文本消息无需等待10分钟老化、应在扫描命中后立即转换为task_detail，仅无文本内容的消息继续按过期规则处理。
+ Proxy需求：../proxy/iteration/20260521-1/REQUIREMENT.md
> 新增自 iteration/20260517-2/REQUIREMENT.md
+ add-request命令新增可选参数--schema（Json String），对应任务明细的response_schema
> 新增自 iteration/20260517-3/REQUIREMENT.md
+ 定时器执行任务明细时如有response_schema，则转发/v1/chat/completions时附加到metadata.response_schema
+ 通过META_ID找到原始add-request后，使用对应插件send回复前对SSE响应进行JSON标准化处理
> 新增自 iteration/20260519-1/REQUIREMENT.md
+ 插件日志统一路径为release/plugins/插件名.log，通过/api/plugins/log?name=插件名读取
> 新增自 iteration/20260519-2/REQUIREMENT.md
+ /api/token接口增加__url、__model、__model_fast、__model_thinking、__model_multi_input、__model_multi_output属性并持久化
+ token命令返回增加以上属性
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 转发/v1/chat/completions时如当前模型配置了__url、__model等属性则加入metadata中传递
> 新增自 iteration/20260520-2/REQUIREMENT.md
+ /api/config支持删除指定模型
> 新增自 iteration/20260517-2/REQUIREMENT.md
+ add-request命令新增可选参数--schema（Json String），对应任务明细的response_schema
> 新增自 iteration/20260517-3/REQUIREMENT.md
+ 定时器执行任务明细时如有response_schema，则转发/v1/chat/completions时附加到metadata.response_schema
+ 通过META_ID找到原始add-request后，使用对应插件send回复前对SSE响应进行JSON标准化处理
> 新增自 iteration/20260519-1/REQUIREMENT.md
+ 插件日志统一路径为release/plugins/插件名.log，通过/api/plugins/log?name=插件名读取
> 新增自 iteration/20260519-2/REQUIREMENT.md
+ /api/token接口增加__url、__model、__model_fast、__model_thinking、__model_multi_input、__model_multi_output属性并持久化
+ token命令返回增加以上属性
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 转发/v1/chat/completions时如当前模型配置了__url、__model等属性则加入metadata中传递
> 新增自 iteration/20260520-2/REQUIREMENT.md
+ /api/config支持删除指定模型
> 新增自 iteration/20260517-2/REQUIREMENT.md
+ add-request命令新增可选参数--schema（Json String），对应任务明细的response_schema
> 新增自 iteration/20260517-3/REQUIREMENT.md
+ 定时器执行任务明细时如有response_schema，则转发/v1/chat/completions时附加到metadata.response_schema
+ 通过META_ID找到原始add-request后，使用对应插件send回复前对SSE响应进行JSON标准化处理
> 新增自 iteration/20260519-1/REQUIREMENT.md
+ 插件日志统一路径为release/plugins/插件名.log，通过/api/plugins/log?name=插件名读取
> 新增自 iteration/20260519-2/REQUIREMENT.md
+ /api/token接口增加__url、__model、__model_fast、__model_thinking、__model_multi_input、__model_multi_output属性并持久化
+ token命令返回增加以上属性
> 新增自 iteration/20260520-1/REQUIREMENT.md
+ 转发/v1/chat/completions时如当前模型配置了__url、__model等属性则加入metadata中传递
> 新增自 iteration/20260520-2/REQUIREMENT.md
+ /api/config支持删除指定模型


### 新建Agent默认配置
> 新增自 ./iteration/20260524-2/REQUIREMENT.md
+ 技术规范的指导下收口：
    + 新建 Agent 后，将由参数 --default-dir 指定的目录内容复制到 Agent 目录，默认为应用启动程序所在目录下的 config 目录

> 新增自 ./iteration/20260615-2/REQUIREMENT.md
+ 修改.../config目录的作用
    + ../config/config.json: 用于主应用的config，需要在打包时打包进应用目录或APP（区分Mac和Linux）
    + ../config剩余文件或目录（skills）、SOUL.md、USER.md等（除config.json）用于创建新Agent时复制到工作目录
+ 修改打包逻辑和新建Agent复制目录逻辑

### 创建任务全链路router_disable
> 新增自 ./iteration/20260524-3/REQUIREMENT.md
+ 技术规范的指导下收口：
    + HTTP POST `/api/cron/create?agentId=xxx`：创建备忘录元数据时增加 router_disable 参数（boolean，默认true关闭）
    + HTTP POST `/api/cron/detail/metadata?agentId=xxx`：查询元数据时返回 router_disable 参数
    + 映射规则：开启SWARM时 router_disable=false，关闭SWARM时 router_disable=true
    + 周期任务、一次性任务均需要全链路保证 router_disable 一致
    + 禁止在执行链路中丢失该字段，禁止回退为 Agent config.json 中的 router_disable

### 插件配置router_disable
> 新增自 ./iteration/20260524-4/REQUIREMENT.md
+ 技术规范的指导下收口：
    + HTTP `/api/plugins/config` 增加 router_disable 参数传入，默认 true
    + HTTP `/api/plugins/meta` 中返回每个插件的 router_disable 参数
    + 在对应存储 connect_meta 中增加字段
    + 指定插件 add-request 转换为备忘录明细时传递 router_disable 参数

### 蜂群开关参数名变更
> 新增自 ./iteration/20260524-5/REQUIREMENT.md
+ 技术规范的指导下收口：
    + 蜂群开关参数 swarm 改为 router_disable，类型不变，语意相反（router_disable=true 表示关闭）
    + HTTP /api/edit 中 swarm 改为 router_disable，默认为 true

### 启动自动补齐Agent配置
> 新增自 ./iteration/20260525-1/REQUIREMENT.md
+ 技术规范的指导下收口：
    + 启动时 --agent-dir 指向空目录时自动补齐 DEF_AGENT，复制 --default-dir 目录内容

### Token消费记录
> 新增自 ./iteration/20260527-1/REQUIREMENT.md
+ 技术规范的指导下收口：
    + 新增命令 token，记录 Token 消费明细保存至数据库
    + 新增 /api/consume 接口获取指定 Agent 在指定时间范围内的 Token 消费数据

> 新增自 ./iteration/20260614-3/REQUIREMENT.md
+ 技术收口：
    + 增加token get命令，获取最近N条，或指定时间段的token数据
        + 读取的数据为token在本地数据库存储的用量数据
        + 数据查询方式需要与接口/api/consume相同
    + 案例
    ``` 使用integration代理proxy，查询最新500条
    integration token --n 500
    ```
    ``` 使用integration代理proxy，查询2026-06-14 12:00:00至2026-06-14 14:00:00最新500条
    integration token --n 500 --start "2026-06-14 12:00:00" --close "2026-06-14 14:00:00"
    ```
    + 不能破坏现有token命令，如下
    ``` 使用integration代理proxy
    integration token
    integration token --provider deepseek
    integration token --agentId demo-agent --model deepseek-chat --function cli/get --thinking 10 --input 20 --total 30 --cache 5
    ```
    + 增加--help
    ```
    integration token get --help
    ```
    + Proxy需求：../proxy/iteration/20260614-2/REQUIREMENT.md

### 统一变量名和转发报文
> 新增自 ./iteration/20260528-1/REQUIREMENT.md
+ 技术规范的指导下收口：
    + 统一变量名（蜂群开关→router_disable，思考模式→thinking，HTML开关→html，模型→model）
    + 转发 /v1/chat/completions 和 /cli/get 时按统一报文格式
    + 唯一端口，所有功能均使用 --port 指定端口，默认 8080

### 获取DeviceId
> 新增自 ./iteration/20260530-1/REQUIREMENT.md
+ 技术规范的指导下收口：
    + 新增 /api/deviceId 获取 DeviceID

### /install_app 增加python3检测
> 新增自 ./iteration/20260530-2/REQUIREMENT.md
+ 技术规范的指导下收口：
    + 修改 /install_app 接口，如果 python3 没有安装则在返回中添加 "python3"

### SSE响应完成系统通知
> 新增自 ./iteration/20260619-1/REQUIREMENT.md
+ 任一（普通对话、备忘录任务）SSE响应整体完成（包含完成、异常）后如果是MAC OS系统需要调用系统通知
+ 通知内容"DeepRight通知"
+ 通知图标：应用程序的图标

### 撰写手册
+ 编写USER_GUIDE.md

### 其他要求
+ REQUIREMENT.md为需求文档，禁止编写
+ 相关迭代：iteration/日期/REQUIREMENT.md
> 合并截止：./iteration/20260619-1/REQUIREMENT.md，下次合并从此之后的新迭代开始
