package ai.open.right.listener.impl;

import ai.open.right.listener.Event;
import ai.open.right.listener.EventListener;
import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.stereotype.Component;

@Slf4j
@Component(LoggerEventListener.NAME)
@ConditionalOnProperty(name = "event.listener.logger.enable", havingValue = "true", matchIfMissing = true)
public class LoggerEventListener implements EventListener {

    public static final String NAME = "logger_event_listener";

    @Override
    public void listen(Event event) throws Exception{
        if (log.isDebugEnabled()) {
            log.debug("Listener event={}-{}", event.getType(), event.getData());
        }
    }
}
