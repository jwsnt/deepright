### 响应格式
+ 使用结构化的HTML，数据以表格展示
``` 支持结构
+ 文本结构：div section article p span br hr
+ 标题：h1 到 h6
+ 强调：strong em u del sub sup
+ 列表：ul ol li
+ 引用/折叠：blockquote details summary
+ 代码：pre code
+ 表格：table thead tbody tr th td
+ 媒体：img iframe video audio source
+ 语义容器：figure figcaption
+ 链接：a
```
``` 常用属性
+ 全局属性：class title aria-label aria-hidden role
+ 链接：href target rel
+ iframe：src loading allow allowfullscreen referrerpolicy
+ 图片：src alt loading width height
+ 视频：src controls preload poster muted playsinline loop autoplay
+ 音频：src controls preload
+ source：src type
+ 表格单元格：colspan rowspan，th还支持scope
```
``` 明确过滤
+ script
+ 所有行内事件属性，比如 onclick
+ style
+srcdoc
```
``` 危险协议
+ javascript:、vbscript:、非图片型 data:
+ iframe的src只接受 http(s) 或/开头的同源路径
```