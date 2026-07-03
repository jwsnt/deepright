### 需求
+ 随着需求迭代，需求汇总文档需要持续增加/合并/修正新内容，并标注因为哪个子需求迭代引起的变更
+ 需求迭代的目录格式为时间日期+索引，时间越接近当前时间说明需求越新，同时间索引值越大说明需求越新
+ 需求迭代可能是新增功能、修改功能、调整逻辑等，需要在需求汇总文档中记录变更内容，以及对应引用需求迭代文件的相对路径

#### 要求
+ 保持原需求汇总文件的格式，增加/合并/修正内容先检查原文档中是否有对应分类
    + 如果有，归入对应分类
    + 如果无，单独新开分类，并参考原分类格（注意：所有新增分类必须位于`编写代码`模块之前）
+ 每个模块的需求汇总都会记录上次最后更新的时间/位置，从该位置继续而不要重头开始
+ 同步更新同目录下的USER_GUIDE.md
+ UPDATE.md本身禁止更新

#### 修正
+ 当前目录和子孙目录下名称为REQUIREMENT.md的文件中如果标记了`同步代码：../integration/REQUIREMENT.md`, 那么就需要检查这个引用的相对路径是否正确
    + 文件位于当前目录下的`integration/REQUIREMENT.md`
    + 相对位置引用不正确就修正

#### Knowledge模块
+ 需求汇总路径：../knowledge/REQUIREMENT.md
+ 需求迭代路径：../knowledge/iteration

#### Agent模块
+ 需求汇总路径：../agent/REQUIREMENT.md
+ 需求迭代路径：../agent/iteration

#### CLI GET模块
+ 需求汇总路径：../cli-get/REQUIREMENT.md
+ 需求迭代路径：../cli-get/iteration

#### CRON模块
+ 需求汇总路径：../cron/REQUIREMENT.md
+ 需求迭代路径：../cron/iteration

#### SKILLS模块
+ 需求汇总路径：../skills/REQUIREMENT.md
+ 需求迭代路径：../skills/iteration

#### STATIC模块
+ 需求汇总路径：../static/REQUIREMENT.md

#### CONNECT模块
+ 该模块比较特殊，需要同时汇总插件子需求
+ 需求汇总路径：../connect/REQUIREMENT.md
+ 需求迭代路径：../connect/iteration
    + 飞书插件：
        + 需求汇总路径：../connect/feishu/REQUIREMENT.md
        + 需求迭代路径：../connect/feishu/iteration
    + 邮件插件：
        + 需求汇总路径：../connect/email/REQUIREMENT.md
        + 需求迭代路径：../connect/email/iteration

#### PROXY模块
+ 需求汇总路径：../proxy/REQUIREMENT.md
+ 需求迭代路径：../proxy/iteration

#### SITE模块
+ 需求汇总路径：../site/REQUIREMENT.md
+ 需求迭代路径：../site/iteration

#### INTEGRATION模块
+ 需求汇总路径：../integration/REQUIREMENT.md
+ 需求迭代路径：../integration/iteration

#### 路径修正
- 验证所有REQUIREMENT.md中`integration/REQUIREMENT.md`相对路径引用，全部正确无需修正
