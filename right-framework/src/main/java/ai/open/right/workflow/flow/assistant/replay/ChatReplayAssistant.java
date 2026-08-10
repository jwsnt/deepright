package ai.open.right.workflow.flow.assistant.replay;

import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.DefaultAssistant;
import ai.open.right.workflow.flow.config.WorkflowConfig;
import ai.open.right.workflow.flow.track.TrackChatBody;
import ai.open.right.workflow.flow.track.TrackChatService;
import com.google.common.collect.ImmutableMap;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.CollectionUtils;

import java.util.List;

@Setter
@Getter
@Slf4j
// 指定Chat的任务响应回放
public class ChatReplayAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-task-replay";

    protected TrackChatService trackChatService;

    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        List<TrackChatBody> chatBodies = this.trackChatService.restore(workTask);
        if (log.isDebugEnabled()) {
            log.debug("Track chat replay={}", CollectionUtils.isEmpty(chatBodies) ? 0 : chatBodies.size());
        }
        this.chainOr2Endpoint(workflowConfig, workTask, this.buildContent(chatBodies, workTask));
    }

    protected String buildContent(List<TrackChatBody> chatBodies, WorkflowTask workTask) throws Exception {
        if (!StringUtils.isEmpty(workTask.getQuery())) {
            if (log.isDebugEnabled()) {
                log.debug("Chat replay will use query and chat={}", workTask.getQuery());
            }
            return JsonUtils.write(ImmutableMap.of("query", workTask.getQuery(), "chat", chatBodies));
        } else {
            if (log.isDebugEnabled()) {
                log.debug("Chat replay will just use chat");
            }
            return JsonUtils.write(chatBodies);
        }
    }


    @ConditionalOnProperty(name = "replay.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired
        protected TrackChatService trackChatService;

        @Bean(ChatReplayAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = ChatReplayAssistant.WORKFLOW_NAME)
        public ChatReplayAssistant chatReplayAssistant() throws Exception {
            ChatReplayAssistant chatReplayAssistant = new ChatReplayAssistant();
            BeanUtils.copyProperties(this, chatReplayAssistant);
            log.info("ChatReplayAssistant inited");
            return chatReplayAssistant;
        }
    }
}
