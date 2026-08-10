### 验证测试
+ 每一条都必须通过
###### 代码检查
+ 创建CDP服务必须静默使用--stealth
``` 实际调用obscura --stealth
./browser instance create --agentId agent-a --chatId ctrip-home
```
+ Playwright命令自动添加符合如下Chrome特征的navigator.platform和userAgent
```
Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36
```
###### 运行检查
+ 使用browser启动CDP
+ 验证：
    + navigator.platform和userAgent一致且符合Chrome特征
    + navigator.webdriver为undefined
    + navigator.userAgentData 使用MAC指纹链
    + maxTouchPoints Mac触控板应有值，大于0
    + screen 使用分辨率2560x1440
    + 时区与语言 使用当前系统一致
    + WebGL渲染器 模拟GPU信息
    + screen分辨率 2560x1440
