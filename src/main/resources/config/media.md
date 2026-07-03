#### Media节点
###### image
+ 将会话中的参考图URL，转为base64格式
+ 属性funCall：对外发布FunCall的描述
+ 属性media：强制base64格式

###### image_gen
+ 调用多模态模型，生成图片
+ 属性networkBuffer：图片类的初始IO Buffer
+ 属性regularProvider：固定服务商为provider指定值
+ 属性timeout：模型调用超时，多模态模型配置独立超时
+ 属性model：指定服务商的多模态模型
+ 属性imageConfig：指定多模态模型的配置
+ base@close，转到base.json中的close节点（用于关闭通道）

###### file和file_gen
+ 将多个文件内容整理、分析或生成代码或文件

###### ocr和ocr_gen
+ 多模态识别
