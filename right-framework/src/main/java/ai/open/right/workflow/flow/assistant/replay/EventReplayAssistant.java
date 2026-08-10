package ai.open.right.workflow.flow.assistant.replay;

import ai.open.right.listener.Event;
import ai.open.right.listener.EventReplay;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.assistant.DefaultAssistant;
import ai.open.right.workflow.flow.config.WorkflowConfig;
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
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.List;

@Setter
@Getter
@Slf4j
// 内部事件回放
public class EventReplayAssistant extends DefaultAssistant {

    public static final String WORKFLOW_NAME = "def-event-replay";

    protected EventReplay eventReplay;

    public void execute(WorkflowConfig workflowConfig, WorkflowTask workTask) throws Exception {
        // 仅当注册EventRelay后可用
        Assert.notNull(this.eventReplay, "The Event Replay can not be empty. Please check if the implementation class of EventReplay has been configured");
        List<Event> events = this.eventReplay.replay(workTask);
        if (log.isDebugEnabled()) {
            log.debug("Event Replay={}", CollectionUtils.isEmpty(events) ? 0 : events.size());
        }
        this.chainOr2Endpoint(workflowConfig, workTask, this.buildContent(events, workTask));
    }

    protected String buildContent(List<Event> events, WorkflowTask workTask) throws Exception {
        if (!StringUtils.isEmpty(workTask.getQuery())) {
            if (log.isDebugEnabled()) {
                log.debug("Event replay will use query and events={}", workTask.getQuery());
            }
            return JsonUtils.write(ImmutableMap.of("query", workTask.getQuery(), "events", events));
        } else {
            if (log.isDebugEnabled()) {
                log.debug("Event replay will just use events");
            }
            return JsonUtils.write(events);
        }
    }

    @ConditionalOnProperty(name = "replay.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends DefInitConfig {

        @Autowired(required = false)
        protected EventReplay eventReplay;

        @Bean(EventReplayAssistant.WORKFLOW_NAME)
        @ConditionalOnMissingBean(name = EventReplayAssistant.WORKFLOW_NAME)
        public EventReplayAssistant eventReplayAssistant() throws Exception {
            EventReplayAssistant eventReplayAssistant = new EventReplayAssistant();
            BeanUtils.copyProperties(this, eventReplayAssistant);
            log.info("EventReplayAssistant inited");
            return eventReplayAssistant;
        }
    }
}
