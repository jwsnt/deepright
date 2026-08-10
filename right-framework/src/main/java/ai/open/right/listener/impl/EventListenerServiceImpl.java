package ai.open.right.listener.impl;

import ai.open.right.WorkflowException;
import ai.open.right.listener.Event;
import ai.open.right.listener.EventListener;
import ai.open.right.listener.EventListenerService;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.builder.ToStringBuilder;
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
public class EventListenerServiceImpl implements EventListenerService {

    public static final String NAME = "eventListenerServiceImpl";

    protected List<EventListener> eventListener;

    @Override
    public void listen(Event event) throws Exception {
        if (CollectionUtils.isEmpty(this.eventListener)) {
            return;
        }
        try {
            if (log.isDebugEnabled()) {
                log.debug("Event listen={}", event);
            }
            Assert.notNull(event.getData(), "Event listen body can not be empty");
            event = event.init();
            if (!CollectionUtils.isEmpty(this.eventListener)) {
                for (EventListener each : this.eventListener) {
                    try {
                        each.listen(event);
                    } catch (Exception e) {
                        if (log.isWarnEnabled()) {
                            log.warn(e.getMessage(), e);
                        }
                    }
                }
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
    }

    @ConditionalOnProperty(name = "event.listener.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired(required = false)
        protected List<EventListener> eventListener;

        @Bean(name = EventListenerServiceImpl.NAME)
        @ConditionalOnMissingBean(value = EventListenerService.class)
        public EventListenerService eventListenerService() throws Exception {
            EventListenerServiceImpl eventListenerService = new EventListenerServiceImpl();
            BeanUtils.copyProperties(this, eventListenerService);
            log.info("EventListenerServiceImpl inited={}", ToStringBuilder.reflectionToString(eventListenerService));
            return eventListenerService;
        }
    }
}