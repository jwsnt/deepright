---
name: __internal_model
description: 管理文字转语音、音频文字提取、图片主体提取、人物视频对口型和视频主体提取任务
---

### 通用
+ 使用`#app api ...`，终端等价命令为`integration api ...`。默认Agent为`#agentId`。路径必须是Agent工作目录内的相对路径
+ 任务状态`queued`为等待，`running`为执行中，创建只入队。用`list`查询状态，用`log`查询日志、进度和错误
+ 每类支持`check|list|create|cancel|restart|delete|log`
+ 先执行`#app api <功能> --help`查看完整参数和案例
+ `restart`和`delete`仅限`failed`或`cancelled`
+ `cancel`仅限`queued`或`running`
+ 所有任务共享执行和等待队列
+ 重试清空旧日志并重分配输出
+ 删除不删除文件
```
#app api <功能> check
#app api <功能> list --agentId "#agentId" --status queued --page 1
#app api <功能> log --agentId "#agentId" --id 12
```

### 五类创建命令
###### VoxCPM 文字转语音
+ `voxcpm create --agentId ID --textPath PATH --outputName NAME.wav`，`--textPath`可重复，`--referenceAudioPath`、`--scenario`、`--control`可提供一次或逐文本提供
+ 场景为 `balanced|quality|fast|clean|warm_narration|lively`
```
#app api voxcpm create --agentId "#agentId" --textPath "scripts/intro.txt" --outputName "intro.wav" --scenario warm_narration
```
###### Whisper 提取音频文字
+ `whisper create --agentId ID --path AUDIO`，`--path`可重复
+ 场景为`chinese_meeting|chinese_accurate|realtime|batch|cpu|mixed_technical`
```
#app api whisper create --agentId "#agentId" --path "audios/meeting.mp3" --scenario chinese_meeting
```
###### Rembg 提取图片主体
+ `rembg create --agentId ID --path IMAGE`，`--path`可重复，可选`--model`和`--alpha-matting`，模型为`u2net|u2net_human_seg|u2netp|u2net_cloth_seg|silueta|isnet-general-use|isnet-anime`
```
#app api rembg create --agentId "#agentId" --path "images/product.jpg" --model u2net --alpha-matting
```
###### Wav2Lip 人物视频对口型
+ `wav2lip create --agentId ID --videoPath VIDEO --audioPath AUDIO`，视频和音频数量相同，按出现顺序配对，最多64组
```
#app api wav2lip create --agentId "#agentId" --videoPath "videos/person.mp4" --audioPath "audios/line.wav"
```
###### RVM 视频提取主体
+ `rvm create --agentId ID --path VIDEO`。`--path`可重复，最多64个，`--scenario standard|quality|fast`可提供一次或逐视频提供
```
#app api rvm create --agentId "#agentId" --path "videos/product.mp4" --scenario quality
```
