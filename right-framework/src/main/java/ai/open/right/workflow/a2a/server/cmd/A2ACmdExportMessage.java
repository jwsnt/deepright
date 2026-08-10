package ai.open.right.workflow.a2a.server.cmd;

import ai.open.right.context.UserContext;
import ai.open.right.integration.RightConfig;
import ai.open.right.integration.RightService;
import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.a2a.A2ARequest;
import ai.open.right.workflow.a2a.A2AResponse;
import ai.open.right.workflow.a2a.protocol.*;
import ai.open.right.workflow.a2a.server.A2ACmdExportService;
import ai.open.right.workflow.flow.llm.Segment;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.NotifierWriteBack;
import ai.open.right.workflow.notify.NotifierWriteBase;
import ai.open.right.workflow.sync.SyncCallable;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.util.Assert;

import java.util.*;

@Slf4j
@Setter
@Getter
abstract public class A2ACmdExportMessage implements A2ACmdExportService {

    protected RightService rightService;

    protected Integer timeout4Llm;

    // 构建异步回调
    abstract public SyncCallable buildSyncCallable(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception;

    @Override
    public void cmd(A2ARequest a2aRequest) throws Exception {
        MessageRequest messageRequest = this.buildMessageRequest(a2aRequest);
        // 构建Segment回调
        SyncCallable syncCallable = this.buildSyncCallable(a2aRequest, messageRequest);
        Assert.notNull(this.rightService, "The right service can not be empty, please config `integration.enable`");
        this.rightService.get(this.buildRightConfig(a2aRequest, messageRequest, syncCallable));
    }

    protected RightConfig buildRightConfig(A2ARequest a2aRequest, MessageRequest messageRequest, SyncCallable syncCallable) throws Exception {
        String[] pair = this.buildDimension(a2aRequest, messageRequest);
        return RightConfig.builder()
                .notifierWriteBack(this.buildNotifierWriteBack(a2aRequest, messageRequest))
                .mediaContext(this.buildMediaContext(a2aRequest, messageRequest))
                .conversation(this.buildConversation(a2aRequest, messageRequest))
                .userContext(this.buildUserContext(a2aRequest, messageRequest))
                .histories(this.buildHistories(a2aRequest, messageRequest))
                .metadata(this.buildMetadata(a2aRequest, messageRequest))
                .query(this.buildTextQuery(a2aRequest, messageRequest))
                .trace(this.buildTrace(a2aRequest, messageRequest))
                .chat(this.buildChat(a2aRequest, messageRequest))
                .syncCallable(syncCallable)
                .timeout(this.timeout4Llm)
                .workflow(pair[1]).biz(pair[0]).build();
    }

    protected NotifierWriteBack buildNotifierWriteBack(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
        return new A2ANotifierWriteBack(a2aRequest);
    }

    // 构建MediaContext
    protected List<MediaContext> buildMediaContext(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
        List<MediaContext> mediaContexts = new ArrayList<MediaContext>();
        // 从Part中提取FilePart
        for (Part part : messageRequest.getMessage().getParts()) {
            if (part.isKind(Part.FILE_KIND)) {
                MediaContext mediaContext = new MediaContext();
                mediaContext.setType(part.getFile().getMimeType());
                mediaContext.setData(part.getFile().getContent());
                mediaContexts.add(mediaContext);
            }
        }
        return mediaContexts;
    }

    // 构建Meta，合并Header + Json
    protected Map<String, Object> buildMetadata(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
        // Message覆盖Header
        messageRequest.putIfAbsent(a2aRequest.getHeaders()).putIfAbsent(messageRequest.getMessage().getMetadata());
        return messageRequest.getMetadata();
    }

    // 构建历史记录
    protected List<History> buildHistories(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
        List<History> histories = new ArrayList<History>();
        for (Part part : messageRequest.getMessage().getParts()) {
            History history = new History();
            // 标记Role=User
            history.setUser();
            switch (part.getKind()) {
                case Part.DATA_KIND:
                    // 读取Data Part
                    history.setContent(this.buildDataPart(part));
                    break;
                case Part.FILE_KIND:
                    // 读取File Part
                    history.setContent(this.buildFilePart(part));
                    break;
                case Part.TEXT_KIND:
                    // 读取Text Part
                    history.setContent(this.buildTextPart(part));
                    break;
                default:
                    if (log.isWarnEnabled()) {
                        // 无法解析的Kind
                        log.warn("A2A parse error={}", part.getKind());
                    }
            }
            histories.add(history);
        }
        return histories;
    }

    // 构建UserContext
    protected UserContext buildUserContext(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
        // Meta已经经过合并
        return UserContext.setDefault(JsonUtils.transfer(messageRequest.getMetadata(), UserContext.class));
    }

    // 构建会话ID
    protected String buildConversation(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
        // 使用Message.ID
        return messageRequest.getMessage().getMessageId();
    }

    // 构建维度
    protected String[] buildDimension(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
        String[] uri = a2aRequest.getPath().split("/");
        if (log.isDebugEnabled()) {
            log.debug("A2A dimension={}", Arrays.toString(uri));
        }
        return SplitUtils.split(uri[1]);
    }

    // 构建用户查询
    protected String buildTextQuery(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
        return JsonUtils.write(messageRequest);
    }

    protected String buildTrace(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
        // 先取Header后取Metadata
        return StringUtils.defaultString(a2aRequest.getTrace(), String.class.cast(messageRequest.getMetadata().get("trace")));
    }

    protected String buildChat(A2ARequest a2aRequest, MessageRequest messageRequest) throws Exception {
        return String.valueOf(a2aRequest.getId());
    }

    // 转换为MessageRequest
    protected MessageRequest buildMessageRequest(A2ARequest a2aRequest) throws Exception {
        Map<String, Object> params = Map.class.cast(a2aRequest.getContent().get("params"));
        Assert.notEmpty(params, "A2A message's params can not be empty");
        return JsonUtils.transfer(params, MessageRequest.class);
    }

    // 解析Data Part
    protected String buildDataPart(Part part) throws Exception {
        return JsonUtils.write(part.getData());
    }

    // 解析File Part
    protected String buildFilePart(Part part) throws Exception {
        return part.getFile().getContent();
    }

    // 解析Text Part
    protected String buildTextPart(Part part) throws Exception {
        return part.getText();
    }

    @Getter
    // 处理A2A Source
    public static class A2ANotifierWriteBack extends NotifierWriteBase {

        protected final A2ARequest request;

        public A2ANotifierWriteBack(A2ARequest request) {
            this.request = request;
        }

        @Override
        public void writeSource(Segment segment) throws Exception {
            // 如果不为JSON则转为Simple Data Part
            if (!JsonUtils.like(segment.getContent())) {
                Task task = this.buildTask(segment, this.buildArtifact(segment, this.buildPart(segment)));
                this.request.writeStream(this.buildA2Response(segment, task));
            } else {
                // 直接使用A2ACmdResponse
                this.request.writeStream(JsonUtils.transfer(segment.getContent(), A2ACmdResponse.class));
            }
        }

        @Override
        public void writeBack(Segment segment) throws Exception {
            this.writeSource(segment);
        }

        protected A2AResponse buildA2Response(Segment segment, Task task) throws Exception {
            return A2ACmdResponse.builder()
                    .id(this.request.getId())
                    // 默认TEXT文本为False（不关闭流）
                    .finished(false)
                    .result(task)
                    .build();
        }

        protected Task buildTask(Segment segment, Artifact artifact) throws Exception {
            return Task.builder()
                    .timestamp(A2ACmdCallableOnce.FORMATTER.format(System.currentTimeMillis()))
                    .contextId(String.valueOf(segment.getIndex()))
                    .status(TaskStatus.builder()
                            .state(TaskStatus.STATUS_COMPLETED)
                            .build())
                    .metadata(segment.getMetadata())
                    .artifacts(List.of(artifact))
                    .id(this.request.getId())
                    .build()
                    .reset();
        }

        protected Artifact buildArtifact(Segment segment, Part part) throws Exception {
            return Artifact.builder()
                    .artifactId(UUID.randomUUID().toString())
                    .parts(List.of(part))
                    .build()
                    .reset();
        }

        protected Part buildPart(Segment segment) throws Exception {
            return Part.builder()
                    .metadata(segment.getMetadata())
                    .text(segment.getContent())
                    .build();
        }
    }

    @Setter
    @Getter
    public static class A2ACmdInitConfig {

        @Autowired(required = false)
        protected RightService rightService;

        // 消息超时
        @Value("${a2a.message.timeout:1800000}")
        protected Integer timeout4Llm;
    }
}
