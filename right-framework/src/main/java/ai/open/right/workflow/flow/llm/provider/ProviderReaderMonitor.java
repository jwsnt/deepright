package ai.open.right.workflow.flow.llm.provider;

import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

@ConditionalOnProperty(name = "monitor.reader.enable", havingValue = "true", matchIfMissing = true)
@Slf4j
@Service
public class ProviderReaderMonitor {

    @Scheduled(initialDelayString = "${monitor.reader.initialDelay:30000}", fixedRateString = "${monitor.reader.fixedRate:30000}")
    public String monitor() throws Exception {
        StringBuffer content = new StringBuffer("The reader status=").append(ProviderReaderCallback.RUNNER_COUNTER.get());
        if (log.isInfoEnabled()) {
            log.info(content.toString());
        }
        return content.toString();
    }
}
