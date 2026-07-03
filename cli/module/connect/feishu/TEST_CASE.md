### 必过测试
+ 飞书纯文字消息，推送add-request，任务明细执行返回文字消息，回调send发送飞书
+ 飞书纯文字消息，推送add-request，任务明细执行返回图片或文件消息，回调send发送飞书
+ 飞书纯文字消息，推送add-request，任务明细执行返回文字和图片或文件消息，回调send发送飞书
+ 飞书纯图片或文件消息，不推送add-request，补充图片消息，不推送add-request，直到消息过期
+ 飞书纯图片或文件消息，不推送add-request，补充文字消息，推送add-request，任务明细执行返回文字和图片或文件消息，回调send发送飞书
+ 飞书纯文字消息，推送add-request，任务明细执行返回PDF/DOC/XLS/PPT/MP4/OPUS等非图片附件，回调send发送飞书文件消息，文件上传类型与消息类型保持一致
